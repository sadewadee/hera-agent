// Package pathstest provides helpers for tests that touch the Hera
// filesystem. Use Isolate to give a test its own HERA_HOME + CWD so
// two tests don't step on each other's files.
package pathstest

import (
	"testing"
)

// Isolate creates a fresh tempdir, points HERA_HOME at it, and changes
// the test's working directory into it. Returns the tempdir path so
// the test can stage fixtures under it. Env and CWD are restored by
// t.Cleanup automatically (t.Setenv and t.Chdir install cleanups).
//
// Opt-in per test — Go's testing package has no autouse mechanism,
// so call this explicitly in every test that needs filesystem
// isolation from sibling tests.
func Isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)
	t.Chdir(dir)
	return dir
}
