#!/bin/bash

# =============================================================================
# 日志服务启动脚本
# 支持编译、运行、编译并运行三种模式，支持后台运行
# =============================================================================

set -euo pipefail

# 颜色定义
declare -r RED='\033[0;31m'
declare -r GREEN='\033[0;32m'
declare -r YELLOW='\033[1;33m'
declare -r NC='\033[0m'

# 项目配置
declare -r BINARY_NAME="log-service"
declare -r BUILD_DIR="bin"
declare -r MAIN_FILE="cmd/server/main.go"
declare -r LOG_DIR="logs"
declare -r PID_FILE="logs/log-service.pid"

# 帮助信息
show_help() {
    cat << EOF
${GREEN}日志服务管理脚本${NC}

用法: $0 [选项]

选项:
    ${YELLOW}-b, --build${NC}       仅编译项目
    ${YELLOW}-r, --run${NC}         前台运行（适合开发调试）
    ${YELLOW}-a, --all${NC}         编译并前台运行（默认）
    ${YELLOW}-d, --daemon${NC}      编译并后台运行（生产模式）
    ${YELLOW}-s, --stop${NC}        停止后台运行的服务
    ${YELLOW}-l, --log${NC}         查看运行日志
    ${YELLOW}-c, --clean${NC}       清理编译产物和日志
    ${YELLOW}-h, --help${NC}        显示此帮助信息

示例:
    $0              # 编译并前台运行
    $0 -d           # 编译并后台运行
    $0 -s           # 停止后台服务
    $0 -l           # 查看运行日志
    $0 -r           # 仅运行（不编译）
    $0 -b           # 仅编译

日志位置: ${LOG_DIR}/log-service.log
EOF
}

# 打印日志
log_info() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] [INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] [WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR]${NC} $1"
}

# 检查 Go 环境
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go 环境，请先安装 Go"
        exit 1
    fi
}

# 编译项目
build_project() {
    log_info "开始编译..."
    
    mkdir -p "$BUILD_DIR"
    go mod tidy
    go build -ldflags "-s -w" -o "$BUILD_DIR/$BINARY_NAME" "$MAIN_FILE"
    
    log_info "编译成功: $BUILD_DIR/$BINARY_NAME"
}

# 创建日志目录
ensure_log_dir() {
    if [ ! -d "$LOG_DIR" ]; then
        mkdir -p "$LOG_DIR"
    fi
}

# 前台运行
run_foreground() {
    local binary_path="$BUILD_DIR/$BINARY_NAME"
    
    if [ ! -f "$binary_path" ]; then
        build_project
    fi
    
    log_info "启动服务（前台模式）..."
    log_info "按 Ctrl+C 停止服务"
    echo "----------------------------------------"
    
    "$binary_path"
}

# 后台运行
run_daemon() {
    local binary_path="$BUILD_DIR/$BINARY_NAME"
    
    # 检查是否已在运行
    if [ -f "$PID_FILE" ]; then
        local old_pid=$(cat "$PID_FILE")
        if kill -0 "$old_pid" 2>/dev/null; then
            log_warn "服务已在运行中 (PID: $old_pid)"
            log_warn "使用 '$0 -s' 停止后再启动"
            exit 1
        else
            rm -f "$PID_FILE"
        fi
    fi
    
    if [ ! -f "$binary_path" ]; then
        build_project
    fi
    
    ensure_log_dir
    
    log_info "启动服务（后台模式）..."
    
    # 后台启动，输出重定向到日志文件
    nohup "$binary_path" > "$LOG_DIR/log-service.log" 2>&1 &
    local pid=$!
    
    # 保存 PID
    echo "$pid" > "$PID_FILE"
    
    # 等待一下确认进程启动成功
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        log_info "服务已启动"
        log_info "PID: $pid"
        log_info "日志文件: $LOG_DIR/log-service.log"
        log_info "停止服务: $0 -s"
    else
        log_error "服务启动失败，请检查日志: $LOG_DIR/log-service.log"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# 停止后台服务
stop_daemon() {
    if [ ! -f "$PID_FILE" ]; then
        log_warn "未找到 PID 文件，服务可能未运行"
        exit 1
    fi
    
    local pid=$(cat "$PID_FILE")
    
    if ! kill -0 "$pid" 2>/dev/null; then
        log_warn "进程 $pid 已不存在"
        rm -f "$PID_FILE"
        exit 1
    fi
    
    log_info "正在停止服务 (PID: $pid)..."
    kill "$pid"
    
    # 等待进程退出
    local timeout=10
    while [ $timeout -gt 0 ]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            log_info "服务已停止"
            rm -f "$PID_FILE"
            return 0
        fi
        sleep 1
        timeout=$((timeout - 1))
    done
    
    # 强制停止
    log_warn "进程未响应，强制停止..."
    kill -9 "$pid" 2>/dev/null
    rm -f "$PID_FILE"
    log_info "服务已强制停止"
}

# 查看日志
show_log() {
    local log_file="$LOG_DIR/log-service.log"
    
    if [ ! -f "$log_file" ]; then
        log_warn "日志文件不存在: $log_file"
        exit 1
    fi
    
    log_info "显示最近 50 行日志..."
    tail -n 50 "$log_file"
}

# 清理
clean_project() {
    log_info "清理中..."
    
    # 先停止服务
    if [ -f "$PID_FILE" ]; then
        stop_daemon 2>/dev/null || true
    fi
    
    [ -d "$BUILD_DIR" ] && rm -rf "$BUILD_DIR" && log_info "已删除: $BUILD_DIR/"
    [ -d "$LOG_DIR" ] && rm -rf "$LOG_DIR" && log_info "已删除: $LOG_DIR/"
    [ -d "data" ] && rm -rf "data" && log_info "已删除: data/"
    
    go clean -cache 2>/dev/null || true
    log_info "清理完成"
}

# 主函数
main() {
    local script_dir=$(cd "$(dirname "$0")" && pwd)
    cd "$script_dir"
    
    check_go_env
    
    local mode="all"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--build)
                mode="build"
                shift
                ;;
            -r|--run)
                mode="run"
                shift
                ;;
            -a|--all)
                mode="all"
                shift
                ;;
            -d|--daemon)
                mode="daemon"
                shift
                ;;
            -s|--stop)
                mode="stop"
                shift
                ;;
            -l|--log)
                mode="log"
                shift
                ;;
            -c|--clean)
                mode="clean"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    case $mode in
        build)
            build_project
            ;;
        run)
            run_foreground
            ;;
        all)
            build_project
            run_foreground
            ;;
        daemon)
            build_project
            run_daemon
            ;;
        stop)
            stop_daemon
            ;;
        log)
            show_log
            ;;
        clean)
            clean_project
            ;;
    esac
}

trap 'echo; log_warn "收到中断信号"; exit 0' INT TERM

main "$@"
