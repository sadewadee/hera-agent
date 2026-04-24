package cli

// CodexModel describes an OpenAI-compatible model.
type CodexModel struct {
	ID            string
	Name          string
	ContextWindow int
	PricePer1KIn  float64
	PricePer1KOut float64
}

// CodexModels is the catalog of known models.
var CodexModels = []CodexModel{
	{ID: "gpt-4o", Name: "GPT-4o", ContextWindow: 128000, PricePer1KIn: 0.005, PricePer1KOut: 0.015},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ContextWindow: 128000, PricePer1KIn: 0.00015, PricePer1KOut: 0.0006},
	{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", ContextWindow: 128000, PricePer1KIn: 0.01, PricePer1KOut: 0.03},
	{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", ContextWindow: 16385, PricePer1KIn: 0.0005, PricePer1KOut: 0.0015},
	{ID: "claude-3-opus", Name: "Claude 3 Opus", ContextWindow: 200000, PricePer1KIn: 0.015, PricePer1KOut: 0.075},
	{ID: "claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", ContextWindow: 200000, PricePer1KIn: 0.003, PricePer1KOut: 0.015},
	{ID: "claude-3-haiku", Name: "Claude 3 Haiku", ContextWindow: 200000, PricePer1KIn: 0.00025, PricePer1KOut: 0.00125},
	{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", ContextWindow: 1000000, PricePer1KIn: 0.00125, PricePer1KOut: 0.005},
	{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", ContextWindow: 1000000, PricePer1KIn: 0.000075, PricePer1KOut: 0.0003},
	{ID: "mistral-large", Name: "Mistral Large", ContextWindow: 128000, PricePer1KIn: 0.002, PricePer1KOut: 0.006},
	{ID: "mistral-small", Name: "Mistral Small", ContextWindow: 128000, PricePer1KIn: 0.0002, PricePer1KOut: 0.0006},
	{ID: "llama-3.1-405b", Name: "Llama 3.1 405B", ContextWindow: 131072, PricePer1KIn: 0.003, PricePer1KOut: 0.003},
	{ID: "llama-3.1-70b", Name: "Llama 3.1 70B", ContextWindow: 131072, PricePer1KIn: 0.00059, PricePer1KOut: 0.00079},
}

// FindModel looks up a model by ID.
func FindModel(id string) *CodexModel {
	for i, m := range CodexModels {
		if m.ID == id { return &CodexModels[i] }
	}
	return nil
}
