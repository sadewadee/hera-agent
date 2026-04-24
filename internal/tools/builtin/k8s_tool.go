package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// K8sTool provides Kubernetes management operations via kubectl.
type K8sTool struct{}

type k8sToolArgs struct {
	Action    string `json:"action"`
	Resource  string `json:"resource,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Output    string `json:"output,omitempty"`
	Selector  string `json:"selector,omitempty"`
}

func (t *K8sTool) Name() string { return "kubernetes" }

func (t *K8sTool) Description() string {
	return "Kubernetes management: get, describe, logs, and list resources via kubectl."
}

func (t *K8sTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["get", "describe", "logs", "contexts", "namespaces", "top"],
				"description": "Kubernetes action."
			},
			"resource": {
				"type": "string",
				"description": "Resource type (pods, services, deployments, etc.)."
			},
			"name": {
				"type": "string",
				"description": "Resource name."
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace. Defaults to current context namespace."
			},
			"output": {
				"type": "string",
				"enum": ["wide", "yaml", "json"],
				"description": "Output format."
			},
			"selector": {
				"type": "string",
				"description": "Label selector (e.g. app=nginx)."
			}
		},
		"required": ["action"]
	}`)
}

func (t *K8sTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a k8sToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	runKubectl := func(kubectlArgs ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "kubectl", kubectlArgs...)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	switch a.Action {
	case "get":
		if a.Resource == "" {
			return &tools.Result{Content: "resource type is required for get", IsError: true}, nil
		}
		kubectlArgs := []string{"get", a.Resource}
		if a.Name != "" {
			kubectlArgs = append(kubectlArgs, a.Name)
		}
		if a.Namespace != "" {
			kubectlArgs = append(kubectlArgs, "-n", a.Namespace)
		}
		if a.Output != "" {
			kubectlArgs = append(kubectlArgs, "-o", a.Output)
		}
		if a.Selector != "" {
			kubectlArgs = append(kubectlArgs, "-l", a.Selector)
		}
		out, err := runKubectl(kubectlArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl get: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "describe":
		if a.Resource == "" || a.Name == "" {
			return &tools.Result{Content: "resource and name are required for describe", IsError: true}, nil
		}
		kubectlArgs := []string{"describe", a.Resource, a.Name}
		if a.Namespace != "" {
			kubectlArgs = append(kubectlArgs, "-n", a.Namespace)
		}
		out, err := runKubectl(kubectlArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl describe: %s %v", out, err), IsError: true}, nil
		}
		// Truncate large descriptions
		if len(out) > 50*1024 {
			out = out[:50*1024] + "\n...[truncated]"
		}
		return &tools.Result{Content: out}, nil

	case "logs":
		if a.Name == "" {
			return &tools.Result{Content: "pod name is required for logs", IsError: true}, nil
		}
		kubectlArgs := []string{"logs", "--tail=100", a.Name}
		if a.Namespace != "" {
			kubectlArgs = append(kubectlArgs, "-n", a.Namespace)
		}
		out, err := runKubectl(kubectlArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl logs: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "contexts":
		out, err := runKubectl("config", "get-contexts")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl contexts: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "namespaces":
		out, err := runKubectl("get", "namespaces")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl namespaces: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "top":
		resource := a.Resource
		if resource == "" {
			resource = "pods"
		}
		kubectlArgs := []string{"top", resource}
		if a.Namespace != "" {
			kubectlArgs = append(kubectlArgs, "-n", a.Namespace)
		}
		out, err := runKubectl(kubectlArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("kubectl top: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterK8s registers the Kubernetes tool with the given registry.
func RegisterK8s(registry *tools.Registry) {
	registry.Register(&K8sTool{})
}
