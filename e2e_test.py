#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
日志服务端到端测试脚本

功能:
1. 上传测试 zip 文件
2. 查询文件列表
3. 下载文件并验证内容一致性
4. 测试错误场景
"""

import os
import sys
import hashlib
import json
import tempfile
import zipfile

import requests

# 服务配置
BASE_URL = "http://localhost:8080"
UPLOAD_URL = f"{BASE_URL}/upload"
QUERY_URL = f"{BASE_URL}/query"
DOWNLOAD_URL = f"{BASE_URL}/download"

# 颜色输出
GREEN = '\033[92m'
RED = '\033[91m'
YELLOW = '\033[93m'
RESET = '\033[0m'


def log_success(msg):
    """打印成功日志"""
    print(f"{GREEN}[PASS]{RESET} {msg}")


def log_error(msg):
    """打印错误日志"""
    print(f"{RED}[FAIL]{RESET} {msg}")


def log_info(msg):
    """打印信息日志"""
    print(f"{YELLOW}[INFO]{RESET} {msg}")


def calc_file_hash(filepath):
    """计算文件 MD5 哈希"""
    hash_md5 = hashlib.md5()
    with open(filepath, "rb") as f:
        for chunk in iter(lambda: f.read(4096), b""):
            hash_md5.update(chunk)
    return hash_md5.hexdigest()


def test_upload_valid_file():
    """测试上传有效文件"""
    log_info("测试1: 上传有效 zip 文件")
    
    # 创建临时测试文件
    with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as tmp:
        tmp_path = tmp.name
    
    # 创建 zip 文件
    test_filename = "myapp_abc123_20240101120000.zip"
    with zipfile.ZipFile(tmp_path, 'w') as zf:
        zf.writestr("test.log", "This is a test log content")
    
    # 上传文件
    with open(tmp_path, 'rb') as f:
        files = {'file': (test_filename, f, 'application/zip')}
        response = requests.post(UPLOAD_URL, files=files)
    
    os.unlink(tmp_path)
    
    if response.status_code != 200:
        log_error(f"上传失败, 状态码: {response.status_code}, 响应: {response.text}")
        return False
    
    data = response.json()
    if data.get('code') != 0:
        log_error(f"上传失败: {data.get('message')}")
        return False
    
    log_success(f"上传成功: {data.get('data')}")
    return True


def test_upload_invalid_filename():
    """测试上传无效文件名"""
    log_info("测试2: 上传无效文件名")
    
    # 创建临时文件
    with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as tmp:
        tmp_path = tmp.name
    
    with zipfile.ZipFile(tmp_path, 'w') as zf:
        zf.writestr("test.log", "content")
    
    # 使用无效文件名上传
    with open(tmp_path, 'rb') as f:
        files = {'file': ('invalid.zip', f, 'application/zip')}
        response = requests.post(UPLOAD_URL, files=files)
    
    os.unlink(tmp_path)
    
    if response.status_code != 400:
        log_error(f"期望状态码 400, 实际: {response.status_code}")
        return False
    
    log_success("无效文件名被正确拒绝")
    return True


def test_query_files():
    """测试查询文件"""
    log_info("测试3: 查询文件列表")
    
    # 按名称查询
    response = requests.get(QUERY_URL, params={'name': 'myapp'})
    
    if response.status_code != 200:
        log_error(f"查询失败, 状态码: {response.status_code}")
        return False
    
    data = response.json()
    if data.get('code') != 0:
        log_error(f"查询失败: {data.get('message')}")
        return False
    
    files = data.get('data', {}).get('files', [])
    count = data.get('data', {}).get('count', 0)
    
    log_success(f"查询成功, 找到 {count} 个文件")
    for f in files:
        log_info(f"  - {f['name']} ({f['size']} bytes)")
    
    return True


def test_download_file():
    """测试下载文件并验证内容"""
    log_info("测试4: 下载文件并验证内容")
    
    filename = "myapp_abc123_20240101120000.zip"
    
    # 下载文件
    response = requests.get(DOWNLOAD_URL, params={'filename': filename})
    
    if response.status_code != 200:
        log_error(f"下载失败, 状态码: {response.status_code}")
        return False
    
    # 保存下载的文件
    download_path = f"/tmp/downloaded_{filename}"
    with open(download_path, 'wb') as f:
        f.write(response.content)
    
    # 验证 zip 内容
    try:
        with zipfile.ZipFile(download_path, 'r') as zf:
            namelist = zf.namelist()
            log_info(f"下载的 zip 包含文件: {namelist}")
            
            # 读取内容
            with zf.open('test.log') as log_file:
                content = log_file.read().decode('utf-8')
                if content == "This is a test log content":
                    log_success("下载文件内容验证通过")
                    os.unlink(download_path)
                    return True
                else:
                    log_error(f"内容不匹配: {content}")
                    return False
    except zipfile.BadZipFile:
        log_error("下载的文件不是有效的 zip")
        return False


def test_download_not_found():
    """测试下载不存在的文件"""
    log_info("测试5: 下载不存在的文件")
    
    response = requests.get(
        DOWNLOAD_URL, 
        params={'filename': 'nonexistent_abc123_20240101120000.zip'}
    )
    
    if response.status_code != 404:
        log_error(f"期望状态码 404, 实际: {response.status_code}")
        return False
    
    log_success("不存在的文件返回 404")
    return True


def test_query_with_commit():
    """测试按 commit ID 查询"""
    log_info("测试6: 按 commit ID 查询")
    
    response = requests.get(
        QUERY_URL, 
        params={'name': 'myapp', 'commit': 'abc123'}
    )
    
    if response.status_code != 200:
        log_error(f"查询失败, 状态码: {response.status_code}")
        return False
    
    data = response.json()
    count = data.get('data', {}).get('count', 0)
    
    log_success(f"按 commit 查询成功, 找到 {count} 个文件")
    return True


def test_concurrent_upload():
    """测试并发上传"""
    log_info("测试7: 并发上传测试")
    
    import concurrent.futures
    
    def upload_file(idx):
        with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as tmp:
            tmp_path = tmp.name
        
        filename = f"concurrent{idx}-app_commit{idx}-id_2024010112000{idx}.zip"
        with zipfile.ZipFile(tmp_path, 'w') as zf:
            zf.writestr(f"log{idx}.txt", f"Content {idx}")
        
        with open(tmp_path, 'rb') as f:
            files = {'file': (filename, f, 'application/zip')}
            response = requests.post(UPLOAD_URL, files=files)
        
        os.unlink(tmp_path)
        return response.status_code == 200
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        futures = [executor.submit(upload_file, i) for i in range(5)]
        results = [f.result() for f in concurrent.futures.as_completed(futures)]
    
    success_count = sum(results)
    if success_count == 5:
        log_success(f"并发上传成功: {success_count}/5")
        return True
    else:
        log_error(f"并发上传失败: {success_count}/5")
        return False


def main():
    """主函数"""
    print("=" * 60)
    print("日志服务端到端测试")
    print("=" * 60)
    
    # 检查服务是否可用
    try:
        requests.get(f"{BASE_URL}/query", timeout=2)
    except requests.exceptions.ConnectionError:
        log_error("无法连接到日志服务，请确保服务已启动")
        sys.exit(1)
    
    log_info("服务连接正常")
    print()
    
    # 执行所有测试
    tests = [
        test_upload_valid_file,
        test_upload_invalid_filename,
        test_query_files,
        test_download_file,
        test_download_not_found,
        test_query_with_commit,
        test_concurrent_upload,
    ]
    
    results = []
    for test in tests:
        try:
            result = test()
            results.append(result)
        except Exception as e:
            log_error(f"测试异常: {e}")
            results.append(False)
        print()
    
    # 打印测试结果汇总
    print("=" * 60)
    print("测试汇总")
    print("=" * 60)
    passed = sum(results)
    total = len(results)
    
    if passed == total:
        print(f"{GREEN}全部通过: {passed}/{total}{RESET}")
        return 0
    else:
        print(f"{RED}部分失败: {passed}/{total}{RESET}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
