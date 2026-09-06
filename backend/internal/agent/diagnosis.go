package agent

import (
	"context"
	"fmt"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// DiagnosisAgent analyzes the root cause from observations and evidence.
type DiagnosisAgent struct{}

func (DiagnosisAgent) Type() AgentType { return Diagnosis }

func (a DiagnosisAgent) Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error {
	prompt := `你是一名 SRE 诊断工程师。根据观测到的数据，分析告警的根因。
如果需要更多证据，可以调用 docker_logs 查看日志或 prometheus_query 查询指标。
分析完成后，用中文输出你的诊断结论，包括可能的原因和判断依据。`

	result, err := ReAct(ctx, a.Type(), state, registry, client, prompt, 5)
	if err != nil {
		return fmt.Errorf("diagnosis failed: %w", err)
	}
	state.Diagnosis = result
	state.Observations = append(state.Observations, result)
	state.AddStep(Step{Agent: string(a.Type()), Type: "final", Content: result})
	return nil
}
