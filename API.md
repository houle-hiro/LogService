# 接口文档

基础地址：`http://<host>:8080`

## 鉴权

所有接口（除 `/healthz`、`/login`）都需要鉴权，三种方式任选：

| 方式 | 用法 |
|---|---|
| 请求头 | `X-Token: <口令>` |
| 查询参数 | `?token=<口令>` |
| 会话 cookie | 浏览器登录后由服务端下发，自动携带 |

未通过鉴权返回 `401`：

```json
{"code":401,"message":"未登录或 token 无效，请在请求头带 X-Token"}
```

## 用户名规则

- 1~64 字符，正则 `^[a-z0-9][a-z0-9._@+-]{0,63}$`
- 统一转小写，不能含 `..`
- 每个用户名对应一个存储目录 `data/{username}/`

---

## 1. 健康检查

```
GET /healthz
```

免鉴权。

```json
{"code":0,"message":"ok","data":{"status":"ok","time":"2026-01-01T12:00:00+08:00"}}
```

---

## 2. 上传文件

```
POST /upload
Content-Type: multipart/form-data
```

| 参数 | 位置 | 必填 | 说明 |
|---|---|---|---|
| file | form | 是 | 任意格式文件 |
| username | query 或 form | 是* | 目标用户名；登录态下可省略，取会话用户 |

```bash
curl -H "X-Token: $TOKEN" -F "file=@release.tar.gz" \
  "http://host:8080/upload?username=alice"
```

成功：

```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "username": "alice",
    "name": "release.tar.gz",
    "size": 1048576,
    "overwritten": false
  }
}
```

`overwritten` 为 `true` 表示覆盖了同名文件。

| HTTP | 场景 |
|---|---|
| 400 | 缺 username、文件名非法、未带 file 字段 |
| 413 | 超过 `max_size_mb` |
| 500 | 写盘失败 |

---

## 3. 下载文件

```
GET /download?username=alice&filename=release.tar.gz
```

username 在登录态下可省略。

```bash
curl -H "X-Token: $TOKEN" -OJ \
  "http://host:8080/download?username=alice&filename=release.tar.gz"
```

成功响应头：

| 头 | 值 |
|---|---|
| Content-Type | 按扩展名推断，未知为 `application/octet-stream` |
| Content-Length | 文件字节数 |
| Content-Disposition | `attachment; filename="..."; filename*=UTF-8''...` |

| HTTP | 场景 |
|---|---|
| 400 | 缺 filename 或文件名含路径 |
| 404 | 文件不存在或不属于该用户 |

---

## 4. 查询文件

```
GET /query?username=alice&keyword=build
```

| 参数 | 必填 | 说明 |
|---|---|---|
| username | 是* | 登录态下可省略 |
| keyword | 否 | 文件名子串，不区分大小写 |

`name` 作为 `keyword` 的兼容别名保留。

```bash
curl -H "X-Token: $TOKEN" "http://host:8080/query?username=alice&keyword=log"
```

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "username": "alice",
    "keyword": "log",
    "files": [
      {
        "username": "alice",
        "name": "build.log",
        "size": 2048,
        "modified_time": "2026-01-01T12:00:00+08:00"
      }
    ],
    "count": 1
  }
}
```

只返回该用户名目录下的文件，按修改时间倒序。

---

## 5. 删除文件

```
DELETE /file?username=alice&filename=build.log
```

```bash
curl -X DELETE -H "X-Token: $TOKEN" \
  "http://host:8080/file?username=alice&filename=build.log"
```

| HTTP | 场景 |
|---|---|
| 200 | 删除成功 |
| 400 | 缺 filename 或文件名非法 |
| 404 | 文件不存在 |

---

## 6. 网页登录与退出

```
GET  /login
POST /login   form: username, token, next
GET|POST /logout
GET  /api/me
```

登录成功返回 `302` 并下发 `fs_session` cookie（HttpOnly、SameSite=Lax、默认 12 小时）。`next` 只接受站内相对路径。

`/api/me` 返回：

```json
{"code":0,"message":"ok","data":{"username":"alice"}}
```

---

## 通用响应格式

```json
{"code": 0, "message": "状态描述", "data": null}
```

## 错误码

| HTTP | code | 说明 |
|---|---|---|
| 200 | 0 | 成功 |
| 400 | 400 | 参数错误或名称非法 |
| 401 | 401 | 未登录或口令错误 |
| 404 | 404 | 文件不存在 |
| 405 | 405 | 方法不允许 |
| 413 | 413 | 文件过大 |
| 429 | 429 | 请求过于频繁 |
| 500 | 500 | 服务内部错误 |

## 文件名规则

- 取客户端文件名的最后一段，剥离路径
- 禁止 `/`、`\`、控制字符，不能是 `.` 或 `..`
- 最长 255 字节，UTF-8
- 扩展名不限