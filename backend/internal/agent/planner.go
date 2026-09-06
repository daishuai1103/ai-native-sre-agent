package agent

import (
	"context"
	"fmt"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// PlannerAgent decides the remediation action from the diagnosis.
type PlannerAgent struct{}

func (PlannerAgent) Type() AgentType { return Planner }

func (a PlannerAgent) Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error {
	prompt := `你是一名 SRE 规划工程师。根据诊断结论，制定处置计划。
可用工具：docker_restart（重启容器）、docker_update_limit（调整资源限制）、docker_prune（清理资源）、service_restart（重启服务）。
请选择合适的工具并制定执行步骤，用中文输出你的处置计划。`

	if state.Diagnosis != "" {
		state.AddMessage(llm.Message{Role: "assistant", Content: state.Diagnosis})
	}

	result, err := ReAct(ctx, a.Type(), state, registry, client, prompt, 3)
	if err != nil {
		return fmt.Errorf("planner failed: %w", err)
	}
	state.Plan = result
	state.AddStep(Step{Agent: string(a.Type()), Type: "final", Content: result})
	return nil
}
