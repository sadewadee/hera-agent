package llm

// RegisterAll registers all 12 built-in LLM providers with the given registry.
// Call this instead of listing individual Register* calls in each binary.
func RegisterAll(r *Registry) {
	RegisterOpenAI(r)
	RegisterAnthropic(r)
	RegisterGemini(r)
	RegisterOllama(r)
	RegisterMistral(r)
	RegisterOpenRouter(r)
	RegisterCompatible(r)
	RegisterNous(r)
	RegisterHuggingFace(r)
	RegisterGLM(r)
	RegisterKimi(r)
	RegisterMiniMax(r)
}
