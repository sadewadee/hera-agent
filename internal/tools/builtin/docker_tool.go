package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// DockerTool provides Docker container management operations.
type DockerTool struct{}

type dockerToolArgs struct {
	Action    string   `json:"action"`
	Image     string   `json:"image,omitempty"`
	Container string   `json:"container,omitempty"`
	Command   []string `json:"command,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Env       []string `json:"env,omitempty"`
	Name      string   `json:"name,omitempty"`
}

func (t *DockerTool) Name() string { return "docker" }

func (t *DockerTool) Description() string {
	return "Manages Docker containers and images: run, stop, list, logs, build, exec."
}

func (t *DockerTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["ps", "images", "run", "stop", "rm", "logs", "exec", "build", "pull"],
				"description": "Docker action to perform."
			},
			"image": {"type": "string", "description": "Docker image name."},
			"container": {"type": "string", "description": "Container ID or name."},
			"command": {"type": "array", "items": {"type": "string"}, "description": "Command to run."},
			"ports": {"type": "array", "items": {"type": "string"}, "description": "Port mappings (e.g. 8080:80)."},
			"env": {"type": "array", "items": {"type": "string"}, "description": "Environment variables (KEY=VALUE)."},
			"name": {"type": "string", "description": "Container name."}
		},
		"required": ["action"]
	}`)
}

func (t *DockerTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a dockerToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	runDocker := func(dockerArgs ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	switch a.Action {
	case "ps":
		out, err := runDocker("ps", "--format", "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker ps: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "images":
		out, err := runDocker("images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.Size}}")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker images: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "run":
		if a.Image == "" {
			return &tools.Result{Content: "image is required for run", IsError: true}, nil
		}
		dockerArgs := []string{"run", "-d"}
		if a.Name != "" {
			dockerArgs = append(dockerArgs, "--name", a.Name)
		}
		for _, p := range a.Ports {
			dockerArgs = append(dockerArgs, "-p", p)
		}
		for _, e := range a.Env {
			dockerArgs = append(dockerArgs, "-e", e)
		}
		dockerArgs = append(dockerArgs, a.Image)
		dockerArgs = append(dockerArgs, a.Command...)
		out, err := runDocker(dockerArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker run: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Container started: %s", out)}, nil

	case "stop":
		if a.Container == "" {
			return &tools.Result{Content: "container is required for stop", IsError: true}, nil
		}
		out, err := runDocker("stop", a.Container)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker stop: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Container stopped: %s", out)}, nil

	case "rm":
		if a.Container == "" {
			return &tools.Result{Content: "container is required for rm", IsError: true}, nil
		}
		out, err := runDocker("rm", a.Container)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker rm: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Container removed: %s", out)}, nil

	case "logs":
		if a.Container == "" {
			return &tools.Result{Content: "container is required for logs", IsError: true}, nil
		}
		out, err := runDocker("logs", "--tail", "100", a.Container)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker logs: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "exec":
		if a.Container == "" || len(a.Command) == 0 {
			return &tools.Result{Content: "container and command are required for exec", IsError: true}, nil
		}
		dockerArgs := append([]string{"exec", a.Container}, a.Command...)
		out, err := runDocker(dockerArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker exec: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "pull":
		if a.Image == "" {
			return &tools.Result{Content: "image is required for pull", IsError: true}, nil
		}
		out, err := runDocker("pull", a.Image)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("docker pull: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "build":
		return &tools.Result{Content: "Use the shell tool for docker build with custom Dockerfile paths"}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterDocker registers the docker tool with the given registry.
func RegisterDocker(registry *tools.Registry) {
	registry.Register(&DockerTool{})
}
