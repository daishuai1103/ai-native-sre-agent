# AI Native SRE Agent

基于 `netbox-community/netbox-docker` 的 AI Native SRE 智能运维系统，具备 **Tool Calling 结构 + 多 Agent 闭环** 的完整实现。

> 从发现异常 → 智能路由 → 多 Agent 诊断 → 自动修复 → 生成报告，全链路可视化。

---

## ✨ 功能特性

- 🤖 **5 个 Specialist Agent**：Observer → Diagnosis → Planner → Executor → Reporter
- 🔧 **Tool Calling（ReAct 模式）**：10 个工具注册在 Tool Registry，Agent 自主调用
- 🧠 **智能路由**：混合路由架构（LightGBM/ONNX + 规则回退），5 类告警分类
- 🚨 **自动告警采集**：Docker API + Prometheus 双源巡检，30 秒间隔
- 💚 **自动自愈**：支持容器重启、资源调整、磁盘清理等修复动作（DRY_RUN 安全模式）
- 📊 **可视化前端**：三栏布局 + 实时 SSE 推送 + ReAct 思考过程可视化
- 🎯 **仪表盘**：告警统计、自愈率、处理耗时、来源分布
- ⚡ **模拟告警**：一键注入演示告警，方便展示

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                     React 可视化前端                          │
│  [告警列表 | ReAct执行链路 | 诊断报告 | 仪表盘 | 路由卡片]   │
└──────────────────────────────┬──────────────────────────────┘
                               │ SSE 实时推送
┌──────────────────────────────▼──────────────────────────────┐
│                    Orchestrator 编排器                        │
│  Observer → Diagnosis → Planner → Executor → Reporter       │
└───────────────┬──────────────────────────┬───────────────────┘
                │                          │
        ┌───────▼───────┐          ┌───────▼───────┐
        │  HybridRouter │          │ Tool Registry │
        │ ONNX + 规则   │          │ 10 个工具      │
        └───────┬───────┘          └───────┬───────┘
                │                          │
        ┌───────▼──────────────────────────▼───────┐
        │  Docker API  │  Prometheus  │  NetBox    │
        └───────────────────────────────────────────┘
```

---

## 📁 项目结构

```
ai-native-sre-agent/
├── backend/
│   ├── cmd/server/main.go       # 入口
│   ├── internal/
│   │   ├── agent/                # 5 个 Agent + Orchestrator + ReAct
│   │   ├── alerter/              # 自动告警采集
│   │   ├── router/               # 智能路由（混合 ONNX + 规则）
│   │   ├── tools/                # Tool Registry + 10 个工具
│   │   ├── server/               # HTTP + SSE + EventBus
│   │   ├── llm/                  # LLM 客户端
│   │   ├── config/               # 配置加载
│   │   └── models/               # 数据模型
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/           # UI 组件
│   │   ├── api/                  # API 封装
│   │   ├── store/                # 状态管理
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── scripts/
│   ├── setup_vm.sh               # VM 环境初始化脚本
│   ├── inject_anomaly.sh         # 异常注入脚本
│   ├── train_router.py           # LightGBM 训练 + ONNX 导出
│   └── prometheus.yml            # Prometheus 配置
├── docs/
│   └── architecture.md           # 架构文档
├── docker-compose.yml            # NetBox + Redis + PG + Prometheus
├── .env.example                  # 环境变量模板
├── start.sh                      # 一键启动脚本（Linux/macOS）
├── Makefile                      # 编译命令
├── LICENSE
└── README.md
```

---

## 🚀 快速开始

### 环境要求

| 依赖 | 版本要求 | 说明 |
|---|---|---|
| Docker | 20.10+ | 运行 netbox-docker |
| Docker Compose | v2+ | 一键编排 |
| Go | 1.22+ | 后端开发 |
| Node.js | 20+ | 前端开发 |
| Python | 3.10+ | 训练路由模型（可选） |

### 三步启动

#### 第 1 步：克隆项目

```bash
git clone <你的仓库地址>
cd ai-native-sre-agent
```

#### 第 2 步：配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填入你的 OPENAI_API_KEY
```

或者直接设置环境变量：

```bash
export OPENAI_API_KEY=sk-xxxx
export OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_MODEL=gpt-4o-mini
export AGENT_DRY_RUN=true
```

#### 第 3 步：启动服务

**方式一：使用启动脚本（推荐首次使用）**

```bash
bash start.sh
```

**方式二：手动启动**

```bash
# 终端 1：启动 NetBox + Prometheus
docker compose up -d

# 终端 2：启动后端
cd backend
go mod tidy
go run ./cmd/server

# 终端 3：启动前端
cd frontend
npm install
npm run dev
```

然后打开 http://localhost:5173 即可使用。

---

## 🎮 使用说明

### 方式 1：模拟告警演示（推荐）

