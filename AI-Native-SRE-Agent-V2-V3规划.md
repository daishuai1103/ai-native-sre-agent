# AI Native SRE Agent - V2 / V3 版本规划

> 基于 V1（当前版本）的架构，结合 MCP、Skill、DeepSeek Harness、Prompt 工程 等概念，规划后续演进方向。

---

## 版本路线图

| 版本 | 主题 | 核心变化 | 预计工作量 |
|---|---|---|---|
| **V1**（当前） | 多 Agent Tool Calling 闭环 | 5 个 Specialist Agent + 10 个工具 + 规则路由 + 可视化前端 | 已完成 |
| **V2** | Skill 化 + MCP 化 | 工具接入 MCP 协议 + Agent 封装为 Skill + 动态编排 | 3-5 天 |
| **V3** | Harness 级架构 | 类 DeepSeek Harness 的完整 Agent 框架 + 插件生态 + 安全沙箱 | 1-2 周 |

---

## V1 现状回顾

### 已实现能力
- ✅ 5 个 Specialist Agent（Observer / Diagnosis / Planner / Executor / Reporter）
- ✅ ReAct 思考循环 + Tool Calling
- ✅ 10 个运维工具（Docker 6 个 + Prometheus 1 个 + 通用 3 个）
- ✅ 混合智能路由（规则 + ONNX LightGBM）
- ✅ 自动告警采集（Docker + Prometheus）
- ✅ 三栏可视化前端（状态栏 + 告警列表 + ReAct 链路 + 报告）
- ✅ DRY_RUN 安全模式
- ✅ 模拟告警注入（演示用）

### 架构局限
- ❌ 工具硬编码，扩展需要改代码
- ❌ Agent 是固定流水线，不能动态编排
- ❌ Prompt 散落在代码里，不好管理
- ❌ 没有统一的安全沙箱和权限控制
- ❌ 没有 Skill/插件的概念，不可复用

---

# V2：Skill 化 + MCP 化

## 核心目标

> 把工具层接入 MCP 协议，把 Agent 封装为可复用的 Skill，实现动态编排和可插拔扩展。

---

## V2.1 工具层接入 MCP 协议

### 为什么做
- 当前工具是硬编码的 Go 结构体，扩展需要改代码
- MCP 是行业标准协议，工具可以跨语言、跨系统复用
- 以后加 Kubernetes、MySQL、Jenkins 工具，不用改 Go 代码

### 实现方式

#### 方案 A：内置 MCP Client（推荐）

```
┌─────────────────────────────────────────────────────┐
│                   Go Agent 后端                      │
│  ┌──────────┐    ┌──────────────────┐               │
│  │  Agent   │───▶│  MCP Client      │◀── 工具注册   │
│  │ Orchestrator │  │  (Tool Registry)│               │
│  └──────────┘    └────────┬─────────┘               │
│                           │                          │
│              ┌────────────┼────────────┐            │
│              ▼            ▼            ▼            │
│         本地工具      MCP Server   MCP Server       │
│         (Go实现)    (Docker)     (Prometheus)      │
└─────────────────────────────────────────────────────┘
```

**具体步骤**：

1. **定义 MCP Tool 接口**（替代当前的 Tool 接口）
   ```go
   type MCPTool interface {
       Name() string
       Description() string
       Schema() map[string]any  // JSON Schema for parameters
       Call(ctx context.Context, args map[string]any) (string, error)
   }
   ```

2. **实现 MCP Client**
   - 支持 stdio 模式（本地子进程）
   - 支持 HTTP 模式（远程 MCP Server）
   - 自动发现工具列表
   - 统一封装调用

3. **迁移现有工具为 MCP Server**
   - Docker 工具 → `docker-mcp-server`（可用 Python 写）
   - Prometheus 工具 → `prometheus-mcp-server`
   - 或者保持 Go 实现，通过 stdio MCP 协议暴露

4. **Registry 改造成 MCP Client + 本地工具混合**
   - 启动时从配置读取 MCP Server 列表
   - 连接每个 Server，获取工具列表
   - 合并到统一的工具注册表
   - Agent 调用工具时，透明路由到对应的 MCP Server

