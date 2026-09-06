package agent

import (
	"context"
	"fmt"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// ExecutorAgent executes the planned remediation and verifies the result.
type ExecutorAgent struct{}

func (ExecutorAgent) Type() AgentType { return Executor }

func (a ExecutorAgent) Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error {
	prompt := `你是一名 SRE 执行工程师。根据处置计划，调用对应的工具执行修复动作。
执行完成后，调用 docker_stats 工具验证执行结果。
用中文输出你执行了什么动作以及验证结果。`

	if state.Plan != "" {
		state.AddMessage(llm.Message{Role: "assistant", Content: state.Plan})
	}

	result, err := ReAct(ctx, a.Type(), state, registry, client, prompt, 5)
	if err != nil {
		return fmt.Errorf("executor failed: %w", err)
	}
	state.ActionResult = result
	state.AddStep(Step{Agent: string(a.Type()), Type: "final", Content: result})
	return nil
}
