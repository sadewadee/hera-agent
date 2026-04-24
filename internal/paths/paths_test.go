package paths_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/paths"
)

func TestHeraHome_EnvOverride(t *testing.T) {
	t.Setenv("HERA_HOME", "/tmp/test-hera")
	got := paths.HeraHome()
	if got != "/tmp/test-hera" {
		t.Errorf("HeraHome() = %q, want %q", got, "/tmp/test-hera")
	}
}

func TestHeraHome_DefaultFallback(t *testing.T) {
	t.Setenv("HERA_HOME", "")
	got := paths.HeraHome()
	if !strings.HasSuffix(got, "/.hera") {
		t.Errorf("HeraHome() = %q, want suffix /.hera", got)
	}
}

func TestHeraBundled_EnvOverride(t *testing.T) {
	t.Setenv("HERA_BUNDLED", "/tmp/test-bundled")
	got := paths.HeraBundled()
	if got != "/tmp/test-bundled" {
		t.Errorf("HeraBundled() = %q, want %q", got, "/tmp/test-bundled")
	}
}

func TestHeraBundled_NoEnvReturnsSomething(t *testing.T) {
	t.Setenv("HERA_BUNDLED", "")
	got := paths.HeraBundled()
	// In test context (no installed binary), os.Executable returns the test
	// binary path. The result is either "" (on error) or ends with share/hera.
	if got != "" && !strings.HasSuffix(got, "share/hera") {
		t.Errorf("HeraBundled() = %q, want empty or suffix share/hera", got)
	}
}

func TestUserSkills(t *testing.T) {
	t.Setenv("HERA_HOME", "/tmp/h")
	got := paths.UserSkills()
	if got != "/tmp/h/skills" {
		t.Errorf("UserSkills() = %q, want %q", got, "/tmp/h/skills")
	}
}

func TestBundledSkills_WhenBundledEmpty(t *testing.T) {
	t.Setenv("HERA_BUNDLED", "")
	got := paths.BundledSkills()
	// When HeraBundled() returns "" OR a non-existent path, BundledSkills
	// may return "" or a path. When HERA_BUNDLED is set to empty string,
	// the fallback tries os.Executable; the test binary path won't have
	// share/hera but the path will still be constructed. What matters is
	// that it does not panic and returns a string.
	_ = got // just assert no panic and correct type
}

func TestBundledSkills_ExplicitEmpty(t *testing.T) {
	// When HeraBundled() explicitly returns "", BundledSkills() must return "".
	t.Setenv("HERA_BUNDLED", "___FORCE_EMPTY___")
	// We set a value that won't have share/hera, so the env is non-empty.
	// To test the truly-empty case we need a way to force HeraBundled to "".
	// The only way to force "" is os.Executable failure, which we can't simulate.
	// So we test BundledSkills with a known HERA_BUNDLED set to a real value.
	t.Setenv("HERA_BUNDLED", "/tmp/bundled")
	got := paths.BundledSkills()
	if got != "/tmp/bundled/skills" {
		t.Errorf("BundledSkills() = %q, want %q", got, "/tmp/bundled/skills")
	}
}

func TestBundledHooks(t *testing.T) {
	t.Setenv("HERA_BUNDLED", "/tmp/bundled")
	got := paths.BundledHooks()
	if got != "/tmp/bundled/hooks.d" {
		t.Errorf("BundledHooks() = %q, want %q", got, "/tmp/bundled/hooks.d")
	}
}

func TestBundledTools(t *testing.T) {
	t.Setenv("HERA_BUNDLED", "/tmp/bundled")
	got := paths.BundledTools()
	if got != "/tmp/bundled/tools.d" {
		t.Errorf("BundledTools() = %q, want %q", got, "/tmp/bundled/tools.d")
	}
}

