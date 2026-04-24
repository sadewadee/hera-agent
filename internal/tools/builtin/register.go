package builtin

import (
	"os"
	"path/filepath"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// ToolDeps holds references to dependencies needed by built-in tools.
type ToolDeps struct {
	MemoryManager *memory.Manager
	Config        *config.Config
	// SkillGenerator is optional. When set, skill_create calls without a
	// content field route to the generator for LLM-powered skill creation.
	SkillGenerator SkillGenerator
	// CronScheduler is optional. When set, the cronjob tool operates against
	// the live scheduler. When nil, the tool reports that cron is disabled.
	CronScheduler *cron.Scheduler
	// SessionDB is optional. When set, the session_search tool can query
	// past conversation transcripts. When nil, the tool is not registered.
	SessionDB SessionDB
	// Version is the semantic version string used by tools that report
	// a client version to external APIs (e.g. url_safety -> Google Safe
	// Browsing clientVersion). Empty string falls back to "0.0.0".
	Version string
}

// RegisterAll registers all built-in tools with the given registry.
func RegisterAll(registry *tools.Registry, deps ToolDeps) {
	protectedPaths := defaultProtectedPaths()
	dangerousApprove := false
	skillsDir := defaultSkillsDir()

	if deps.Config != nil {
		if len(deps.Config.Security.ProtectedPaths) > 0 {
			protectedPaths = deps.Config.Security.ProtectedPaths
		}
		dangerousApprove = deps.Config.Security.DangerousApprove
	}

	RegisterDatetime(registry)
	RegisterFiles(registry, protectedPaths)
	RegisterShell(registry, protectedPaths, dangerousApprove)
	RegisterSkills(registry, skillsDir, deps.SkillGenerator)
	RegisterWeb(registry, os.Getenv("EXA_API_KEY"), os.Getenv("FIRECRAWL_API_KEY"))
	RegisterClaudeCode(registry)
	RegisterMedia(registry)

	// New tools
	RegisterBrowser(registry)
	RegisterVision(registry)
	RegisterVoice(registry)
	RegisterTerminal(registry)
	RegisterCodeExec(registry)
	RegisterTodo(registry)
	RegisterMCPToolBridge(registry)
	// send_message is wired separately by each entrypoint AFTER the
	// gateway is constructed (gateway typically lives below ToolDeps in
	// main.go setup order). RegisterAll no longer wires it.
	RegisterMixture(registry)
	RegisterRLTraining(registry)
	RegisterURLSafety(registry, deps.Version)
	RegisterPatch(registry)
	RegisterClarify(registry)
	RegisterCheckpoint(registry)
	RegisterInterrupt(registry)
	RegisterFuzzy(registry)
	RegisterApproval(registry)
	RegisterCredential(registry)
	RegisterBudget(registry)
	RegisterDebug(registry)
	RegisterProcess(registry)
	RegisterToolResult(registry)
	RegisterEnvPassthrough(registry)
	RegisterBinaryExt(registry)
	RegisterANSI(registry)
	RegisterOSV(registry)
	RegisterTirith(registry)
	RegisterWebsitePolicy(registry)
	RegisterOpenRouterClient(registry)
	RegisterSkillsGuard(registry)
	RegisterSkillsSync(registry)
	RegisterCronJob(registry, deps.CronScheduler)
	RegisterTranscription(registry)

	// New wave tools
	RegisterScreenshot(registry)
	RegisterPDFReader(registry)
	RegisterCalendar(registry)
	RegisterContacts(registry)
	RegisterNotifications(registry)
	RegisterClipboard(registry)
	RegisterArchive(registry)
	RegisterNetwork(registry)
	RegisterSystemInfo(registry)
	RegisterGit(registry)
	RegisterDocker(registry)
	RegisterK8s(registry)
	RegisterSSH(registry)
	RegisterDatabase(registry)
	RegisterHTTPClient(registry)
	RegisterJSON(registry)
	RegisterCSV(registry)
	RegisterRegex(registry)
	RegisterMath(registry)
	RegisterTranslation(registry)

	RegisterBrowserAutomation(registry)

	// Orphan tools that were previously defined but not wired into RegisterAll.
	RegisterImageGen(registry)
	RegisterNeuTTS(registry)
	RegisterSessionSearch(registry, deps.SessionDB) // nil-safe; registers only when DB is set

	if deps.MemoryManager != nil {
		RegisterMemory(registry, deps.MemoryManager)
		RegisterMemoryNotes(registry, deps.MemoryManager)
		RegisterSessionTools(registry, deps.MemoryManager)
	}
}

func defaultProtectedPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".aws", "credentials"),
	}
}

func defaultSkillsDir() string {
	return paths.UserSkills()
}