### 配置文件示例
```yaml
# config/mcp.yaml
mcp_servers:
  - name: docker
    type: stdio
    command: python3
    args: ["servers/docker_mcp.py"]
  - name: prometheus
    type: http
    url: http://localhost:9090/mcp
  - name: kubernetes
    type: stdio
    command: npx
    args: ["@k8s-mcp/server"]
```

### 对 V1 的改动量
- 新增：`internal/mcp/client.go`（MCP 客户端）
- 修改：`internal/tools/registry.go`（接入 MCP）
- 新增：`mcp-servers/` 目录（各工具的 MCP Server 实现）
- Agent 代码基本不用改（工具调用接口不变）

### 工作量评估
- MCP Client 实现：1 天
- Docker/Prometheus 工具迁移为 MCP Server：1 天
- 配置化 + 测试：1 天
- **合计：约 3 天**

---

## V2.2 Agent 封装为 Skill

### 为什么做
- 当前 5 个 Agent 是硬编码的流水线，不灵活
- Skill 化后可以动态组合、按需加载、版本管理
- 不同告警类型可以走不同的 Skill 组合

### 实现方式

#### Skill 定义
```go
type Skill struct {
    Name        string                 // 技能名，如 "redis-diagnosis"
    Version     string                 // 版本号
    Description string                 // 技能描述
    SystemPrompt string                // 核心 Prompt
    Tools       []string               // 可用工具名列表
    Workflow    WorkflowDef            // 工作流定义
    Knowledge   map[string]string      // 领域知识库
}
```

#### 工作流定义（Workflow）
```yaml
# skills/redis-diagnosis/skill.yaml
name: redis-diagnosis
version: 1.0.0
description: Redis 故障诊断与修复技能
tools:
  - docker_stats
  - docker_logs
  - docker_restart
  - prometheus_query
  - redis_info

workflow:
  - step: observe
    agent: observer
    prompt_template: "观测 Redis 容器 {container} 的运行状态..."
    tools: [docker_stats, redis_info]
  - step: diagnose
    agent: diagnosis
    prompt_template: "根据观测数据诊断 Redis 问题根因..."
    tools: [docker_logs, prometheus_query]
  - step: plan
    agent: planner
    prompt_template: "制定修复计划..."
  - step: execute
    agent: executor
    prompt_template: "执行修复计划并验证..."
    tools: [docker_restart]
  - step: report
    agent: reporter
    prompt_template: "生成结构化报告..."
```

#### Skill 注册表
```go
type SkillRegistry struct {
    skills map[string]*Skill
}

func (r *SkillRegistry) Register(skill *Skill) { ... }
func (r *SkillRegistry) Get(name string) (*Skill, error) { ... }
func (r *SkillRegistry) List() []*Skill { ... }
```

#### 动态路由 → Skill 匹配
```
告警 → 智能路由 → 匹配 Skill → 执行 Skill 工作流
                 ├─ redis 问题   → "redis-diagnosis" Skill
                 ├─ mysql 问题   → "mysql-diagnosis" Skill
                 ├─ nginx 问题   → "network-diagnosis" Skill
                 └─ 通用问题     → "general-sre" Skill
```

### 文件结构
```
skills/
  ├── redis-diagnosis/
  │   ├── skill.yaml
  │   ├── prompts/
  │   │   ├── observer.md
  │   │   ├── diagnosis.md
  │   │   └── ...
  │   └── knowledge/
  │       └── best_practices.md
  ├── mysql-diagnosis/
  ├── network-diagnosis/
  └── general-sre/
```

### 对 V1 的改动量
- 新增：`internal/skill/` 目录（Skill 定义、注册、加载）
- 新增：`skills/` 目录（各 Skill 的 yaml + prompt + knowledge）
- 修改：`internal/agent/orchestrator.go` → 改为执行 Skill Workflow
- 修改：`internal/router/` → 路由结果增加 Skill 名映射

### 工作量评估
- Skill 框架（定义 + 注册 + 加载）：1 天
- Workflow 执行引擎：1 天
- 迁移 5 个 Agent 为 5 个 Skill：1 天
- 智能路由 → Skill 匹配：0.5 天
- **合计：约 3.5 天**

---

## V2.3 Prompt 工程化管理

### 为什么做
- 当前 Prompt 散落在 Go 代码里，不好维护和版本管理
- Prompt 改了效果好不好，没有评估机制
- 不同场景需要不同的 Prompt 策略