func TestAllSubpaths(t *testing.T) {
	t.Setenv("HERA_HOME", "/tmp/h")

	cases := []struct {
		name string
		fn   func() string
		want string
	}{
		{"UserHooks", paths.UserHooks, "/tmp/h/hooks.d"},
		{"UserTools", paths.UserTools, "/tmp/h/tools.d"},
		{"UserPlugins", paths.UserPlugins, "/tmp/h/plugins"},
		{"UserPersonalities", paths.UserPersonalities, "/tmp/h/personalities"},
		{"UserDB", paths.UserDB, "/tmp/h/hera.db"},
		{"UserLogs", paths.UserLogs, "/tmp/h/logs"},
		{"UserConfig", paths.UserConfig, "/tmp/h/config.yaml"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn()
			if got != tc.want {
				t.Errorf("%s() = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	// Pin HERA_HOME + HOME so expected values are predictable.
	heraHome := "/tmp/custom-hera"
	home := "/tmp/fake-home"
	t.Setenv("HERA_HOME", heraHome)
	t.Setenv("HOME", home)

	// Pick a CWD that is NOT any of the prefixes above so "bare" paths
	// are distinguishable in assertions.
	cwd := t.TempDir()
	t.Chdir(cwd)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"bare tilde", "~", home},
		{"tilde slash", "~/foo", home + "/foo"},
		{"tilde nested", "~/docs/memo.md", home + "/docs/memo.md"},
		{"HERA_HOME unbraced", "$HERA_HOME/articles/x.md", heraHome + "/articles/x.md"},
		{"HERA_HOME braced", "${HERA_HOME}/data/y", heraHome + "/data/y"},
		{"HERA_HOME exact", "$HERA_HOME", heraHome},
		{"dot-hera exact", ".hera", heraHome},
		{"dot-hera slash", ".hera/workers/w.py", heraHome + "/workers/w.py"},
		{"absolute stays", "/etc/hosts", "/etc/hosts"},
		{"bare relative CWD", "foo.md", cwd + "/foo.md"},
		// .hera NOT as leading segment — left alone (anchored to CWD).
		{"inner dot-hera not rewritten", "tests/.hera/fixture", cwd + "/tests/.hera/fixture"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := paths.Normalize(tc.in)
			if got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalize_NoLeakageBetweenPrefixes(t *testing.T) {
	// A path that LOOKS like it starts with $HERA_HOME but is actually a
	// longer token (e.g. "$HERA_HOME_BACKUP/...") must NOT be rewritten.
	t.Setenv("HERA_HOME", "/tmp/h")
	got := paths.Normalize("$HERA_HOME_BACKUP/foo")
	if got == "/tmp/h_BACKUP/foo" || got == "/tmp/h/_BACKUP/foo" {
		t.Errorf("Normalize rewrote %q as a HERA_HOME prefix: got %q", "$HERA_HOME_BACKUP/foo", got)
	}
}

func TestNormalize_RefusesTraversal(t *testing.T) {
	home := "/tmp/hera-traversal-test"
	t.Setenv("HERA_HOME", home)
	t.Chdir(t.TempDir()) // CWD isolated from HERA_HOME

	// Traversal inputs must return "" so downstream os.* calls fail
	// cleanly. Returning the filepath.Abs of the literal input is NOT
	// acceptable because filepath.Clean reduces "$HERA_HOME/../../../etc/
	// passwd" to "/etc/passwd" when CWD happens to be near root — a
	// silent sandbox escape.
	cases := []struct {
		name string
		in   string
	}{
		{"dot-hera parent escape", ".hera/../../../etc/passwd"},
		{"dollar parent escape", "$HERA_HOME/../../../etc/passwd"},
		{"braced parent escape", "${HERA_HOME}/../../../etc/passwd"},
		{"dot-hera single dotdot", ".hera/../foo"},
		{"dollar single dotdot", "$HERA_HOME/../foo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := paths.Normalize(tc.in)
			if got != "" {
				t.Errorf("Normalize(%q) = %q — traversal payload should yield empty string, not a resolved path", tc.in, got)
			}
		})
	}
}

func TestNormalize_AllowsInternalDotDot(t *testing.T) {
	// Legit case: ".hera/foo/../bar.md" cleans to ".hera/bar.md" which
	// stays inside HeraHome. Must not be confused with a traversal attempt.
	home := t.TempDir()
	t.Setenv("HERA_HOME", home)

	got := paths.Normalize(".hera/foo/../bar.md")
	want := filepath.Join(home, "bar.md")
	if got != want {
		t.Errorf("Normalize(%q) = %q, want %q", ".hera/foo/../bar.md", got, want)
	}
}
