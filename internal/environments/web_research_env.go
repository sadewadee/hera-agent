package environments

// WebResearchEnv provides an environment for web research tasks.
type WebResearchEnv struct {
	BaseEnv
	MaxSearches int
	MaxScrapes  int
}

// NewWebResearchEnv creates a web research environment.
func NewWebResearchEnv() *WebResearchEnv {
	base := NewBaseEnv()
	base.Name = "web-research"
	base.Tools = []string{"web_search", "browser", "memory_save", "memory_search"}
	return &WebResearchEnv{BaseEnv: *base, MaxSearches: 20, MaxScrapes: 10}
}
