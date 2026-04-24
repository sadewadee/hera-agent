// Package paths provides the single source of truth for all Hera filesystem
// paths. It respects HERA_HOME and HERA_BUNDLED environment variables, with
// sensible fallbacks for both installed-binary and go-install users.
//
// Every function re-reads environment variables on each call so tests can
// override them with t.Setenv without needing any global state reset.
package paths

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// HeraHome returns the user's Hera home directory.
//
// Priority:
//  1. $HERA_HOME environment variable (if non-empty)
//  2. ~/.hera (os.UserHomeDir + "/.hera")
func HeraHome() string {
	if h := os.Getenv("HERA_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hera")
}

// HeraBundled returns the read-only reference tree shipped with the binary.
//
// Priority:
//  1. $HERA_BUNDLED environment variable (if non-empty)
//  2. <binary-dir>/../share/hera (binary-relative, resolves symlinks)
//
// Returns "" when the environment variable is unset AND os.Executable fails.
// Callers must check os.Stat on the result before reading from it — the path
// may exist or not depending on the install method.
func HeraBundled() string {
	if b := os.Getenv("HERA_BUNDLED"); b != "" {
		return b
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks so that a symlinked binary finds the real binary's
	// adjacent share/ directory, not the symlink directory.
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe
	}
	binDir := filepath.Dir(real)
	return filepath.Join(binDir, "..", "share", "hera")
}

// UserSkills returns the directory where skills synced from the bundle live.
// Path: HeraHome()/skills
func UserSkills() string {
	return filepath.Join(HeraHome(), "skills")
}

// BundledSkills returns the read-only bundled skills directory.
// Returns "" when HeraBundled() returns "".
// Path: HeraBundled()/skills
func BundledSkills() string {
	b := HeraBundled()
	if b == "" {
		return ""
	}
	return filepath.Join(b, "skills")
}

// UserHooks returns the user's hooks directory.
// Path: HeraHome()/hooks.d
func UserHooks() string {
	return filepath.Join(HeraHome(), "hooks.d")
}

// BundledHooks returns the bundled example hooks directory.
// Returns "" when HeraBundled() returns "".
// Path: HeraBundled()/hooks.d
func BundledHooks() string {
	b := HeraBundled()
	if b == "" {
		return ""
	}
	return filepath.Join(b, "hooks.d")
}

// UserTools returns the user's custom tools directory.
// Path: HeraHome()/tools.d
func UserTools() string {
	return filepath.Join(HeraHome(), "tools.d")
}

// BundledTools returns the bundled example tools directory.
// Returns "" when HeraBundled() returns "".
// Path: HeraBundled()/tools.d
func BundledTools() string {
	b := HeraBundled()
	if b == "" {
		return ""
	}
	return filepath.Join(b, "tools.d")
}

// UserPlugins returns the directory where git-installed user plugins live.
// Path: HeraHome()/plugins
func UserPlugins() string {
	return filepath.Join(HeraHome(), "plugins")
}

// UserPersonalities returns the user's personality overrides directory.
// Path: HeraHome()/personalities
func UserPersonalities() string {
	return filepath.Join(HeraHome(), "personalities")
}

// UserDB returns the default SQLite database path.
// Path: HeraHome()/hera.db
func UserDB() string {
	return filepath.Join(HeraHome(), "hera.db")
}

// UserLogs returns the log directory.
// Path: HeraHome()/logs
func UserLogs() string {
	return filepath.Join(HeraHome(), "logs")
}

// UserConfig returns the user's config.yaml path.
// Path: HeraHome()/config.yaml
func UserConfig() string {
	return filepath.Join(HeraHome(), "config.yaml")
}

// Normalize resolves a user-supplied path for file-I/O tools so that
// "~", "$HERA_HOME", and ".hera/" all map to the right destination
// regardless of the process's current working directory.
//
// Rules (checked in order):
//
//   - ""                                    -> "" (no-op)
//   - "~"                                   -> $HOME
//   - "~/<rest>" or "~\<rest>"              -> $HOME/<rest>
//   - "$HERA_HOME/<rest>" / "${HERA_HOME}/<rest>"
//     -> HeraHome()/<rest>
//   - ".hera" / ".hera/<rest>" / ".hera\<rest>"
//     -> HeraHome()/<rest> (logged)
//   - absolute path                         -> as-is
//   - other relative path                   -> filepath.Abs (CWD-relative)
//
// The ".hera/…" redirect is a safety net: the LLM frequently emits that
// prefix thinking "the Hera home folder", but filepath.Abs would resolve
// it against CWD. We log at Info level so the user can see the redirect.
func Normalize(p string) string {
	if p == "" {
		return ""
	}

	// Tilde expansion (uses os.UserHomeDir; no shell required).
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(filepath.Separator)) {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
		return p
	}

	// $HERA_HOME / ${HERA_HOME} expansion. Only handle the exact prefix
	// so we don't accidentally mangle values that happen to contain
	// "$HERA_HOME" mid-string. Containment-checked: a traversal payload
	// like "$HERA_HOME/../../etc/passwd" returns "" and the caller's
	// os.* operation will fail cleanly instead of resolving to an
	// attacker-chosen absolute path via filepath.Abs cleaning.
	if rest, ok := trimPrefix(p, "$HERA_HOME"); ok {
		if joined, ok := joinWithinHome(rest); ok {
			return joined
		}
		slog.Warn("paths.Normalize: $HERA_HOME traversal attempt refused", "input", p)
		return ""
	}
	if rest, ok := trimPrefix(p, "${HERA_HOME}"); ok {
		if joined, ok := joinWithinHome(rest); ok {
			return joined
		}
		slog.Warn("paths.Normalize: ${HERA_HOME} traversal attempt refused", "input", p)
		return ""
	}

	// .hera/... safety-net redirect. Applies only when the path starts
	// with ".hera" as its first segment -- we do NOT rewrite things like
	// "tests/.hera/fixture". Traversal is refused same as HERA_HOME.
	if p == ".hera" {
		return HeraHome()
	}
	if strings.HasPrefix(p, ".hera/") || strings.HasPrefix(p, ".hera"+string(filepath.Separator)) {
		rest := p[len(".hera")+1:]
		if joined, ok := joinWithinHome(rest); ok {
			slog.Info("paths.Normalize: redirected .hera/… to HERA_HOME",
				"input", p,
				"output", joined,
			)
			return joined
		}
		slog.Warn("paths.Normalize: .hera/… traversal attempt refused", "input", p)
		return ""
	}

	// Absolute stays as-is; relative anchored to CWD (CLI semantics).
	if filepath.IsAbs(p) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// joinWithinHome joins sub onto HeraHome() and returns the result only
// if the cleaned path stays within HeraHome() (equal to it, or a strict
// descendant). Returns ("", false) on a traversal attempt like
// "../../etc/passwd".
func joinWithinHome(sub string) (string, bool) {
	home := HeraHome()
	joined := filepath.Join(home, sub) // filepath.Join already Cleans.
	if joined == home {
		return joined, true
	}
	if strings.HasPrefix(joined, home+string(filepath.Separator)) {
		return joined, true
	}
	return "", false
}

// trimPrefix returns the remainder of s after prefix when prefix is
// followed by a path separator (or ends the string). Returns (rest,
// true) on match, ("", false) otherwise.
func trimPrefix(s, prefix string) (string, bool) {
	if s == prefix {
		return "", true
	}
	if !strings.HasPrefix(s, prefix) {
		return "", false
	}
	next := s[len(prefix)]
	if next == '/' || next == filepath.Separator {
		return s[len(prefix)+1:], true
	}
	return "", false
}
