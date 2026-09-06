# AI Native SRE Agent 架构设计

## 1. 整体目标

以 `netbox-community/netbox-docker` 为数据来源，构建一个具备 Tool Calling 结构的 Agent，完成 **发现异常 → 给出诊断建议 → 采取处置动作** 的完整闭环。

## 2. 核心流程

```
Alert Input
    ↓
Intelligent Router (规则 + LightGBM/ONNX)
    ↓
Agent Orchestrator
    ↓
Observer Agent → Diagnosis Agent → Planner Agent → Executor Agent → Reporter Agent
    ↑                      ↑                      ↑
Tool Registry  ←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←
    ↑
netbox-docker (Docker / Prometheus / NetBox API)
```

## 3. Agent 职责

| Agent | 职责 | 典型 Tool |
|---|---|---|
| Observer | 发现异常，收集关键指标 | docker_stats, prometheus_query |
| Diagnosis | 分析根因 | docker_logs, prometheus_query, llm_call |
| Planner | 制定处置计划 | 无外部工具，纯推理 |
| Executor | 执行处置动作 | action_tool |
| Reporter | 生成中文报告 | llm_call |

## 4. Tool Calling 机制

每个 Agent 内部运行一个 ReAct Loop：

```
Observation → Thought → Action (Tool Call) → Observation → ... → Final Answer
```

LLM 根据上下文决定调用哪个 Tool，Agent Orchestrator 解析 `tool_calls`，执行 Tool，回填结果。

## 5. 智能路由

- **规则路由**: 基于关键字匹配告警类型（container/db/cache/network/web）。
- **ONNX 路由**: 使用 LightGBM 训练的分类器，导出 ONNX 后由 Go 加载推理。
- **混合策略**: ONNX 置信度低时回退到规则路由。

## 6. 前端可视化

- Chatbot UI：输入告警、展示报告。
- Trace Panel：实时展示每个 Agent 的执行步骤、Tool Call 参数与结果。
- Router Card：展示路由决策与置信度。

## 7. 数据来源

`netbox-docker` 启动后暴露：

- Docker API：容器状态、日志、重启
- Prometheus：redis/postgres/netbox metrics
- NetBox API：设备/站点信息（可选）
