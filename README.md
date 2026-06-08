# Log Service

日志文件管理服务，支持 zip 格式日志文件的上传、下载、查询和自动老化清理。

## 功能

- **上传** - 上传 zip 日志文件，自动验证文件名格式
- **下载** - 根据文件名下载日志文件
- **查询** - 按名称和/或 commit_id 前缀查询文件列表
- **老化清理** - 自动清理超时文件，可配置保留时长
- **限流** - 内置请求限流保护

## 文件命名规范

```
{名称}_{commit_id}_{datetime}.zip
```

- 名称和 commit_id 仅允许 `a-zA-Z0-9-`
- datetime 为 14 位数字，格式 `YYYYMMDDhhmmss`
- 示例: `myapp_abc123_20240101120000.zip`

## 快速开始

### 环境要求

- Go 1.21+
- Linux/macOS

### 编译与运行

```bash
# 编译
./run.sh -b

# 前台运行（开发调试）
./run.sh -r

# 编译并前台运行（默认）
./run.sh

# 编译并后台运行（生产模式）
./run.sh -d

# 停止后台服务
./run.sh -s

# 查看运行日志
./run.sh -l

# 清理编译产物和日志
./run.sh -c
```

## 配置

配置文件: `configs/config.yaml`

```yaml
server:
  port: 8080                # 监听端口
  read_timeout: 30          # 读取超时(秒)
  write_timeout: 30         # 写入超时(秒)
  idle_timeout: 120         # 空闲超时(秒)

storage:
  path: "./data"            # 文件存储路径
  max_age_seconds: 2592000  # 保留时长(秒)，默认30天

upload:
  max_size_mb: 100          # 最大上传大小(MB)

ratelimit:
  limit: 100                # 时间窗口内最大请求数
  interval_seconds: 60      # 限流时间窗口(秒)
```

## 目录结构

```
log-service/
├── cmd/server/main.go       # 入口
├── configs/config.yaml      # 配置文件
├── internal/
│   ├── api/                 # HTTP 处理器
│   ├── config/              # 配置加载
│   ├── middleware/           # 中间件(日志/限流/恢复)
│   ├── model/               # 数据模型
│   └── service/             # 业务逻辑(存储/清理/验证)
├── data/                    # 日志文件存储(自动创建)
├── logs/                    # 运行日志
├── run.sh                   # 启动脚本
└── e2e_test.py              # 端到端测试
```

## 存储结构

文件按名称分组存储在子目录中：

```
data/
├── myapp/
│   ├── myapp_abc123_20240101120000.zip
│   └── myapp_abc123_20240102120000.zip
└── service/
    └── service_commit789_20241231235959.zip
```

## 中间件

- **日志** - 记录每个请求的方法、路径、状态码和耗时
- **Panic 恢复** - 捕获 panic 并返回 500 错误
- **限流** - 基于滑动窗口的请求限流

## 测试

```bash
# 单元测试
cd log-service && go test ./internal/...

# 端到端测试
python3 e2e_test.py
```

## License

Internal use only.
