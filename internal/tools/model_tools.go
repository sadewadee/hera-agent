package tools

// ModelInfo holds basic model metadata for tools that need it.
type ModelToolInfo struct {
	ModelID       string
	ContextWindow int
	SupportsTools bool
}

// ModelToolProvider is implemented by tools that need LLM model info.
type ModelToolProvider interface {
	Tool
	// SetModelInfo provides model metadata to the tool.
	SetModelInfo(info ModelToolInfo)
}
