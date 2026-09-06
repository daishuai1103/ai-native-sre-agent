package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/invopop/jsonschema"
	"github.com/wk8/go-ordered-map/v2"
)

// DockerStatsTool returns container resource usage.
type DockerStatsTool struct{}

func (DockerStatsTool) Name() string { return "docker_stats" }
func (DockerStatsTool) Description() string {
	return "Get CPU and memory stats for a Docker container."
}
func (DockerStatsTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"container"},
	}
}
func (DockerStatsTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	cmd := exec.CommandContext(ctx, "docker", "stats", container, "--no-stream", "--format", "{{.Name}}: CPU {{.CPUPerc}} MEM {{.MemUsage}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker stats failed: %w, output: %s", err, string(out))
	}
	return string(out), nil
}

// DockerRestartTool restarts a container (respects dry-run).
type DockerRestartTool struct{}

func (DockerRestartTool) Name() string        { return "docker_restart" }
func (DockerRestartTool) Description() string { return "Restart a Docker container." }
func (DockerRestartTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"container"},
	}
}
func (DockerRestartTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would restart container %s", container), nil
	}
	cmd := exec.CommandContext(ctx, "docker", "restart", container)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker restart failed: %w, output: %s", err, string(out))
	}
	return fmt.Sprintf("restarted %s: %s", container, string(out)), nil
}

// DockerLogsTool returns recent logs from a container.
type DockerLogsTool struct{}

func (DockerLogsTool) Name() string        { return "docker_logs" }
func (DockerLogsTool) Description() string { return "Get recent logs from a Docker container." }
func (DockerLogsTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	props.Set("lines", &jsonschema.Schema{Type: "integer", Description: "number of log lines to return", Default: 100})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
	}
}
func (DockerLogsTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	lines := 100
	if v, ok := args["lines"]; ok {
		switch n := v.(type) {
		case int:
			lines = n
		case int8:
			lines = int(n)
		case int16:
			lines = int(n)
		case int32:
			lines = int(n)
		case int64:
			lines = int(n)
		case float64:
			lines = int(n)
		}
	}
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", lines), container)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w, output: %s", err, string(out))
	}
	return string(out), nil
}

// DockerPsTool lists running containers.
type DockerPsTool struct{}

func (DockerPsTool) Name() string        { return "docker_ps" }
func (DockerPsTool) Description() string { return "List running Docker containers." }
func (DockerPsTool) Schema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
	}
}
func (DockerPsTool) Call(ctx context.Context, args map[string]any) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}} {{.Status}} {{.Image}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker ps failed: %w, output: %s", err, string(out))
	}
	return string(out), nil
}
