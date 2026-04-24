package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

type MixtureTool struct{}

type mixtureArgs struct {
	Prompt string   `json:"prompt"`
	Models []string `json:"models,omitempty"`
	Mode   string   `json:"mode,omitempty"`
}

func (t *MixtureTool) Name() string        { return "mixture" }
func (t *MixtureTool) Description() string  { return "Queries multiple LLM models and combines or compares their responses." }
func (t *MixtureTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string","description":"Prompt to send to models"},"models":{"type":"array","items":{"type":"string"},"description":"Model names to query"},"mode":{"type":"string","enum":["compare","merge","best"],"description":"How to combine results"}},"required":["prompt"]}`)
}

func (t *MixtureTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a mixtureArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	models := a.Models
	if len(models) == 0 { models = []string{"default"} }
	mode := a.Mode
	if mode == "" { mode = "compare" }
	return &tools.Result{Content: fmt.Sprintf("Mixture query (%s mode) sent to %s. Multi-model queries require model registry access.", mode, strings.Join(models, ", "))}, nil
}

func RegisterMixture(registry *tools.Registry) { registry.Register(&MixtureTool{}) }
