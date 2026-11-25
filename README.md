# 文件中转服务

按用户名隔离的文件上传、查询、下载服务。任意格式文件，带网页界面。

## 功能

- **上传** - 任意扩展名，按用户名分目录存放，同名覆盖
- **下载** - 流式下载，中文文件名不乱码，类型按扩展名推断
- **查询** - 按用户名列出文件，支持文件名关键字过滤
- **删除** - 删除单个文件
- **网页** - 拖拽上传、文件列表、下载、删除，口令登录
- **自动清理** - 按文件修改时间清理过期文件，可关闭

## 快速开始

```bash
# 设置访问口令
export AUTH_TOKEN=$(openssl rand -hex 24)

# 编译并后台运行
./run.sh -d

# 查看日志
./run.sh -l

# 停止
./run.sh -s
```

浏览器打开 `http://<host>:8080/`，输入用户名与口令即可使用。

`run.sh` 会自动加载 `configs/.env`，可以把 `AUTH_TOKEN` 写在该文件里，避免进入仓库。

## 配置

配置文件 `configs/config.yaml`：

```yaml
server:
  port: 8080
  read_timeout: 60
  write_timeout: 300
  idle_timeout: 120

storage:
  path: "./data"            # 实际结构 {path}/{username}/{filename}
  max_age_seconds: 2592000  # 0 表示不清理

upload:
  max_size_mb: 1024

ratelimit:
  limit: 300
  interval_seconds: 60

auth:
  enabled: true
  token: ""                 # 留空则读 AUTH_TOKEN 环境变量
  session_secret: ""        # 留空则由 token 派生
  session_ttl_hours: 12
```

密钥优先级：`AUTH_TOKEN` 环境变量 > `auth.token`。启用鉴权但两者都为空时服务拒绝启动。

## 接口

| 接口 | 方法 | 说明 |
|---|---|---|
| `/healthz` | GET | 健康检查，免鉴权 |
| `/upload` | POST | 上传，字段 `file`，参数 `username` |
| `/download` | GET | 下载，参数 `username`、`filename` |
| `/query` | GET | 查询，参数 `username`、`keyword` |
| `/file` | DELETE | 删除，参数 `username`、`filename` |
| `/login` `/logout` | GET/POST | 网页登录与退出 |
| `/api/me` | GET | 当前登录用户 |

鉴权方式三选一：请求头 `X-Token`、查询参数 `token`、网页会话 cookie。完整字段与错误码见 [API.md](API.md)。

## 存储结构

```
data/
├── alice/
│   ├── build.log
│   └── release-v1.tar.gz
└── bob/
    └── screenshot.png
```

一个用户一个目录，目录名即用户名。上传先写临时文件再原子改名，不会读到半截文件。

## 目录结构

```
LogService/
├── cmd/server/main.go
├── configs/config.yaml
├── internal/
│   ├── api/          # HTTP 处理器与网页模板
│   ├── config/       # 配置加载
│   ├── middleware/   # 日志、限流、鉴权、会话、Panic 恢复
│   ├── model/        # 数据模型
│   └── service/      # 存储、校验、清理
├── run.sh
└── e2e_test.py
```

## 测试

```bash
# 单元测试
go test ./...

# 端到端测试
AUTH_TOKEN=<口令> BASE_URL=http://127.0.0.1:8080 python3 e2e_test.py
```

## 已知限制

- 网页「下载时自选保存位置」依赖浏览器的文件系统访问 API，只在 HTTPS 或 localhost 下生效。HTTP 部署时回退为浏览器默认下载目录，可在浏览器设置里勾选「下载前询问每个文件的保存位置」。
- 8080 目前是明文 HTTP，口令在网络上可被嗅探。对公网长期使用建议前置 Caddy/Nginx 启用 HTTPS。