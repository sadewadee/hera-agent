package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecklist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, err := NewChecklist(path)
	require.NoError(t, err)
	assert.NotNil(t, cl)
	assert.Empty(t, cl.Items())
}

func TestChecklist_Add(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)

	item := cl.Add("Write tests")
	assert.Equal(t, 1, item.ID)
	assert.Equal(t, "Write tests", item.Text)
	assert.False(t, item.Done)
}

func TestChecklist_Add_Multiple(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)

	cl.Add("Task 1")
	cl.Add("Task 2")
	cl.Add("Task 3")

	items := cl.Items()
	assert.Len(t, items, 3)
	assert.Equal(t, 1, items[0].ID)
	assert.Equal(t, 2, items[1].ID)
	assert.Equal(t, 3, items[2].ID)
}

func TestChecklist_Remove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	cl.Add("Task 1")
	cl.Add("Task 2")

	ok := cl.Remove(1)
	assert.True(t, ok)
	assert.Len(t, cl.Items(), 1)
	assert.Equal(t, "Task 2", cl.Items()[0].Text)
}

func TestChecklist_Remove_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	ok := cl.Remove(99)
	assert.False(t, ok)
}

func TestChecklist_Toggle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	cl.Add("Task 1")

	ok := cl.Toggle(1)
	assert.True(t, ok)
	assert.True(t, cl.Items()[0].Done)

	ok = cl.Toggle(1)
	assert.True(t, ok)
	assert.False(t, cl.Items()[0].Done)
}

func TestChecklist_Toggle_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	ok := cl.Toggle(99)
	assert.False(t, ok)
}

func TestChecklist_List_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	result := cl.List()
	assert.Contains(t, result, "empty")
}

func TestChecklist_List_WithItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	cl.Add("Task A")
	cl.Add("Task B")
	cl.Toggle(1)

	result := cl.List()
	assert.Contains(t, result, "[x] Task A")
	assert.Contains(t, result, "[ ] Task B")
	assert.Contains(t, result, "1/2 completed")
}

func TestChecklist_Persistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	cl.Add("Persistent task")
	cl.Toggle(1)

	// Load a new checklist from the same file
	cl2, err := NewChecklist(path)
	require.NoError(t, err)

	items := cl2.Items()
	require.Len(t, items, 1)
	assert.Equal(t, "Persistent task", items[0].Text)
	assert.True(t, items[0].Done)
}

func TestChecklist_Items_ReturnsCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checklist.json")
	cl, _ := NewChecklist(path)
	cl.Add("Task 1")

	items := cl.Items()
	items[0].Text = "Modified"

	assert.Equal(t, "Task 1", cl.Items()[0].Text)
}

func TestChecklist_LoadsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checklist.json")
	os.WriteFile(path, []byte("invalid json"), 0644)

	cl, err := NewChecklist(path)
	require.NoError(t, err)
	assert.Empty(t, cl.Items())
}
