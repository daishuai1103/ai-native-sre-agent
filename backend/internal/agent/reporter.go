package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// ReporterAgent summarizes the incident end to end.
type ReporterAgent struct{}

func (ReporterAgent) Type() AgentType { return Reporter }

func (a ReporterAgent) Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error {
	prompt := `你是一名 SRE 报告撰写工程师。请根据完整的处理过程，生成一份结构化的中文报告。
报告必须包含以下 Markdown 章节：
## 异常概述
## 诊断分析
## 处置动作
## 执行结果
## 后续建议

请用简洁专业的语言撰写。`

	state.AddMessage(llm.Message{Role: "user", Content: fmt.Sprintf("Alert: %s", state.Task.Alert)})
	if len(state.Observations) > 0 {
		state.AddMessage(llm.Message{Role: "user", Content: fmt.Sprintf("Observations:\n%s", strings.Join(state.Observations, "\n"))})
	}
	if state.Diagnosis != "" {
		state.AddMessage(llm.Message{Role: "user", Content: fmt.Sprintf("Diagnosis: %s", state.Diagnosis)})
	}
	if state.Plan != "" {
		state.AddMessage(llm.Message{Role: "user", Content: fmt.Sprintf("Plan: %s", state.Plan)})
	}
	if state.ActionResult != "" {
		state.AddMessage(llm.Message{Role: "user", Content: fmt.Sprintf("ActionResult: %s", state.ActionResult)})
	}

	result, err := ReAct(ctx, a.Type(), state, registry, client, prompt, 2)
	if err != nil {
		return fmt.Errorf("reporter failed: %w", err)
	}
	state.Report = result
	state.AddStep(Step{Agent: string(a.Type()), Type: "final", Content: result})
	return nil
}
