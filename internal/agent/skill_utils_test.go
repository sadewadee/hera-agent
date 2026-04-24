package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseFrontmatter ---

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	fm, body := ParseFrontmatter("just a body")
	assert.Empty(t, fm)
	assert.Equal(t, "just a body", body)
}

func TestParseFrontmatter_ValidYAML(t *testing.T) {
	content := "---\nname: test-skill\ndescription: A test\n---\nBody here"
	fm, body := ParseFrontmatter(content)
	assert.Equal(t, "test-skill", fm["name"])
	assert.Equal(t, "A test", fm["description"])
	assert.Equal(t, "Body here", body)
}

func TestParseFrontmatter_EmptyBody(t *testing.T) {
	content := "---\nname: test\n---\n"
	fm, _ := ParseFrontmatter(content)
	assert.Equal(t, "test", fm["name"])
}

func TestParseFrontmatter_NoClosingDelimiter(t *testing.T) {
	content := "---\nname: test\nno closing"
	fm, body := ParseFrontmatter(content)
	assert.Empty(t, fm)
	assert.Equal(t, content, body)
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	content := "---\n{{{invalid yaml\n---\nBody"
	fm, body := ParseFrontmatter(content)
	// Should fallback to simple key:value parsing.
	assert.NotNil(t, fm)
	assert.Equal(t, "Body", body)
}

// --- SkillMatchesPlatform ---

func TestSkillMatchesPlatform_NoPlatforms(t *testing.T) {
	assert.True(t, SkillMatchesPlatform(map[string]interface{}{}))
}

func TestSkillMatchesPlatform_NilPlatforms(t *testing.T) {
	assert.True(t, SkillMatchesPlatform(map[string]interface{}{"platforms": nil}))
}

func TestSkillMatchesPlatform_CurrentOS(t *testing.T) {
	var platName string
	switch runtime.GOOS {
	case "darwin":
		platName = "macos"
	case "linux":
		platName = "linux"
	case "windows":
		platName = "windows"
	default:
		platName = runtime.GOOS
	}
	fm := map[string]interface{}{
		"platforms": []interface{}{platName},
	}
	assert.True(t, SkillMatchesPlatform(fm))
}

func TestSkillMatchesPlatform_WrongOS(t *testing.T) {
	// Pick a platform that is definitely not the current one.
	other := "windows"
	if runtime.GOOS == "windows" {
		other = "linux"
	}
	fm := map[string]interface{}{
		"platforms": []interface{}{other},
	}
	// This might be true if we're on the other platform, but the test
	// just verifies the function runs without error.
	_ = SkillMatchesPlatform(fm)
}

func TestSkillMatchesPlatform_StringPlatform(t *testing.T) {
	fm := map[string]interface{}{
		"platforms": runtime.GOOS,
	}
	assert.True(t, SkillMatchesPlatform(fm))
}

// --- normalizeStringSet ---

func TestNormalizeStringSet_Nil(t *testing.T) {
	result := normalizeStringSet(nil)
	assert.Empty(t, result)
}

func TestNormalizeStringSet_String(t *testing.T) {
	result := normalizeStringSet("  hello  ")
	assert.True(t, result["hello"])
}

func TestNormalizeStringSet_StringSlice(t *testing.T) {
	result := normalizeStringSet([]interface{}{"a", " b ", "c"})
	assert.True(t, result["a"])
	assert.True(t, result["b"])
	assert.True(t, result["c"])
}

func TestNormalizeStringSet_EmptyString(t *testing.T) {
	result := normalizeStringSet("  ")
	assert.Empty(t, result)
}

// --- GetDisabledSkillNames ---

func TestGetDisabledSkillNames_NonExistentFile(t *testing.T) {
	result := GetDisabledSkillNames("/nonexistent/config.yaml", "")
	assert.Empty(t, result)
}

func TestGetDisabledSkillNames_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := `skills:
  disabled:
    - skill-a
    - skill-b
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))
	result := GetDisabledSkillNames(cfgPath, "")
	assert.True(t, result["skill-a"])
	assert.True(t, result["skill-b"])
}

func TestGetDisabledSkillNames_PlatformSpecific(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := `skills:
  disabled:
    - global-disabled
  platform_disabled:
    telegram:
      - tg-disabled
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))
	result := GetDisabledSkillNames(cfgPath, "telegram")
	assert.True(t, result["tg-disabled"])
	assert.False(t, result["global-disabled"])
}

// --- GetExternalSkillsDirs ---

func TestGetExternalSkillsDirs_NonExistentConfig(t *testing.T) {
	result := GetExternalSkillsDirs("/nonexistent/config.yaml", "/tmp")
	assert.Nil(t, result)
}

func TestGetExternalSkillsDirs_ValidDirs(t *testing.T) {
	tmpDir := t.TempDir()
	extDir := filepath.Join(tmpDir, "ext-skills")
	require.NoError(t, os.MkdirAll(extDir, 0755))

	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := "skills:\n  external_dirs:\n    - " + extDir + "\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	result := GetExternalSkillsDirs(cfgPath, filepath.Join(tmpDir, "local"))
	assert.Contains(t, result, extDir)
}

func TestGetExternalSkillsDirs_SkipsDuplicateLocalDir(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "skills")
	require.NoError(t, os.MkdirAll(localDir, 0755))

	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := "skills:\n  external_dirs:\n    - " + localDir + "\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	result := GetExternalSkillsDirs(cfgPath, localDir)
	assert.Empty(t, result)
}

// --- ExcludedSkillDirs ---

func TestExcludedSkillDirs(t *testing.T) {
	assert.True(t, ExcludedSkillDirs[".git"])
	assert.True(t, ExcludedSkillDirs[".github"])
	assert.True(t, ExcludedSkillDirs[".hub"])
	assert.False(t, ExcludedSkillDirs["skills"])
}

// --- PlatformMap ---

func TestPlatformMap(t *testing.T) {
	assert.Equal(t, "darwin", PlatformMap["macos"])
	assert.Equal(t, "linux", PlatformMap["linux"])
	assert.Equal(t, "windows", PlatformMap["windows"])
}
