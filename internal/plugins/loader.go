package plugins

import (
	"log/slog"
	"path/filepath"

	"github.com/sadewadee/hera/internal/config"
)

// LoadResult holds what was merged from enabled plugins.
// The entrypoint uses these paths to wire skills, hooks, tools, and MCP.
type LoadResult struct {
	// SkillDirs are absolute paths to skills/ dirs from enabled plugins.
	// Pass each to skills.NewLoader() before calling LoadAll().
	SkillDirs []string
	// HookDirs are absolute paths to hooks.d/ dirs from enabled plugins.
	// Start a gateway.HooksWatcher for each dir.
	HookDirs []string
	// ToolDirs are absolute paths to tools.d/ dirs from enabled plugins.
	// Start a builtin.CustomToolWatcher for each dir.
	ToolDirs []string
	// MCPEntries are MCP server declarations from enabled plugins.
	// Append these to cfg.MCPServers before wiring MCP clients.
	MCPEntries []config.MCPServerEntry
}

// LoadEnabled walks the plugins directory, finds enabled plugins, and returns
// a LoadResult describing which directories and MCP entries to wire.
//
// Assets are not loaded here — callers receive directory paths and wire them
// using existing loaders. This keeps LoadEnabled decoupled from I/O and easy
// to test without network or git.
func LoadEnabled(pluginsDir string) LoadResult {
	m := NewManager(pluginsDir)
	infos, err := m.List()
	if err != nil {
		slog.Warn("plugins: could not list plugins", "dir", pluginsDir, "err", err)
		return LoadResult{}
	}

	var result LoadResult
	for _, info := range infos {
		if !info.Enabled {
			slog.Debug("plugins: skipping disabled plugin", "name", info.DirName)
			continue
		}
		if info.Manifest == nil || info.Manifest.Provides == nil {
			slog.Debug("plugins: no provides section in plugin.yaml, skipping", "name", info.DirName)
			continue
		}

		p := info.Manifest.Provides

		if p.Skills != "" {
			dir := filepath.Join(info.Path, filepath.FromSlash(p.Skills))
			result.SkillDirs = append(result.SkillDirs, dir)
			slog.Info("plugins: found skills dir", "plugin", info.DirName, "dir", dir)
		}

		if p.Hooks != "" {
			dir := filepath.Join(info.Path, filepath.FromSlash(p.Hooks))
			result.HookDirs = append(result.HookDirs, dir)
			slog.Info("plugins: found hooks dir", "plugin", info.DirName, "dir", dir)
		}

		if p.Tools != "" {
			dir := filepath.Join(info.Path, filepath.FromSlash(p.Tools))
			result.ToolDirs = append(result.ToolDirs, dir)
			slog.Info("plugins: found tools dir", "plugin", info.DirName, "dir", dir)
		}

		if p.MCP != nil && p.MCP.Command != "" {
			entry := config.MCPServerEntry{
				Name:    info.DirName,
				Command: p.MCP.Command,
				Args:    p.MCP.Args,
				Mode:    p.MCP.Mode,
			}
			result.MCPEntries = append(result.MCPEntries, entry)
			slog.Info("plugins: found MCP server", "plugin", info.DirName, "command", p.MCP.Command)
		}
	}

	return result
}
