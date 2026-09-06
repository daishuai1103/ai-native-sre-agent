#!/bin/bash
# ============================================
# AI Native SRE Agent 一键启动脚本（Linux/macOS）
# 用法：bash start.sh
# ============================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "  AI Native SRE Agent 启动脚本"
echo "========================================"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi
echo "✅ Docker 已安装"

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.22+"
    exit 1
fi
echo "✅ Go 已安装 ($(go version))"

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装，请先安装 Node.js 20+"
    exit 1
fi
echo "✅ Node.js 已安装 (v$(node -v))"

# 加载 .env（如果存在）
if [ -f .env ]; then
    echo ""
    echo "📄 加载 .env 配置文件"
    export $(grep -v '^#' .env | xargs)
else
    echo ""
    echo "⚠️  未找到 .env 文件，将使用默认配置"
    echo "   建议复制 .env.example 为 .env 并配置 API Key"
fi

# 检查 API Key
if [ -z "$OPENAI_API_KEY" ] || [ "$OPENAI_API_KEY" = "your_api_key_here" ]; then
    echo ""
    echo "⚠️  未配置 OPENAI_API_KEY，Agent 将走降级模式（规则路由 + fallback）"
    echo "   想体验完整 AI 能力，请在 .env 中配置 API Key"
fi

echo ""
echo "🚀 启动 netbox-docker + Prometheus..."
docker compose up -d
echo "✅ Docker 容器启动中（NetBox 启动需要 1-2 分钟）"

echo ""
echo "📦 安装前端依赖..."
cd frontend
if [ ! -d node_modules ]; then
    npm install
fi
echo "✅ 前端依赖已就绪"

echo ""
echo "📦 下载后端依赖..."
cd ../backend
go mod tidy
echo "✅ 后端依赖已就绪"

echo ""
echo "========================================"
echo "  环境准备完成！"
echo "========================================"
echo ""
echo "接下来请手动启动（需要两个终端）："
echo ""
echo "终端 1 - 后端："
echo "  cd backend"
echo "  # 配置环境变量（如果不用 .env）"
echo "  export OPENAI_API_KEY=你的key"
echo "  export OPENAI_BASE_URL=https://api.openai.com/v1"
echo "  export OPENAI_MODEL=gpt-4o-mini"
echo "  go run ./cmd/server"
echo ""
echo "终端 2 - 前端："
echo "  cd frontend"
echo "  npm run dev"
echo ""
echo "然后打开 http://localhost:5173 即可使用"
echo ""
