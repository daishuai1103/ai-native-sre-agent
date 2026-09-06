package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/invopop/jsonschema"
	"github.com/wk8/go-ordered-map/v2"
)

// DockerUpdateLimitTool adjusts container resource limits (memory/CPU).
type DockerUpdateLimitTool struct{}

func (DockerUpdateLimitTool) Name() string { return "docker_update_limit" }
func (DockerUpdateLimitTool) Description() string {
	return "Update resource limits (memory and/or CPU) for a running Docker container. " +
		"Use when a container is hitting memory or CPU limits and needs more resources."
}
func (DockerUpdateLimitTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	props.Set("memory", &jsonschema.Schema{Type: "string", Description: "memory limit, e.g. '512m', '1g'. Empty means no change."})
	props.Set("cpus", &jsonschema.Schema{Type: "number", Description: "CPU limit in number of cores, e.g. 1.5. 0 means no change."})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"container"},
	}
}
func (DockerUpdateLimitTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	memory, _ := args["memory"].(string)
	cpus := 0.0
	if v, ok := args["cpus"]; ok {
		switch n := v.(type) {
		case float64:
			cpus = n
		case int:
			cpus = float64(n)
		}
	}

	if memory == "" && cpus == 0 {
		return "", fmt.Errorf("at least one of memory or cpus must be specified")
	}

	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would update container %s: memory=%s cpus=%.1f", container, memory, cpus), nil
	}

	cmdArgs := []string{"update"}
	if memory != "" {
		cmdArgs = append(cmdArgs, "--memory", memory)
	}
	if cpus > 0 {
		cmdArgs = append(cmdArgs, "--cpus", strconv.FormatFloat(cpus, 'f', 1, 64))
	}
	cmdArgs = append(cmdArgs, container)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker update failed: %w, output: %s", err, string(out))
	}
	return fmt.Sprintf("updated container %s limits: %s", container, string(out)), nil
}

// DockerPruneTool cleans up unused Docker resources to free disk space.
type DockerPruneTool struct{}

func (DockerPruneTool) Name() string { return "docker_prune" }
func (DockerPruneTool) Description() string {
	return "Remove unused Docker images, containers, volumes, and networks to free disk space. " +
		"Use when disk space is low and there are dangling/unused resources."
}
func (DockerPruneTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("type", &jsonschema.Schema{
		Type:        "string",
		Description: "what to prune: 'system' (all unused), 'images', 'volumes', 'containers'",
		Default:     "system",
	})
	props.Set("prune_volumes", &jsonschema.Schema{
		Type:        "boolean",
		Description: "whether to also prune volumes (dangerous, may lose data)",
		Default:     false,
	})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
	}
}
func (DockerPruneTool) Call(ctx context.Context, args map[string]any) (string, error) {
	pruneType := "system"
	if v, ok := args["type"].(string); ok && v != "" {
		pruneType = v
	}
	pruneVolumes := false
	if v, ok := args["prune_volumes"].(bool); ok {
		pruneVolumes = v
	}

	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would run docker %s prune (volumes=%v)", pruneType, pruneVolumes), nil
	}

	var cmd *exec.Cmd
	switch pruneType {
	case "system":
		args := []string{"system", "prune", "-f"}
		if pruneVolumes {
			args = append(args, "--volumes")
		}
		cmd = exec.CommandContext(ctx, "docker", args...)
	case "images":
		cmd = exec.CommandContext(ctx, "docker", "image", "prune", "-f")
	case "volumes":
		cmd = exec.CommandContext(ctx, "docker", "volume", "prune", "-f")
	case "containers":
		cmd = exec.CommandContext(ctx, "docker", "container", "prune", "-f")
	default:
		return "", fmt.Errorf("unknown prune type: %s", pruneType)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker prune failed: %w, output: %s", err, string(out))
	}
	return string(out), nil
}

// DockerExecTool runs a command inside a container.
type DockerExecTool struct{}

func (DockerExecTool) Name() string { return "docker_exec" }
func (DockerExecTool) Description() string {
	return "Execute a command inside a running Docker container. " +
		"Use for diagnostic commands or service restarts inside the container."
}
func (DockerExecTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	props.Set("command", &jsonschema.Schema{Type: "string", Description: "command to run inside the container"})
	props.Set("timeout_seconds", &jsonschema.Schema{
		Type:        "integer",
		Description: "command timeout in seconds",
		Default:     30,
	})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"container", "command"},
	}
}
func (DockerExecTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	command, _ := args["command"].(string)

	if container == "" || command == "" {
		return "", fmt.Errorf("container and command are required")
	}

	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would exec in %s: %s", container, command), nil
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec failed: %w, output: %s", err, string(out))
	}
	return string(out), nil
}

// ServiceRestartTool restarts a service inside a container (e.g. nginx, redis).
type ServiceRestartTool struct{}

func (ServiceRestartTool) Name() string { return "service_restart" }
func (ServiceRestartTool) Description() string {
	return "Restart a service process inside a container (e.g. nginx, redis, netbox). " +
		"Safer than restarting the entire container as it preserves the container state."
}
func (ServiceRestartTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("container", &jsonschema.Schema{Type: "string", Description: "container name or id"})
	props.Set("service", &jsonschema.Schema{
		Type:        "string",
		Description: "service name to restart, e.g. 'nginx', 'redis', 'netbox'",
	})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"container", "service"},
	}
}
func (ServiceRestartTool) Call(ctx context.Context, args map[string]any) (string, error) {
	container, _ := args["container"].(string)
	service, _ := args["service"].(string)

	if container == "" || service == "" {
		return "", fmt.Errorf("container and service are required")
	}

	if os.Getenv("AGENT_DRY_RUN") == "true" {
		return fmt.Sprintf("[DRY-RUN] would restart service %s in container %s", service, container), nil
	}

	// Try multiple restart approaches
	commands := []string{
		// systemctl (if systemd is available)
		fmt.Sprintf("systemctl restart %s 2>/dev/null", service),
		// service command
		fmt.Sprintf("service %s restart 2>/dev/null", service),
		// supervisorctl
		fmt.Sprintf("supervisorctl restart %s 2>/dev/null", service),
		// s6 (common in Alpine-based containers)
		fmt.Sprintf("s6-svc -r /etc/s6-overlay/s6-rc.d/%s 2>/dev/null", service),
		// kill and let container restart the process (last resort)
		fmt.Sprintf("pkill -HUP %s 2>/dev/null || true", service),
	}

	var lastOut string
	var lastErr error
	for _, cmd := range commands {
		c := exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", cmd)
		out, err := c.CombinedOutput()
		lastOut = string(out)
		lastErr = err
		if err == nil && len(out) > 0 {
			return fmt.Sprintf("restarted %s in %s via service manager: %s", service, container, lastOut), nil
		}
	}

	return fmt.Sprintf("tried all restart methods for %s in %s, last output: %s, last error: %v",
		service, container, lastOut, lastErr), nil
}
