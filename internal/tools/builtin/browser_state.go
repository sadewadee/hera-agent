// Package builtin provides built-in tool implementations.
//
// browser_state.go manages browser automation state tracking for the
// Camofox browser session, including page state, navigation history,
// and screenshot management.
package builtin

import (
	"log/slog"
	"sync"
	"time"
)

// BrowserState tracks the current state of a browser automation session.
type BrowserState struct {
	mu             sync.Mutex
	currentURL     string
	currentTitle   string
	isLoading      bool
	lastScreenshot string
	lastScreenTime time.Time
	history        []BrowserNavEntry
	maxHistory     int
	sessionActive  bool
	consoleErrors  []string
}

// BrowserNavEntry represents a single navigation entry.
type BrowserNavEntry struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
}

// NewBrowserState creates a new browser state tracker.
func NewBrowserState() *BrowserState {
	return &BrowserState{
		maxHistory: 50,
	}
}

// Update updates the browser state with new page info.
func (s *BrowserState) Update(url, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if url != s.currentURL {
		s.history = append(s.history, BrowserNavEntry{
			URL:       url,
			Title:     title,
			Timestamp: time.Now(),
		})
		if len(s.history) > s.maxHistory {
			s.history = s.history[len(s.history)-s.maxHistory:]
		}
	}

	s.currentURL = url
	s.currentTitle = title
	s.isLoading = false
}

// SetLoading marks the browser as loading.
func (s *BrowserState) SetLoading(loading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLoading = loading
}

// SetScreenshot records a screenshot path.
func (s *BrowserState) SetScreenshot(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastScreenshot = path
	s.lastScreenTime = time.Now()
}

// Current returns the current URL and title.
func (s *BrowserState) Current() (url, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentURL, s.currentTitle
}

// IsActive returns whether a browser session is active.
func (s *BrowserState) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionActive
}

// SetActive sets the session active state.
func (s *BrowserState) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionActive = active
	if !active {
		s.currentURL = ""
		s.currentTitle = ""
	}
	slog.Debug("browser session state changed", "active", active)
}

// History returns a copy of the navigation history.
func (s *BrowserState) History() []BrowserNavEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]BrowserNavEntry, len(s.history))
	copy(result, s.history)
	return result
}

// AddConsoleError records a browser console error.
func (s *BrowserState) AddConsoleError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consoleErrors = append(s.consoleErrors, msg)
	if len(s.consoleErrors) > 100 {
		s.consoleErrors = s.consoleErrors[len(s.consoleErrors)-100:]
	}
}

// ConsoleErrors returns recorded console errors.
func (s *BrowserState) ConsoleErrors() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.consoleErrors))
	copy(result, s.consoleErrors)
	return result
}

// Reset clears all browser state.
func (s *BrowserState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentURL = ""
	s.currentTitle = ""
	s.isLoading = false
	s.lastScreenshot = ""
	s.history = nil
	s.sessionActive = false
	s.consoleErrors = nil
}
