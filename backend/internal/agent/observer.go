package agent

import (
	"context"
	"fmt"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// ObserverAgent discovers anomalies by collecting metrics/logs.
type ObserverAgent struct{}

func (ObserverAgent) Type() AgentType { return Observer }

func (a ObserverAgent) Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error {
	prompt := `你是一名 SRE 观测工程师。收到告警后，请通过调用工具来收集证据。
可用工具：docker_stats（查看容器资源状态）、docker_logs（查看容器日志）、prometheus_query（查询监控指标）。
请务必先调用至少 1 个工具来收集数据，不要直接下结论。
收集完数据后，用中文输出简洁的观测总结。`

	result, err := ReAct(ctx, a.Type(), state, registry, client, prompt, 5)
	if err != nil {
		return fmt.Errorf("observer failed: %w", err)
	}
	state.Observations = append(state.Observations, result)
	state.AddStep(Step{Agent: string(a.Type()), Type: "final", Content: result})
	return nil
}
