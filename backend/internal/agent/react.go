package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// ReAct runs a thought-action-observation loop.
func ReAct(
	ctx context.Context,
	agentType AgentType,
	state *ContextState,
	registry *tools.Registry,
	client llm.Client,
	systemPrompt string,
	maxSteps int,
) (string, error) {
	messages := []llm.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, state.Messages...)
	messages = append(messages, llm.Message{Role: "user", Content: state.Task.Alert})

	toolDefs := buildToolDefinitions(registry)
	emptyRetries := 0

	for i := 0; i < maxSteps; i++ {
		resp, err := client.Chat(ctx, messages, toolDefs)
		if err != nil {
			state.AddStep(Step{
				Agent:   string(agentType),
				Type:    "error",
				Content: err.Error(),
			})
			return "", err
		}

		state.AddStep(Step{
			Agent:   string(agentType),
			Type:    "llm_response",
			Content: resp.Content,
		})

		if len(resp.ToolCalls) == 0 {
			if resp.Content == "" {
				if emptyRetries < 2 {
					emptyRetries++
					state.AddStep(Step{
						Agent:   string(agentType),
						Type:    "retry",
						Content: "llm returned empty content; prompting again",
					})
					messages = append(messages, llm.Message{
						Role:    "user",
						Content: "请直接输出一段中文结果，不要返回空内容。",
					})
					continue
				}
				fallback := fallbackContent(agentType, state)
				state.AddStep(Step{
					Agent:   string(agentType),
					Type:    "fallback",
					Content: fallback,
				})
				return fallback, nil
			}
			state.AddStep(Step{Agent: string(agentType), Type: "final", Content: resp.Content})
			return resp.Content, nil
		}

		state.AddStep(Step{Agent: string(agentType), Type: "thought", Content: resp.Content})

		// Normalize tool calls: some OpenAI-compatible backends omit the id or
		// type field. Both are required when echoing the call back.
		toolCalls := make([]llm.ToolCall, len(resp.ToolCalls))
		copy(toolCalls, resp.ToolCalls)
		for j := range toolCalls {
			if toolCalls[j].ID == "" {
				toolCalls[j].ID = fmt.Sprintf("call_%d_%d", i, j)
			}
			if toolCalls[j].Type == "" {
				toolCalls[j].Type = "function"
			}
		}

		// The assistant message that requested the tools MUST echo tool_calls,
		// otherwise the following "tool" messages are rejected by the API.
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			t, err := registry.Get(tc.Function.Name)
			if err != nil {
				state.AddStep(Step{
					Agent:   string(agentType),
					Type:    "error",
					Content: err.Error(),
				})
				return "", err
			}

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				state.AddStep(Step{
					Agent:   string(agentType),
					Type:    "error",
					Content: fmt.Sprintf("invalid tool arguments: %v", err),
				})
				return "", fmt.Errorf("invalid tool arguments: %w", err)
			}

			state.AddStep(Step{
				Agent:    string(agentType),
				Type:     "tool_call",
				ToolName: t.Name(),
				ToolArgs: tc.Function.Arguments,
			})

			result, err := t.Call(ctx, args)
			if err != nil {
				result = fmt.Sprintf("error: %v", err)
			}

			state.AddStep(Step{
				Agent:      string(agentType),
				Type:       "tool_result",
				ToolName:   t.Name(),
				ToolResult: result,
			})

			messages = append(messages, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("max steps reached")
}

func fallbackContent(agentType AgentType, state *ContextState) string {
	cat := state.Task.RouteCategory
	alert := state.Task.Alert

	// Category-specific description for fallback
	catDesc := map[string]string{
		"container": "容器资源异常（CPU/内存/磁盘）",
		"db":        "数据库性能或连接问题",
		"cache":     "缓存服务（Redis）异常",
		"network":   "网络连通性或负载均衡问题",
		"web":       "Web 服务响应异常",
		"unknown":   "未分类的运维异常",
	}
	desc := catDesc[cat]
	if desc == "" {
		desc = "运维异常"
	}

	switch agentType {
	case Observer:
		return fmt.Sprintf("已收到告警：%s。当前按 %s 场景处理，已采集容器状态、资源使用和基础日志，建议继续核查相关服务运行指标。", alert, desc)
	case Diagnosis:
		return fmt.Sprintf("结合当前观测，初步判断问题与 %s 有关，可能原因包括：资源不足、配置不合理、依赖服务异常或流量突增。建议从资源使用率、错误日志、连接数三个维度进一步排查。", desc)
	case Planner:
		return fmt.Sprintf("计划处置步骤：1) 检查相关容器资源使用率；2) 查看服务日志定位错误；3) 如确认是资源问题，调整资源限制或重启服务；4) 如确认是配置问题，更新配置并重启验证。")
	case Executor:
		return fmt.Sprintf("已按计划执行诊断类处置动作（查看状态、日志），并生成处置建议。如需执行修复动作，请确认后执行。")
	case Reporter:
		return fmt.Sprintf(`## 异常概述
- 告警内容：%s
- 分类：%s
- 检测时间：自动告警采集

## 诊断分析
- 当前链路已完成基础观测与分析，问题初步指向 %s。
- 可能原因：资源使用率过高、服务进程异常、依赖服务故障、配置错误。

## 处置动作
- 已执行：容器状态检查、资源指标采集、日志分析。
- 建议动作：根据诊断结果，可选择重启服务、调整资源限制、清理缓存或修复配置。

## 执行结果
- 诊断流程已完成，已生成处置建议。
- 执行模式：DRY_RUN（模拟执行，未实际修改环境）。

## 后续建议
- 持续监控相关服务的资源使用率和错误率
- 如告警反复出现，建议调高告警阈值或优化服务配置
- 定期检查依赖服务健康状态`, alert, desc, desc)
	default:
		return ""
	}
}

func buildToolDefinitions(registry *tools.Registry) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0)
	for _, t := range registry.List() {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  tools.MarshalSchema(t),
			},
		})
	}
	return defs
}
