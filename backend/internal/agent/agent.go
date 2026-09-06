package agent

import (
	"context"
	"time"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// AgentType identifies the role of an agent.
type AgentType string

const (
	Observer  AgentType = "observer"
	Diagnosis AgentType = "diagnosis"
	Planner   AgentType = "planner"
	Executor  AgentType = "executor"
	Reporter  AgentType = "reporter"
)

// Task is the input to an agent workflow.
type Task struct {
	Alert           string  `json:"alert"`
	AlertType       string  `json:"alert_type"`
	SessionID       string  `json:"session_id"`
	RouteCategory   string  `json:"route_category,omitempty"`
	RouteSource     string  `json:"route_source,omitempty"`
	RouteConfidence float64 `json:"route_confidence,omitempty"`
}

// Step represents one thought/action in the agent trace.
type Step struct {
	Agent      string `json:"agent"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolArgs   string `json:"tool_args,omitempty"`
	ToolResult string `json:"tool_result,omitempty"`
}

// ContextState holds the shared memory across agents.
type ContextState struct {
	Task         Task          `json:"task"`
	Messages     []llm.Message `json:"messages"`
	Observations []string      `json:"observations"`
	Diagnosis    string        `json:"diagnosis"`
	Plan         string        `json:"plan"`
	ActionResult string        `json:"action_result"`
	Report       string        `json:"report"`
	Steps        []Step        `json:"steps"`
	StartedAt    time.Time     `json:"started_at,omitempty"`
	FinishedAt   time.Time     `json:"finished_at,omitempty"`
	// OnStep is called for each new step (used for SSE broadcast).
	OnStep func(step Step) `json:"-"`
}

// Agent is implemented by all specialist agents.
type Agent interface {
	Type() AgentType
	Run(ctx context.Context, state *ContextState, registry *tools.Registry, client llm.Client) error
}

func (s *ContextState) AddStep(st Step) {
	s.Steps = append(s.Steps, st)
	if s.OnStep != nil {
		s.OnStep(st)
	}
}

func (s *ContextState) AddMessage(msg llm.Message) {
	s.Messages = append(s.Messages, msg)
}
