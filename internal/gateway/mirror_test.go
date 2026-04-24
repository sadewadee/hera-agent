package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirrorManager_NewMirrorManager(t *testing.T) {
	mm := NewMirrorManager(nil)
	require.NotNil(t, mm)
	assert.Empty(t, mm.Rules())
}

func TestMirrorManager_AddRule(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{
		Name:           "test-rule",
		SourcePlatform: "telegram",
		TargetPlatform: "discord",
		TargetChat:     "chan1",
		Enabled:        true,
	})

	rules := mm.Rules()
	assert.Len(t, rules, 1)
	assert.Equal(t, "test-rule", rules[0].Name)
}

func TestMirrorManager_RemoveRule(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{Name: "r1", Enabled: true})
	mm.AddRule(MirrorConfig{Name: "r2", Enabled: true})

	ok := mm.RemoveRule("r1")
	assert.True(t, ok)
	assert.Len(t, mm.Rules(), 1)
	assert.Equal(t, "r2", mm.Rules()[0].Name)
}

func TestMirrorManager_RemoveRule_NotFound(t *testing.T) {
	mm := NewMirrorManager(nil)
	ok := mm.RemoveRule("nonexistent")
	assert.False(t, ok)
}

func TestMirrorManager_Rules_ReturnsCopy(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{Name: "r1"})
	rules := mm.Rules()
	rules[0].Name = "modified"
	assert.Equal(t, "r1", mm.Rules()[0].Name)
}

func TestMirrorManager_ProcessMessage_NoGateway(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{
		Name:           "r1",
		SourcePlatform: "telegram",
		TargetPlatform: "discord",
		TargetChat:     "chan1",
		Enabled:        true,
	})

	// Should not panic with nil gateway
	err := mm.ProcessMessage(nil, "telegram", "chat1", "hello")
	assert.NoError(t, err)
}

func TestMirrorManager_ProcessMessage_DisabledRule(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{
		Name:           "disabled",
		SourcePlatform: "telegram",
		TargetPlatform: "discord",
		Enabled:        false,
	})

	err := mm.ProcessMessage(nil, "telegram", "chat1", "hello")
	assert.NoError(t, err)
}

func TestMirrorManager_ProcessMessage_WrongPlatform(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{
		Name:           "r1",
		SourcePlatform: "telegram",
		TargetPlatform: "discord",
		Enabled:        true,
	})

	err := mm.ProcessMessage(nil, "slack", "chat1", "hello")
	assert.NoError(t, err)
}

func TestMirrorManager_ProcessMessage_SpecificSourceChat(t *testing.T) {
	mm := NewMirrorManager(nil)
	mm.AddRule(MirrorConfig{
		Name:           "r1",
		SourcePlatform: "telegram",
		SourceChat:     "specific-chat",
		TargetPlatform: "discord",
		TargetChat:     "chan1",
		Enabled:        true,
	})

	// Different source chat should not match
	err := mm.ProcessMessage(nil, "telegram", "other-chat", "hello")
	assert.NoError(t, err)
}
