package swe

import "testing"

func TestBuildToolset_ContainsExpectedTools(t *testing.T) {
	reg := BuildToolset(t.TempDir(), false)
	expected := []string{"file_read", "file_write", "run_command", "patch", "code_exec", "git"}
	for _, name := range expected {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("toolset missing %q", name)
		}
	}
}

func TestBuildToolset_PatchIsRealImpl(t *testing.T) {
	reg := BuildToolset(t.TempDir(), false)
	tool, ok := reg.Get("patch")
	if !ok {
		t.Fatal("patch tool not registered")
	}
	// Verify it's the SWE real impl, not the builtin stub, by type assertion.
	if _, isSWE := tool.(*swePatchTool); !isSWE {
		t.Errorf("patch tool is %T, want *swePatchTool", tool)
	}
}
