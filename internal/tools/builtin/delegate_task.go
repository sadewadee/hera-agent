package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// DelegateTaskRunner is the minimal interface required by DelegateTaskTool to
// invoke a named sub-agent. Implemented by *agent.AgentRegistry.
type DelegateTaskRunner interface {
	DelegateTo(ctx context.Context, targetName, prompt string) (string, error)
}

// DelegationObserver is an optional interface for observing delegation events.
// Implemented by *gateway.AgentBus via its Publish method. Pass nil to skip.
type DelegationObserver interface {
	// PublishDelegation notifies observers that a delegation occurred.
	// callerAgent is the delegating agent name, targetAgent is the target.
	PublishDelegation(callerAgent, targetAgent, payload string)
}

// DelegateTaskTool is a tools.Tool that lets one agent hand a task to another
// agent by name. It requires a DelegateTaskRunner (typically an AgentRegistry)
// to be injected at construction time. An optional DelegationObserver receives
// delegation notifications for audit/observability.
type DelegateTaskTool struct {
	registry   DelegateTaskRunner
	observer   DelegationObserver
	callerName string
}

// NewDelegateTaskTool returns a DelegateTaskTool backed by the given registry.
// If registry is nil the tool returns an explicit error on every Execute call.
// observer may be nil — if provided, delegations are posted to it.
func NewDelegateTaskTool(registry DelegateTaskRunner) *DelegateTaskTool {
	return &DelegateTaskTool{registry: registry}
}

// WithObserver attaches a DelegationObserver to the tool. Returns the same
// tool instance for fluent chaining.
func (t *DelegateTaskTool) WithObserver(obs DelegationObserver) *DelegateTaskTool {
	t.observer = obs
	return t
}

// WithCallerName sets the name of the agent that owns this tool instance.
// The name appears in AgentBus delegation events as the From field.
// Defaults to "unknown" if not set.
func (t *DelegateTaskTool) WithCallerName(name string) *DelegateTaskTool {
	t.callerName = name
	return t
}

type delegateTaskArgs struct {
	Agent   string `json:"agent"`
	Task    string `json:"task"`
	Context string `json:"context,omitempty"`
}

func (t *DelegateTaskTool) Name() string { return "delegate_task" }

func (t *DelegateTaskTool) Description() string {
	return "Delegate a task to another named agent. The target agent runs independently and returns its response."
}

func (t *DelegateTaskTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"agent": {
				"type": "string",
				"description": "Name of the target agent to delegate to"
			},
			"task": {
				"type": "string",
				"description": "Task description or prompt to give the target agent"
			},
			"context": {
				"type": "string",
				"description": "Optional additional context to include with the task"
			}
		},
		"required": ["agent", "task"]
	}`)
}

func (t *DelegateTaskTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	if t.registry == nil {
		return &tools.Result{
			Content: "delegate_task: no agent registry configured",
			IsError: true,
		}, nil
	}

	var a delegateTaskArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{
			Content: fmt.Sprintf("delegate_task: invalid args: %v", err),
			IsError: true,
		}, nil
	}

	if strings.TrimSpace(a.Agent) == "" {
		return &tools.Result{Content: "delegate_task: agent name is required", IsError: true}, nil
	}
	if strings.TrimSpace(a.Task) == "" {
		return &tools.Result{Content: "delegate_task: task is required", IsError: true}, nil
	}

	// Build the full prompt: task + optional context.
	prompt := a.Task
	if strings.TrimSpace(a.Context) != "" {
		prompt = a.Task + "\n\nContext: " + a.Context
	}

	// Notify observers before invoking the target (fire-and-forget on bus).
	if t.observer != nil {
		callerID := t.callerName
		if callerID == "" {
			callerID = "unknown"
		}
		t.observer.PublishDelegation(callerID, a.Agent, fmt.Sprintf("delegated: %s", a.Task))
	}

	resp, err := t.registry.DelegateTo(ctx, a.Agent, prompt)
	if err != nil {
		return &tools.Result{
			Content: fmt.Sprintf("delegate_task: agent %q returned error: %v", a.Agent, err),
			IsError: true,
		}, nil
	}

	return &tools.Result{
		Content: fmt.Sprintf("[Agent %q response]\n%s", a.Agent, resp),
	}, nil
}

// RegisterDelegateTask registers the delegate_task tool with the given registry.
// If delegateRunner is nil, the registered tool will return an error on every call.
func RegisterDelegateTask(registry *tools.Registry, delegateRunner DelegateTaskRunner) {
	registry.Register(NewDelegateTaskTool(delegateRunner))
}
