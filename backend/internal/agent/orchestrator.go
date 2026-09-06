package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/models"
	"github.com/ai-native-sre-agent/backend/internal/tools"
)

// EventPublisher is the interface for publishing SSE events.
type EventPublisher interface {
	Publish(event models.StepEvent)
}

// Orchestrator drives the agent workflow.
type Orchestrator struct {
	registry *tools.Registry
	client   llm.Client
	pub      EventPublisher
}

func NewOrchestrator(registry *tools.Registry, client llm.Client, pub EventPublisher) *Orchestrator {
	return &Orchestrator{registry: registry, client: client, pub: pub}
}

func (o *Orchestrator) Run(ctx context.Context, state *ContextState) (*ContextState, error) {
	if state == nil {
		return nil, fmt.Errorf("nil state")
	}

	sessionID := state.Task.SessionID
	log.Printf("[Orchestrator] %s starting workflow with route=%s", sessionID, state.Task.RouteCategory)

	// Set up step callback for real-time SSE broadcast
	if o.pub != nil {
		state.OnStep = func(step Step) {
			// Only broadcast meaningful steps
			if step.Type == "final" || step.Type == "error" || step.Type == "thought" ||
				step.Type == "tool_call" || step.Type == "tool_result" || step.Type == "fallback" {
				o.pub.Publish(models.StepEvent{
					Type:      models.ReactStep,
					SessionID: sessionID,
					Agent:     step.Agent,
					Step:      step.Type,
					Payload: map[string]any{
						"step_type":    step.Type,
						"content":      step.Content,
						"tool_name":    step.ToolName,
						"tool_args":    step.ToolArgs,
						"tool_result":  step.ToolResult,
					},
					Timestamp: time.Now(),
				})
			}
		}
	}

	agents := []Agent{
		ObserverAgent{},
		DiagnosisAgent{},
		PlannerAgent{},
		ExecutorAgent{},
		ReporterAgent{},
	}

	for _, ag := range agents {
		log.Printf("[Orchestrator] %s -> %s: starting", sessionID, ag.Type())
		o.emit(sessionID, ag.Type(), models.StepStarted, string(ag.Type())+" 开始执行")

		if err := ag.Run(ctx, state, o.registry, o.client); err != nil {
			log.Printf("[Orchestrator] %s -> %s: FAILED: %v", sessionID, ag.Type(), err)
			o.emit(sessionID, ag.Type(), models.AgentFailed, "执行失败: "+err.Error())
			return state, fmt.Errorf("%s: %w", ag.Type(), err)
		}

		log.Printf("[Orchestrator] %s -> %s: completed", sessionID, ag.Type())
		o.emit(sessionID, ag.Type(), models.AgentCompleted, string(ag.Type())+" 完成")
	}

	log.Printf("[Orchestrator] %s all agents done, report generated", sessionID)
	o.emit(sessionID, Reporter, models.ReportGenerated, state.Report)

	return state, nil
}

func (o *Orchestrator) emit(sessionID string, agent AgentType, t models.EventType, msg string) {
	if o.pub == nil {
		return
	}
	o.pub.Publish(models.StepEvent{
		Type:      t,
		SessionID: sessionID,
		Agent:     string(agent),
		Step:      msg,
		Payload:   map[string]any{},
		Timestamp: time.Now(),
	})
}