### 实现方式

#### Prompt 文件化
```
prompts/
  ├── base/
  │   ├── observer.md
  │   ├── diagnosis.md
  │   ├── planner.md
  │   ├── executor.md
  │   └── reporter.md
  ├── skills/
  │   ├── redis-diagnosis/
  │   │   ├── observer.md
  │   │   └── diagnosis.md
  │   └── mysql-diagnosis/
  └── templates/
      ├── tool_calling_guide.md
      ├── output_format.md
      └── safety_guard.md
```

#### Prompt 模板变量
```markdown
你是 {{role}}，负责 {{responsibility}}。

可用工具：
{{tools_list}}

{{workflow_instruction}}

输出要求：
{{output_format}}
```

#### Prompt 版本管理
```go
type PromptVersion struct {
    Version  string
    Prompt   string
    Metrics  map[string]float64  // 准确率、工具调用成功率等
    IsActive bool
}
```

#### Prompt 评估（可选）
- 准备测试集（10-20 条典型告警）
- 批量运行，记录：工具调用准确率、诊断正确率、修复成功率
- A/B 测试不同 Prompt 版本的效果

### 工作量评估
- Prompt 提取到文件：0.5 天
- 模板引擎 + 变量替换：0.5 天
- Prompt 版本管理：0.5 天
- **合计：约 1.5 天**

---

## V2 整体工作量

| 模块 | 工作量 | 优先级 |
|---|---|---|
| MCP 工具接入 | 3 天 | P0 |
| Agent Skill 化 | 3.5 天 | P0 |
| Prompt 工程化 | 1.5 天 | P1 |
| **合计** | **约 8 天** | - |

**最小可行 V2**（只做 MCP + Skill 核心）：约 5 天

---

# V3：Harness 级架构

## 核心目标

> 构建类 DeepSeek Harness 的完整 Agent 运行框架，支持插件生态、安全沙箱、子 Agent 调度，从"运维 Agent"升级为"Agent 运行平台"。

---

## V3.1 安全沙箱与权限控制

### 为什么做
- 当前 DRY_RUN 是全局开关，太粗了
- 不同工具的风险等级不同，需要细粒度控制
- 生产环境需要审批机制

### 实现方式

#### 工具风险分级
| 风险等级 | 工具 | 默认权限 |
|---|---|---|
| 🔵 只读 | docker_stats, docker_logs, prometheus_query | 自动允许 |
| 🟡 低风险 | docker_update_limit, service_restart | 需要确认 |
| 🟠 中风险 | docker_restart, docker_prune | 需要确认 + 二次确认 |
| 🔴 高风险 | docker_exec | 需要审批 + 审计 |

#### 执行前确认机制
```go
type ApprovalRequirement int

const (
    ApprovalNone    ApprovalRequirement = iota  // 直接执行
    ApprovalConfirm                             // 前端确认
    ApprovalReview                              // 人工审批
)
```

#### 沙箱隔离
- 工具执行在独立的 goroutine + context timeout
- 资源限制（内存、CPU、执行时间）
- 操作审计日志（谁、什么时候、执行了什么工具、结果如何）

### 工作量评估：约 2 天

---

## V3.2 子 Agent 调度与任务拆分

### 为什么做
- 当前是线性流水线，复杂问题拆不开
- 大任务可以拆给多个子 Agent 并行处理
- 更接近真实的 SRE 团队协作模式

### 实现方式

#### Planner 可以拆分任务
```
主 Planner
  ├─ 子任务 A：检查 Redis 状态 → Redis 诊断 Skill
  ├─ 子任务 B：检查网络连通性 → 网络诊断 Skill
  ├─ 子任务 C：检查应用日志 → 日志分析 Skill
  └─ 汇总结果 → 生成最终报告
```

#### 子 Agent 调度器
```go
type TaskSpawner struct {
    skillRegistry *SkillRegistry
    // 并发控制
    maxConcurrent int
    // 任务队列
    taskQueue chan SubTask
    // 结果收集
    results map[string]SubTaskResult
}
```

#### 任务上下文隔离
- 每个子 Agent 有独立的 ContextState
- 父 Agent 汇总子 Agent 的结果
- 支持任务依赖（A 完成后 B 才能开始）