1. 打开页面，界面默认干净
2. 点击左侧「⚡ 模拟告警」按钮
3. 5 条不同类型的告警依次注入
4. 点击任意告警，查看中间的 ReAct 执行链路和右侧的报告

### 方式 2：手动输入告警

在底部对话框输入告警描述，例如：
- `Redis 内存使用率过高`
- `PostgreSQL 慢查询增多`
- `Nginx 5xx 错误率上升`

### 方式 3：自动巡检模式

设置 `ALERTER_ENABLED=true` 后启动后端，系统会每 30 秒自动巡检 Docker 和 Prometheus，发现异常自动触发 Agent 处理。

```bash
export ALERTER_ENABLED=true
export ALERTER_INTERVAL=30
go run ./cmd/server
```

---

## 🛠️ 工具清单

| 工具 | 类型 | 功能 | 风险 |
|---|---|---|---|
| `docker_stats` | 诊断 | 查看容器资源使用率 | 低 |
| `docker_logs` | 诊断 | 查看容器日志 | 低 |
| `docker_ps` | 诊断 | 查看容器列表和状态 | 低 |
| `prometheus_query` | 诊断 | 执行 PromQL 查询 | 低 |
| `docker_restart` | 修复 | 重启容器 | 中 |
| `docker_update_limit` | 修复 | 调整容器资源限制 | 低 |
| `docker_prune` | 修复 | 清理未使用资源 | 中 |
| `docker_exec` | 修复 | 容器内执行命令 | 高 |
| `service_restart` | 修复 | 重启容器内服务 | 低 |
| `action` | 修复 | 通用动作执行 | 中 |

> **DRY_RUN 模式**（默认 `AGENT_DRY_RUN=true`）下，修复工具只会模拟执行，不会真的操作容器。

---

## 🧠 智能路由

### 5 种告警分类

| 分类 | 示例 |
|---|---|
| 容器资源 (container) | 容器内存过高、CPU 飙升 |
| 数据库 (db) | PostgreSQL 慢查询、连接数过高 |
| 缓存服务 (cache) | Redis 内存过高、连接数满 |
| 网络 (network) | Nginx 5xx 上升、响应超时 |
| Web 服务 (web) | NetBox 500 错误、页面加载慢 |

### 混合路由架构

- **ONNX 模式**：LightGBM 模型推理，准确率高，需要安装 onnxruntime
- **规则模式**（默认）：关键词匹配，开箱即用，零依赖
- 置信度低于阈值时自动回退到规则路由

### 启用 ONNX 模式

```bash
# 1. 训练模型
cd scripts
pip install lightgbm onnxmltools numpy scikit-learn
python train_router.py --samples 200 \
  --output ../backend/internal/router/model.onnx \
  --vocab-output ../backend/internal/router/vocab.json

# 2. 安装 onnxruntime-go（需要 CGO）
# 参考 https://github.com/microsoft/onnxruntime-go

# 3. 用 onnx tag 编译
cd ../backend
go run -tags onnx ./cmd/server
```

---

## 🔧 开发指南

### 新增一个工具

1. 在 `backend/internal/tools/` 下新建文件，实现 `Tool` 接口：

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Call(ctx context.Context, args map[string]any) (string, error)
}
```

2. 在 `main.go` 中注册：
```go
registry.Register(tools.YourTool{})
```

### 新增一个 Agent

1. 在 `backend/internal/agent/` 下新建文件，实现 `Agent` 接口：

```go
type Agent interface {
    Name() string
    Role() string
    SystemPrompt(route string) string
    Tools() []string
    MaxIterations() int
}
```

2. 在 `orchestrator.go` 的 agent 列表中加入。

### 前端开发

```bash
cd frontend
npm run dev      # 开发模式
npm run build    # 生产构建
npm run preview  # 预览构建结果
```

---

## ⚙️ 环境变量说明

| 变量 | 默认值 | 说明 |
|---|---|---|
| `OPENAI_API_KEY` | `""` | LLM API Key（必填） |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | API 地址 |
| `OPENAI_MODEL` | `gpt-4o-mini` | 模型名称 |
| `AGENT_DRY_RUN` | `true` | DRY_RUN 安全模式 |
| `SERVER_PORT` | `8081` | 后端端口 |
| `PROMETHEUS_URL` | `http://localhost:9090` | Prometheus 地址 |
| `NETBOX_API_TOKEN` | `""` | NetBox API Token |
| `ALERTER_ENABLED` | `false` | 是否开启自动告警巡检 |
| `ALERTER_INTERVAL` | `30` | 巡检间隔（秒） |
| `ONNX_MODEL_PATH` | `internal/router/model.onnx` | ONNX 模型路径 |
| `ONNX_VOCAB_PATH` | `internal/router/vocab.json` | 词典路径 |
| `ROUTER_THRESHOLD` | `0.6` | 路由置信度阈值 |

---

## 📚 相关文档

- [架构文档](docs/architecture.md)

---

## 📝 License

MIT License - 详见 [LICENSE](LICENSE)
