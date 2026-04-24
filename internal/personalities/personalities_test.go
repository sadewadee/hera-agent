package personalities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_BundledKawaiiHasExpandedContent(t *testing.T) {
	got := Resolve("kawaii")
	// Expanded profile contains signature Japanese greetings.
	assert.Contains(t, got, "Ohayou gozaimasu")
	assert.Contains(t, got, "Konbanwa")
	assert.Contains(t, got, "Master")
	assert.NotEqual(t, "kawaii", got, "name should resolve to guidelines content")
}

func TestResolve_BundledHelpfulAndOthers(t *testing.T) {
	for _, name := range []string{"helpful", "concise", "technical"} {
		t.Run(name, func(t *testing.T) {
			got := Resolve(name)
			assert.NotEqual(t, name, got, "expected bundled profile for %q to resolve", name)
			assert.NotEmpty(t, got)
		})
	}
}

func TestResolve_UnknownNamePassesThrough(t *testing.T) {
	got := Resolve("not_a_personality_xyz")
	assert.Equal(t, "not_a_personality_xyz", got)
}

func TestResolve_InlineGuidelinesPassThrough(t *testing.T) {
	inline := "You are a helpful assistant.\nBe brief and accurate."
	got := Resolve(inline)
	assert.Equal(t, inline, got, "multi-line inline values must not be treated as names")
}

func TestResolve_EmptyStayEmpty(t *testing.T) {
	assert.Equal(t, "", Resolve(""))
}

func TestResolve_UserOverrideWins(t *testing.T) {
	// Point HOME at a temp dir and drop a kawaii override there.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	overrideDir := filepath.Join(tmp, ".hera", "personalities")
	require.NoError(t, os.MkdirAll(overrideDir, 0o755))
	override := "name: kawaii\nguidelines: |\n  USER OVERRIDE ACTIVE\n"
	require.NoError(t, os.WriteFile(filepath.Join(overrideDir, "kawaii.yaml"), []byte(override), 0o644))

	got := Resolve("kawaii")
	assert.Equal(t, "USER OVERRIDE ACTIVE", strings.TrimSpace(got))
}

func TestList_ReturnsBundledNames(t *testing.T) {
	names := List()
	assert.Contains(t, names, "kawaii")
	assert.Contains(t, names, "helpful")
	assert.GreaterOrEqual(t, len(names), 10)
}

// TestKawaiiYAML_HasDecisivenessCaveat asserts that the kawaii personality YAML
// contains the action-bias caveat comment next to the clarification phrasing.
// This is a dev-discipline test — comments are not parsed into guidelines, but
// they document the correct usage to future maintainers.
func TestKawaiiYAML_HasDecisivenessCaveat(t *testing.T) {
	data, err := bundled.ReadFile("kawaii.yaml")
	require.NoError(t, err)
	raw := string(data)

	// The YAML must contain the clarification phrasing.
	require.Contains(t, raw, "Butuh klarifikasi", "kawaii.yaml must still contain clarification phrasing")

	// Adjacent to the clarification phrasing, a caveat comment must be present.
	assert.Contains(t, raw, "# Caveat:", "kawaii.yaml must have a Caveat comment near clarification phrasing")
	assert.Contains(t, raw, "Default: act on obvious interpretation", "kawaii.yaml Caveat must instruct action-bias default")
}