### 工作量评估：约 2 天

---

## V3.3 插件化架构

### 为什么做
- 参考 DeepSeek Harness 的"一切皆插件"理念
- 第三方可以开发自己的工具、Skill、数据源
- 形成生态

### 插件类型
| 插件类型 | 作用 | 示例 |
|---|---|---|
| Tool 插件 | 新增工具 | k8s-tool, mysql-tool |
| Skill 插件 | 新增技能 | k8s-skill, ml-skill |
| LLM 插件 | 新增模型供应商 | deepseek-plugin, qwen-plugin |
| 数据源插件 | 新增告警源 | zabbix-plugin, datadog-plugin |
| 通知插件 | 新增通知渠道 | dingtalk-plugin, feishu-plugin |

### 插件加载机制
```go
type Plugin interface {
    Name() string
    Version() string
    Init(ctx context.Context, cfg map[string]any) error
}

type ToolPlugin interface {
    Plugin
    Tools() []Tool
}

type SkillPlugin interface {
    Plugin
    Skills() []Skill
}
```

支持三种加载方式：
1. **Go 内置插件**（编译进二进制）
2. **RPC 插件**（子进程，通过 gRPC 通信）
3. **WASM 插件**（轻量级，沙箱安全）

### 工作量评估：约 3 天

---

## V3.4 可观测性平台

### 为什么做
- 当前只有 SSE 实时推送，没有历史记录
- 没有统计分析（自愈率、平均处理时间、常见告警类型）
- 没有回放和调试能力

### 实现方式

#### 历史会话存储
- SQLite / PostgreSQL 存储所有 session
- 支持按时间、告警类型、处理状态查询
- 支持回看完整的 ReAct 链路

#### 统计看板
- 自愈率趋势图
- MTTR（平均修复时间）
- 告警类型分布
- 工具调用频率排行
- Agent 成功率统计

#### 调试与回放
- 可以回放某个 session 的完整执行过程
- 可以修改 Prompt 重新运行对比效果
- 可以手动注入故障做回归测试

### 工作量评估：约 2 天

---

## V3 整体工作量

| 模块 | 工作量 | 优先级 |
|---|---|---|
| 安全沙箱与权限 | 2 天 | P0 |
| 子 Agent 调度 | 2 天 | P1 |
| 插件化架构 | 3 天 | P1 |
| 可观测性平台 | 2 天 | P1 |
| **合计** | **约 9 天** | - |

---

# 概念关系总结

## 各概念的层次关系

```
业务场景（SRE 运维）
  │
  ▼
Skill（技能）= Prompt + 工具集 + 工作流 + 知识库
  │
  ├── Prompt（灵魂）：角色设定、思考方式、输出格式
  ├── 工具集（手脚）：通过 MCP 协议调用外部能力
  └── 工作流（骨架）：多步编排、子任务拆分
        │
        ▼
  Harness（马具/框架）= Skill 运行时 + 工具调度 + 安全沙箱 + 插件生态
        │
        ▼
  大模型（大脑）：提供推理能力
```

## 一句话总结

| 概念 | 一句话定位 |
|---|---|
| **Prompt** | Agent 的灵魂，决定怎么想、怎么说 |
| **Tool** | Agent 的手脚，让 Agent 能做事 |
| **MCP** | 工具的标准接口协议，让工具可以跨语言跨系统复用 |
| **Skill** | Prompt + 工具 + 工作流的封装，是可复用的能力单元 |
| **Harness** | Skill 的运行平台，提供安全、调度、插件等基础设施 |
| **Agent** | 模型 + Harness + Skill，能独立完成任务的智能体 |

---

# 版本演进路径建议

## 如果还有 3-5 天
做 **V2 核心**：MCP 工具接入 + Skill 化框架
- 演示时可以说："工具层基于 MCP 协议，Agent 封装为可复用 Skill"

## 如果还有 1-2 周
做 **完整 V2** + V3 的安全沙箱和可观测性
- 演示时可以说："类 Harness 架构，支持插件扩展、权限控制、历史回放"

## 如果只有 1 天
**不要做**，把 V1 跑顺，演讲时讲清楚概念关系就行
- 演示 V1 系统
- 演讲里提 V2/V3 规划
- 展示你对行业趋势的理解
