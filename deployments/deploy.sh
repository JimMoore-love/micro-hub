#!/bin/bash
# ============================================================
# MicroHub 一键部署脚本
# 支持：本地部署 / 远程部署 / Agent 分发
# 用法：
#   ./deploy.sh local        — 本地部署（编译+启动 API Server + 前端）
#   ./deploy.sh docker       — Docker Compose 部署
#   ./deploy.sh agent <ip>   — 分发并启动 Agent 到远程服务器
#   ./deploy.sh build        — 编译所有组件（API Server + Agent + 前端）
#   ./deploy.sh stop         — 停止所有服务
#   ./deploy.sh status       — 查看运行状态
# ============================================================

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 路径
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
API_DIR="$PROJECT_DIR/api-server"
FRONTEND_DIR="$PROJECT_DIR/frontend-vue"
AGENT_DIR="$PROJECT_DIR/agent"

# 默认配置
API_PORT=${API_PORT:-8081}
FRONTEND_PORT=${FRONTEND_PORT:-3200}
SERVER_URL=${SERVER_URL:-"http://localhost:$API_PORT"}

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "${BLUE}[STEP]${NC} $1"; }

# ==================== 编译 ====================

build_all() {
    log_step "编译所有组件"

    # API Server
    log_info "编译 API Server..."
    cd "$API_DIR"
    go build -o microhub-api .
    log_info "API Server 编译成功"

    # Agent (当前平台)
    log_info "编译 Agent (当前平台)..."
    cd "$AGENT_DIR"
    go build -o microhub-agent .
    log_info "Agent 编译成功"

    # Agent Linux 交叉编译
    log_info "编译 Agent (Linux amd64)..."
    GOOS=linux GOARCH=amd64 go build -o microhub-agent-linux-amd64 .
    log_info "Linux Agent 编译成功"

    # 前端
    log_info "编译前端..."
    cd "$FRONTEND_DIR"
    if [ -d "node_modules" ]; then
        npx vite build
        log_info "前端编译成功"
    else
        log_warn "node_modules 不存在，跳过前端编译"
    fi

    log_info "所有组件编译完成"
}

# ==================== 本地部署 ====================

deploy_local() {
    log_step "本地部署"

    # 检查 MySQL
    log_info "检查 MySQL..."
    if ! nc -z localhost 3306 2>/dev/null; then
        log_warn "MySQL 3306 端口不可达，请先启动 MySQL"
        log_warn "ServBay: D:\\ServBay\\bin\\mysql-server.cmd"
        log_warn "Docker: docker run -d --name mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root mysql:8"
        exit 1
    fi

    # 编译
    build_all

    # 停止旧进程
    stop_all

    # 启动 API Server
    log_step "启动 API Server (端口 $API_PORT)..."
    cd "$API_DIR"
    nohup ./microhub-api > /tmp/microhub-api.log 2>&1 &
    API_PID=$!
    log_info "API Server PID: $API_PID"

    # 等待 API Server 启动
    sleep 3
    if curl -s "http://localhost:$API_PORT/health" > /dev/null; then
        log_info "API Server 启动成功"
    else
        log_error "API Server 启动失败，查看日志: /tmp/microhub-api.log"
        cat /tmp/microhub-api.log
        exit 1
    fi

    # 启动前端
    log_step "启动前端 (端口 $FRONTEND_PORT)..."
    cd "$FRONTEND_DIR"
    nohup npx vite --port $FRONTEND_PORT > /tmp/microhub-frontend.log 2>&1 &
    FRONTEND_PID=$!
    log_info "Frontend PID: $FRONTEND_PID"

    sleep 3
    log_info "前端启动成功"

    # 启动本机 Agent
    log_step "启动本机 Agent..."
    cd "$AGENT_DIR"
    nohup ./microhub-agent -server "$SERVER_URL" -name "local-node" > /tmp/microhub-agent.log 2>&1 &
    AGENT_PID=$!
    log_info "Agent PID: $AGENT_PID"

    log_step "部署完成"
    echo ""
    echo "  前端:    http://localhost:$FRONTEND_PORT"
    echo "  API:     http://localhost:$API_PORT/health"
    echo "  登录:    admin / admin123"
    echo "  节点:    http://localhost:$FRONTEND_PORT/nodes"
    echo ""
    echo "  日志:"
    echo "    API Server:  /tmp/microhub-api.log"
    echo "    前端:        /tmp/microhub-frontend.log"
    echo "    Agent:       /tmp/microhub-agent.log"
}

# ==================== Docker 部署 ====================

deploy_docker() {
    log_step "Docker Compose 部署"
    cd "$PROJECT_DIR"

    if ! command -v docker &> /dev/null && ! command -v podman &> /dev/null; then
        log_error "未安装 docker 或 podman"
        exit 1
    fi

    COMPOSE_CMD="docker compose"
    if command -v podman &> /dev/null && ! command -v docker &> /dev/null; then
        COMPOSE_CMD="podman compose"
    fi

    log_info "使用 $COMPOSE_CMD ..."
    $COMPOSE_CMD up -d --build

    log_step "Docker 部署完成"
    $COMPOSE_CMD ps
    echo ""
    echo "  前端:    http://localhost:3200"
    echo "  API:     http://localhost:$API_PORT/health"
    echo "  登录:    admin / admin123"
}

# ==================== Agent 分发 ====================

