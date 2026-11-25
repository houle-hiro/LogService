#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
文件中转服务端到端测试

环境变量:
  BASE_URL    服务地址，默认 http://localhost:8080
  AUTH_TOKEN  访问口令，与服务端一致
"""

import hashlib
import os
import sys
import uuid

import requests

BASE_URL = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
TOKEN = os.environ.get("AUTH_TOKEN", "")
USER = "e2e-" + uuid.uuid4().hex[:8]
OTHER = "e2e-other-" + uuid.uuid4().hex[:8]

GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
RESET = "\033[0m"

passed = 0
failed = 0


def check(name, cond, detail=""):
    global passed, failed
    if cond:
        passed += 1
        print(f"{GREEN}[PASS]{RESET} {name}")
    else:
        failed += 1
        print(f"{RED}[FAIL]{RESET} {name} {detail}")


def md5(data):
    return hashlib.md5(data).hexdigest()


def main():
    if not TOKEN:
        print(f"{YELLOW}[WARN]{RESET} 未设置 AUTH_TOKEN，若服务端启用鉴权将全部失败")

    s = requests.Session()
    auth = {"X-Token": TOKEN}

    # 1. 健康检查免登录
    r = s.get(f"{BASE_URL}/healthz", timeout=10)
    check("健康检查 200", r.status_code == 200, f"实际 {r.status_code}")

    # 2. 未认证访问被拒
    r = requests.get(f"{BASE_URL}/query?username={USER}", timeout=10)
    check("未带 token 查询返回 401", r.status_code == 401, f"实际 {r.status_code}")

    # 3. 登录页可访问
    r = requests.get(f"{BASE_URL}/login", timeout=10)
    check("登录页 200", r.status_code == 200, f"实际 {r.status_code}")

    # 4. 错误口令被拒
    r = requests.post(
        f"{BASE_URL}/login",
        data={"username": USER, "token": "wrong-token", "next": "/"},
        allow_redirects=False,
        timeout=10,
    )
    check("错误口令返回 401", r.status_code == 401, f"实际 {r.status_code}")

    # 5. 正确登录
    r = s.post(
        f"{BASE_URL}/login",
        data={"username": USER, "token": TOKEN, "next": "/"},
        allow_redirects=False,
        timeout=10,
    )
    check("登录成功 302", r.status_code == 302, f"实际 {r.status_code}")
    check("登录下发会话 cookie", "fs_session" in s.cookies, "无 fs_session")

    # 6. 首页可渲染
    r = s.get(f"{BASE_URL}/", timeout=10)
    check("登录后首页 200", r.status_code == 200, f"实际 {r.status_code}")
    check("首页显示用户名", USER in r.text, "页面未出现用户名")

    # 7. 上传任意格式（含中文名、无扩展名）
    uploads = {
        "release.zip": b"PK\x03\x04 zip payload",
        "build.log": b"line1\nline2\n",
        "app-v1.tar.gz": b"\x1f\x8b gzip payload",
        "设计文档.pdf": b"%PDF-1.4 fake",
        "Makefile": b"all:\n\techo hi\n",
    }
    for name, content in uploads.items():
        r = s.post(f"{BASE_URL}/upload", files={"file": (name, content)}, timeout=30)
        ok = r.status_code == 200 and r.json().get("data", {}).get("name") == name
        check(f"上传 {name}", ok, f"实际 {r.status_code} {r.text[:120]}")

    # 8. 同名覆盖
    r = s.post(
        f"{BASE_URL}/upload",
        files={"file": ("build.log", b"v2\n")},
        timeout=30,
    )
    check(
        "同名文件覆盖标记",
        r.status_code == 200 and r.json()["data"].get("overwritten") is True,
        f"实际 {r.status_code} {r.text[:120]}",
    )

    # 9. 查询
    r = s.get(f"{BASE_URL}/query", timeout=10)
    body = r.json()
    names = {f["name"] for f in body["data"]["files"]}
    check("查询返回全部 5 个文件", body["data"]["count"] == 5, f"实际 {body['data']['count']}")
    check("查询结果含中文名", "设计文档.pdf" in names, str(names))

    r = s.get(f"{BASE_URL}/query", params={"keyword": "build"}, timeout=10)
    body = r.json()
    check(
        "关键字过滤",
        body["data"]["count"] == 1 and body["data"]["files"][0]["name"] == "build.log",
        str(body["data"]),
    )

    # 10. 下载完整性与响应头
    for name, content in uploads.items():
        r = s.get(f"{BASE_URL}/download", params={"filename": name}, timeout=30)
        expected = md5(b"v2\n") if name == "build.log" else md5(content)
        check(
            f"下载 {name} 内容一致",
            r.status_code == 200 and md5(r.content) == expected,
            f"实际 {r.status_code} size={len(r.content)}",
        )
        check(
            f"下载 {name} Content-Length 正确",
            r.headers.get("Content-Length") == str(len(r.content)),
            f"实际 {r.headers.get('Content-Length')} vs {len(r.content)}",
        )

    r = s.get(f"{BASE_URL}/download", params={"filename": "设计文档.pdf"}, timeout=10)
    check(
        "中文名下载头带 UTF-8 编码",
        "filename*=UTF-8''" in r.headers.get("Content-Disposition", ""),
        r.headers.get("Content-Disposition", ""),
    )

    # 11. 用户隔离
    r = s.post(
        f"{BASE_URL}/upload",
        params={"username": OTHER},
        files={"file": ("other.txt", b"other user")},
        headers=auth,
        timeout=30,
    )
    check("另一用户上传成功", r.status_code == 200, f"实际 {r.status_code}")

    r = s.get(f"{BASE_URL}/query", timeout=10)
    check("当前用户查不到他人文件", OTHER not in r.text, r.text[:200])

    r = s.get(f"{BASE_URL}/download", params={"filename": "other.txt"}, timeout=10)
    check("当前用户下载他人文件 404", r.status_code == 404, f"实际 {r.status_code}")

    # 12. 路径穿越（URL 里预编码，避免 requests 二次编码）
    for bad_url in [
        f"{BASE_URL}/download?filename=../etc/passwd",
        f"{BASE_URL}/download?filename=..%2Fetc%2Fpasswd",
    ]:
        r = s.get(bad_url, timeout=10)
        check(f"路径穿越被拒 {bad_url.split('filename=')[1]}", r.status_code == 400, f"实际 {r.status_code}")

    r = requests.get(
        f"{BASE_URL}/query", params={"username": "../root"}, headers=auth, timeout=10
    )
    check("非法用户名被拒", r.status_code == 400, f"实际 {r.status_code}")

    # 13. 仅用 X-Token 调用 API
    r = requests.get(
        f"{BASE_URL}/query", params={"username": USER}, headers=auth, timeout=10
    )
    check("X-Token 直接调用 API", r.status_code == 200, f"实际 {r.status_code}")

    # 14. 删除
    r = s.delete(f"{BASE_URL}/file", params={"filename": "Makefile"}, timeout=10)
    check("删除文件 200", r.status_code == 200, f"实际 {r.status_code}")
    r = s.get(f"{BASE_URL}/download", params={"filename": "Makefile"}, timeout=10)
    check("删除后下载 404", r.status_code == 404, f"实际 {r.status_code}")
    r = s.delete(f"{BASE_URL}/file", params={"filename": "Makefile"}, timeout=10)
    check("重复删除 404", r.status_code == 404, f"实际 {r.status_code}")

    # 15. 退出登录
    r = s.post(f"{BASE_URL}/logout", allow_redirects=False, timeout=10)
    check("退出登录跳转", r.status_code == 302, f"实际 {r.status_code}")
    r = s.get(f"{BASE_URL}/query", timeout=10)
    check("退出后访问需登录", r.status_code == 401, f"实际 {r.status_code}")

    print()
    print(f"通过 {passed} 项，失败 {failed} 项")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"{RED}[ERROR]{RESET} {exc}")
        sys.exit(1)