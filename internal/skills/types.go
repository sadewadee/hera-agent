package skills

// Skill represents a loaded skill with metadata and content.
type Skill struct {
	Name          string   `yaml:"name" json:"name"`
	Description   string   `yaml:"description" json:"description"`
	Triggers      []string `yaml:"triggers" json:"triggers"`
	Platforms     []string `yaml:"platforms" json:"platforms"`
	RequiresTools []string `yaml:"requires_tools" json:"requires_tools"`
	Content       string   `json:"content"`   // markdown body (after frontmatter)
	FilePath      string   `json:"file_path"` // path on disk
	Tier          string   `json:"tier"`      // "bundled", "optional", "user"
}
