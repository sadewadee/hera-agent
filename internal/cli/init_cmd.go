package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/syncer"
	"github.com/spf13/cobra"
)

// initCmd returns the `hera init` subcommand.
//
// hera init is idempotent. It:
//  1. Ensures $HERA_HOME directory tree exists.
//  2. Copies example config to $HERA_HOME/config.yaml (only if absent).
//  3. Copies SOUL.md from bundled configs (only if absent).
//  4. Runs the skills syncer: copies bundled skills → $HERA_HOME/skills/,
//     preserving user-modified files.
//
// It exits cleanly when bundled dir is absent (go install path with no tarball).
func (a *App) initCmd() *cobra.Command {
	var ensure bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize or re-seed $HERA_HOME with bundled skills and example config",
		Long: `hera init sets up your Hera home directory (~/.hera by default, or $HERA_HOME).

It copies bundled skills into $HERA_HOME/skills/ using a manifest-based
copy-on-modify strategy: files you have modified are never overwritten.
Running hera init multiple times is safe (idempotent).

Set HERA_BUNDLED to the directory containing the bundled assets (skills/,
configs/, etc.). If HERA_BUNDLED is not set, hera init attempts to locate
the bundled assets relative to the binary. If neither is found, only the
directory structure is created.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(ensure)
		},
	}

	cmd.Flags().BoolVar(&ensure, "ensure", false,
		"non-interactive mode: exit 0 if already initialised, exit 1 only on real error (for entrypoints)")

	return cmd
}

// runInit is the pure logic for hera init, extracted for testability.
func runInit(ensure bool) error {
	home := paths.HeraHome()

	// 1. Ensure directory structure.
	dirs := []string{
		home,
		paths.UserSkills(),
		paths.UserHooks(),
		paths.UserTools(),
		paths.UserPlugins(),
		paths.UserLogs(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}
	slog.Info("hera init: home directory ready", "path", home)

	// 2. Copy example config if absent.
	if err := seedFileIfAbsent(
		bundledConfigPath("hera.example.yaml"),
		paths.UserConfig(),
		"example config",
	); err != nil {
		slog.Warn("hera init: could not seed config", "err", err)
	}

	// 3. Copy SOUL.md if absent.
	soulDst := filepath.Join(home, "SOUL.md")
	if err := seedFileIfAbsent(
		bundledConfigPath("SOUL.md"),
		soulDst,
		"SOUL.md",
	); err != nil {
		slog.Warn("hera init: could not seed SOUL.md", "err", err)
	}

	// 4. Run skills syncer.
	bundledSkills := paths.BundledSkills()
	if bundledSkills == "" {
		slog.Info("hera init: HERA_BUNDLED not set or no bundled dir found; skipping skills sync")
		fmt.Println("hera init: directory structure created. No bundled skills found (set HERA_BUNDLED to seed skills).")
		return nil
	}
	if _, err := os.Stat(bundledSkills); os.IsNotExist(err) {
		slog.Info("hera init: bundled skills dir not found; skipping sync", "path", bundledSkills)
		fmt.Println("hera init: directory structure created. Bundled skills dir not found.")
		return nil
	}

	s := syncer.New(bundledSkills, paths.UserSkills())
	stats, err := s.Sync()
	if err != nil {
		return fmt.Errorf("skills sync: %w", err)
	}

	fmt.Printf("hera init: skills sync complete — %d copied, %d preserved (user-modified), %d skipped\n",
		stats.Copied, stats.Preserved, stats.Skipped)
	slog.Info("hera init: skills sync complete",
		"copied", stats.Copied,
		"preserved", stats.Preserved,
		"skipped", stats.Skipped)

	// 5. Seed bundled hooks.d example files (copy-if-absent, user edits preserved).
	// This seeds reference YAML files into ~/.hera/hooks.d/ so the user has a
	// template to start from. The watcher picks up any *.yaml files placed there.
	seedHooksDir(paths.BundledHooks(), paths.UserHooks())

	return nil
}

// seedHooksDir copies *.yaml files from bundledDir into userDir, skipping any
// that already exist in userDir (copy-if-absent semantics).
// Errors are non-fatal: logged as warnings so a missing bundled hooks dir does
// not prevent init from completing.
func seedHooksDir(bundledDir, userDir string) {
	if bundledDir == "" {
		return
	}
	entries, err := os.ReadDir(bundledDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("hera init: reading bundled hooks.d", "dir", bundledDir, "err", err)
		}
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		dst := filepath.Join(userDir, name)
		src := filepath.Join(bundledDir, name)
		if err := seedFileIfAbsent(src, dst, "hooks.d/"+name); err != nil {
			slog.Warn("hera init: could not seed hook example", "file", name, "err", err)
		}
	}
}

// bundledConfigPath returns the path for a file in $HERA_BUNDLED/configs/.
// Returns empty string when HeraBundled() is empty.
func bundledConfigPath(filename string) string {
	b := paths.HeraBundled()
	if b == "" {
		return ""
	}
	return filepath.Join(b, "configs", filename)
}

// seedFileIfAbsent copies src to dst only when dst does not exist.
// It is a no-op when src is "" (bundled dir unavailable).
func seedFileIfAbsent(src, dst, label string) error {
	if src == "" {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		// dst already exists — skip.
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			// src bundled file missing — warn but not fatal.
			slog.Debug("hera init: bundled file not found, skipping", "file", label, "src", src)
			return nil
		}
		return fmt.Errorf("read bundled %s: %w", label, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", label, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", label, err)
	}
	slog.Info("hera init: seeded file", "file", label, "dst", dst)
	return nil
}
