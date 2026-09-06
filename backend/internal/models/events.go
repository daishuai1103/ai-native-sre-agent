package models

import (
	"encoding/json"
	"time"
)

// EventType describes the kind of SSE event.
type EventType string

const (
	AlertDetected    EventType = "alert_detected"
	StepStarted      EventType = "step_started"
	ToolCalled       EventType = "tool_called"
	ToolResult       EventType = "tool_result"
	AgentCompleted   EventType = "agent_completed"
	AgentFailed      EventType = "agent_failed"
	ReportGenerated  EventType = "report_generated"
	ReactStep        EventType = "react_step"
)

// StepEvent is sent to the frontend in real-time.
type StepEvent struct {
	Type      EventType      `json:"type"`
	SessionID string         `json:"session_id"`
	Agent     string         `json:"agent,omitempty"`
	Step      string         `json:"step,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

func (e StepEvent) JSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}
