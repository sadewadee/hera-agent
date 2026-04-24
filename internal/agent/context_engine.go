package agent

import "sync"

// ContextSource represents a source of context tokens (system, memory, tools, conversation).
type ContextSource struct {
	Name   string
	Tokens int
}

// ContextEngine manages the context window budget, tracking how many tokens each source uses.
type ContextEngine struct {
	mu          sync.Mutex
	maxTokens   int
	sources     map[string]int
	priority    []string
}

// NewContextEngine creates a context engine with the given max token budget.
func NewContextEngine(maxTokens int) *ContextEngine {
	return &ContextEngine{
		maxTokens: maxTokens,
		sources:   make(map[string]int),
		priority:  []string{"system", "tools", "memory", "conversation"},
	}
}

// Allocate reserves tokens for a named source.
func (ce *ContextEngine) Allocate(source string, tokens int) bool {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	used := ce.totalUsedLocked()
	if used+tokens > ce.maxTokens { return false }
	ce.sources[source] = tokens
	return true
}

// Available returns the number of tokens still available.
func (ce *ContextEngine) Available() int {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	return ce.maxTokens - ce.totalUsedLocked()
}

// Usage returns a snapshot of current token usage by source.
func (ce *ContextEngine) Usage() []ContextSource {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	result := make([]ContextSource, 0, len(ce.sources))
	for name, tokens := range ce.sources {
		result = append(result, ContextSource{Name: name, Tokens: tokens})
	}
	return result
}

func (ce *ContextEngine) totalUsedLocked() int {
	total := 0
	for _, t := range ce.sources { total += t }
	return total
}
