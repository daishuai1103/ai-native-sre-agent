package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Tool is the interface that all callable tools must implement.
type Tool interface {
	Name() string
	Description() string
	Schema() *jsonschema.Schema
	Call(ctx context.Context, args map[string]any) (string, error)
}

// Registry stores and retrieves tools by name.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t, nil
}

func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// MarshalSchema returns the JSON schema of a tool as a map.
func MarshalSchema(t Tool) map[string]any {
	b, _ := json.Marshal(t.Schema())
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
