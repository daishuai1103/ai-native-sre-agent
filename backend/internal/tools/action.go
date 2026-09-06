package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/invopop/jsonschema"
	"github.com/wk8/go-ordered-map/v2"
)

// ActionTool executes remediation actions.
type ActionTool struct{}

func (ActionTool) Name() string        { return "execute_action" }
func (ActionTool) Description() string { return "Execute a remediation action." }
func (ActionTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("action_name", &jsonschema.Schema{
		Type:        "string",
		Description: "remediation action name",
		Enum:        []any{"restart_container", "clear_redis_cache"},
	})
	props.Set("container", &jsonschema.Schema{
		Type:        "string",
		Description: "container name or id",
	})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"action_name"},
	}
}

func (ActionTool) Call(ctx context.Context, args map[string]any) (string, error) {
	actionName, _ := args["action_name"].(string)
	container, _ := args["container"].(string)

	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would execute %s on %s", actionName, container), nil
	}

	switch actionName {
	case "restart_container":
		cmd := exec.CommandContext(ctx, "docker", "restart", container)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("restart_container failed: %w, output: %s", err, string(out))
		}
		return string(out), nil
	case "clear_redis_cache":
		cmd := exec.CommandContext(ctx, "docker", "exec", container, "redis-cli", "FLUSHALL")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("clear_redis_cache failed: %w, output: %s", err, string(out))
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unknown action")
	}
}
