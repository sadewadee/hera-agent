package builtin
import ("context";"encoding/json";"fmt";"strings";"github.com/sadewadee/hera/internal/tools")
type SkillsGuardTool struct{}
type skillsGuardArgs struct { SkillName string `json:"skill_name"`; Content string `json:"content,omitempty"` }
func (t *SkillsGuardTool) Name() string { return "skills_guard" }
func (t *SkillsGuardTool) Description() string { return "Validates skill safety before execution: checks for dangerous patterns." }
func (t *SkillsGuardTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"skill_name":{"type":"string","description":"Skill name to validate"},"content":{"type":"string","description":"Skill content to check"}},"required":["skill_name"]}`) }
func (t *SkillsGuardTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a skillsGuardArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	dangerous := []string{"rm -rf", "eval(", "exec(", "sudo ", "> /dev/", "chmod 777"}
	for _, p := range dangerous { if strings.Contains(a.Content, p) { return &tools.Result{Content: fmt.Sprintf("BLOCKED: skill '%s' contains dangerous pattern: %s", a.SkillName, p), IsError: true}, nil } }
	return &tools.Result{Content: fmt.Sprintf("Skill '%s' passed safety check", a.SkillName)}, nil
}
func RegisterSkillsGuard(registry *tools.Registry) { registry.Register(&SkillsGuardTool{}) }
