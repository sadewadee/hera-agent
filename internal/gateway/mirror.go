package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// MirrorConfig describes a single mirroring rule: messages from a source
// platform/chat are forwarded to a target platform/chat.
type MirrorConfig struct {
	Name           string `json:"name" yaml:"name"`
	SourcePlatform string `json:"source_platform" yaml:"source_platform"`
	SourceChat     string `json:"source_chat" yaml:"source_chat"`
	TargetPlatform string `json:"target_platform" yaml:"target_platform"`
	TargetChat     string `json:"target_chat" yaml:"target_chat"`
	Enabled        bool   `json:"enabled" yaml:"enabled"`
}

// MirrorManager watches for messages on source platforms and forwards them
// to configured target platforms.
type MirrorManager struct {
	mu      sync.RWMutex
	rules   []MirrorConfig
	gateway *Gateway
	logger  *slog.Logger
}

// NewMirrorManager creates a new MirrorManager attached to the given gateway.
func NewMirrorManager(gw *Gateway) *MirrorManager {
	return &MirrorManager{
		rules:   make([]MirrorConfig, 0),
		gateway: gw,
		logger:  slog.Default().With("component", "mirror"),
	}
}

// AddRule adds a mirroring rule.
func (mm *MirrorManager) AddRule(rule MirrorConfig) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.rules = append(mm.rules, rule)
}

// RemoveRule removes a mirroring rule by name.
func (mm *MirrorManager) RemoveRule(name string) bool {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for i, r := range mm.rules {
		if r.Name == name {
			mm.rules = append(mm.rules[:i], mm.rules[i+1:]...)
			return true
		}
	}
	return false
}

// Rules returns a copy of all mirroring rules.
func (mm *MirrorManager) Rules() []MirrorConfig {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	cp := make([]MirrorConfig, len(mm.rules))
	copy(cp, mm.rules)
	return cp
}

// ProcessMessage checks if a message from the given source platform/chat
// matches any mirroring rule and forwards it to the target(s).
func (mm *MirrorManager) ProcessMessage(ctx context.Context, sourcePlatform, sourceChat, text string) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for _, rule := range mm.rules {
		if !rule.Enabled {
			continue
		}
		if rule.SourcePlatform != sourcePlatform {
			continue
		}
		if rule.SourceChat != "" && rule.SourceChat != sourceChat {
			continue
		}

		// Forward to target.
		mm.logger.Info("mirroring message",
			"rule", rule.Name,
			"from", fmt.Sprintf("%s/%s", sourcePlatform, sourceChat),
			"to", fmt.Sprintf("%s/%s", rule.TargetPlatform, rule.TargetChat),
		)

		if mm.gateway != nil {
			adapter := mm.gateway.FindAdapter(rule.TargetPlatform)
			if adapter != nil {
				prefix := fmt.Sprintf("[mirrored from %s] ", sourcePlatform)
				outMsg := OutgoingMessage{Text: prefix + text}
				if err := adapter.Send(ctx, rule.TargetChat, outMsg); err != nil {
					mm.logger.Error("mirror send failed",
						"rule", rule.Name,
						"error", err,
					)
				}
			}
		}
	}

	return nil
}
