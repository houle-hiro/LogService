# API 接口文档

基础 URL: `http://localhost:8080`

---

## 1. 健康检查

```
GET /
```

**响应:**

```json
{
  "code": 0,
  "message": "log service is running"
}
```

---

## 2. 上传日志文件

```
POST /upload
Content-Type: multipart/form-data
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | zip 格式日志文件 |

**请求示例:**

```bash
curl -X POST -F "file=@myapp_abc123_20240101120000.zip" http://localhost:8080/upload
```

**成功响应 (200):**

```json
{
  "code": 0,
  "message": "上传成功",
  "data": "myapp_abc123_20240101120000.zip"
}
```

**错误响应 (400):**

未上传文件或表单字段名不匹配时:

```json
{
  "code": 400,
  "message": "上传文件为空: request EOF"
}
```

文件名不符合规范时:

```json
{
  "code": 400,
  "message": "文件名格式错误，正确格式: {名称}_{commit_id}_{datetime}.zip，datetime 为 14 位数字"
}
```

**错误响应 (413):**

上传文件超过 `max_size_mb` 限制时返回。

---

## 3. 下载日志文件

```
GET /download?filename={filename}
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| filename | string | 是 | 文件名，需符合命名规范 |

**请求示例:**

```bash
curl -O -J "http://localhost:8080/download?filename=myapp_abc123_20240101120000.zip"
```

**成功响应:**

- 状态码: 200
- Content-Type: `application/zip`
- Content-Disposition: `attachment; filename="myapp_abc123_20240101120000.zip"`
- 响应体: 文件二进制内容

**错误响应 (400):**

```json
{
  "code": 400,
  "message": "缺少 filename 参数"
}
```

```json
{
  "code": 400,
  "message": "文件名格式错误，正确格式: {名称}_{commit_id}_{datetime}.zip，datetime 为 14 位数字"
}
```

**错误响应 (404):**

```json
{
  "code": 404,
  "message": "文件不存在"
}
```

---

## 4. 查询日志文件

```
GET /query?name={name}&commit={commit}
```

**参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 日志名称前缀 |
| commit | string | 否 | commit_id 前缀 |

两个参数至少传一个。支持的组合:
- `?name=myapp` - 按名称查询
- `?commit=abc123` - 按 commit 查询
- `?name=myapp&commit=abc123` - 同时按名称和 commit 查询

**请求示例:**

```bash
curl "http://localhost:8080/query?name=myapp&commit=abc123"
```

**成功响应 (200):**

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "files": [
      {
        "name": "myapp_abc123_20240101120000.zip",
        "size": 102400,
        "modified_time": "2024-01-01 12:00:00"
      },
      {
        "name": "myapp_abc123_20240102120000.zip",
        "size": 104857,
        "modified_time": "2024-01-02 12:00:00"
      }
    ],
    "count": 2
  }
}
```

**响应字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| files | array | 文件列表 |
| files[].name | string | 文件名 |
| files[].size | int64 | 文件大小(字节) |
| files[].modified_time | string | 最后修改时间 |
| count | int | 匹配文件数量 |

**错误响应 (500):**

```json
{
  "code": 500,
  "message": "查询失败: ..."
}
```

---

## 通用响应格式

```json
{
  "code": 0,
  "message": "状态描述",
  "data": null
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功，非0=错误 |
| message | string | 状态描述 |
| data | any | 可选，响应数据 |

## 错误码

| HTTP 状态码 | code | 说明 |
|-------------|------|------|
| 200 | 0 | 成功 |
| 400 | 400 | 参数错误/文件名不合法 |
| 404 | 404 | 文件不存在 |
| 405 | 405 | HTTP 方法不允许 |
| 413 | 413 | 文件过大 |
| 429 | 429 | 请求过于频繁(限流) |
| 500 | 500 | 服务内部错误 |

## 文件名验证规则

正则: `^[a-zA-Z0-9\-]+_[a-zA-Z0-9\-]+_\d{14}\.zip$`

- 名称部分: 仅字母、数字、连字符
- commit_id 部分: 仅字母、数字、连字符
- datetime 部分: 14 位数字 (YYYYMMDDhhmmss)
- 扩展名: `.zip`

合法示例:
- `myapp-abc123_20240101120000.zip`
- `service-v1_commit-789_20241231235959.zip`

非法示例:
- `my app_abc123_20240101120000.zip` (名称含空格)
- `myapp_abc123_20240101.zip` (datetime 不足14位)
- `myapp_abc123_20240101120000.txt` (非 .zip 扩展名)