deploy_agent() {
    local REMOTE_IP=$1
    if [ -z "$REMOTE_IP" ]; then
        echo "用法: ./deploy.sh agent <远程IP> [节点名称] [SSH用户]"
        echo "示例: ./deploy.sh agent 192.168.1.100 node-2 root"
        exit 1
    fi

    local NODE_NAME=${2:-"node-$(echo $REMOTE_IP | cut -d. -f4)"}
    local SSH_USER=${3:-"root"}
    local REMOTE_DIR="/opt/microhub-agent"
    local AGENT_BIN="$AGENT_DIR/microhub-agent-linux-amd64"

    log_step "部署 Agent 到 $REMOTE_IP (节点: $NODE_NAME)"

    # 编译 Linux Agent
    if [ ! -f "$AGENT_BIN" ]; then
        log_info "编译 Linux Agent..."
        cd "$AGENT_DIR"
        GOOS=linux GOARCH=amd64 go build -o microhub-agent-linux-amd64 .
    fi

    # 创建远程目录
    log_info "创建远程目录..."
    ssh "$SSH_USER@$REMOTE_IP" "mkdir -p $REMOTE_DIR"

    # 上传 Agent
    log_info "上传 Agent 二进制..."
    scp "$AGENT_BIN" "$SSH_USER@$REMOTE_IP:$REMOTE_DIR/microhub-agent"
    ssh "$SSH_USER@$REMOTE_IP" "chmod +x $REMOTE_DIR/microhub-agent"

    # 创建 systemd 服务
    log_info "创建 systemd 服务..."
    ssh "$SSH_USER@$REMOTE_IP" "cat > /etc/systemd/system/microhub-agent.service << 'EOF'
[Unit]
Description=MicroHub Agent
After=network.target

[Service]
Type=simple
ExecStart=$REMOTE_DIR/microhub-agent -server $SERVER_URL -name $NODE_NAME
Restart=always
RestartSec=10
User=root

[Install]
WantedBy=multi-user.target
EOF"

    # 启动服务
    log_info "启动 Agent 服务..."
    ssh "$SSH_USER@$REMOTE_IP" "systemctl daemon-reload && systemctl enable microhub-agent && systemctl start microhub-agent"

    # 检查状态
    sleep 2
    log_info "Agent 状态:"
    ssh "$SSH_USER@$REMOTE_IP" "systemctl status microhub-agent --no-pager" 2>/dev/null || true

    log_step "Agent 部署完成"
    echo ""
    echo "  节点:    $NODE_NAME ($REMOTE_IP)"
    echo "  Server:  $SERVER_URL"
    echo "  日志:    ssh $SSH_USER@$REMOTE_IP journalctl -u microhub-agent -f"
    echo ""
    echo "  在管理界面 http://localhost:$FRONTEND_PORT/nodes 查看节点状态"
}

# ==================== 停止 ====================

stop_all() {
    log_step "停止所有服务"
    pkill -f "microhub-api" 2>/dev/null && log_info "已停止 API Server" || log_info "API Server 未运行"
    pkill -f "microhub-agent" 2>/dev/null && log_info "已停止 Agent" || log_info "Agent 未运行"
    pkill -f "vite.*$FRONTEND_PORT" 2>/dev/null && log_info "已停止前端" || log_info "前端未运行"
}

# ==================== 状态 ====================

show_status() {
    log_step "服务状态"

    # API Server
    if curl -s "http://localhost:$API_PORT/health" > /dev/null 2>&1; then
        log_info "API Server:    运行中 (端口 $API_PORT)"
    else
        log_error "API Server:    未运行"
    fi

    # 前端
    if curl -s "http://localhost:$FRONTEND_PORT/" > /dev/null 2>&1; then
        log_info "前端:          运行中 (端口 $FRONTEND_PORT)"
    else
        log_error "前端:          未运行"
    fi

    # Agent
    if pgrep -f "microhub-agent" > /dev/null 2>&1; then
        log_info "Agent:         运行中"
    else
        log_warn "Agent:         未运行"
    fi

    # 节点列表
    log_info "已注册节点:"
    TOKEN=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/auth/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' 2>/dev/null | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$TOKEN" ]; then
        curl -s "http://localhost:$API_PORT/api/v1/nodes" -H "Authorization: Bearer $TOKEN" 2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)['data']
    if not data:
        print('  (无节点)')
    for n in data:
        print(f\"  {n['name']:<20} {n['ip']:<15} {n['status']:<8} 服务:{n.get('service_count',0)}\")
except:
    print('  (查询失败)')
" 2>/dev/null
    fi
}

# ==================== 主入口 ====================

case "${1:-}" in
    local)  deploy_local ;;
    docker) deploy_docker ;;
    agent)  deploy_agent "$2" "$3" "$4" ;;
    build)  build_all ;;
    stop)   stop_all ;;
    status) show_status ;;
    *)
        echo "MicroHub 一键部署脚本"
        echo ""
        echo "用法: ./deploy.sh <命令> [参数]"
        echo ""
        echo "命令:"
        echo "  local          本地部署（编译+启动 API Server + 前端 + Agent）"
        echo "  docker         Docker Compose 部署"
        echo "  agent <IP>     分发 Agent 到远程服务器"
        echo "  build          编译所有组件"
        echo "  stop           停止所有服务"
        echo "  status         查看运行状态"
        echo ""
        echo "示例:"
        echo "  ./deploy.sh local"
        echo "  ./deploy.sh docker"
        echo "  ./deploy.sh agent 192.168.1.100 node-2 root"
        echo "  ./deploy.sh status"
        ;;
esac
