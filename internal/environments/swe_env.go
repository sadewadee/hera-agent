package environments

// SWEEnv provides a software engineering environment with code tools.
type SWEEnv struct {
	BaseEnv
	RepoPath    string
	Language    string
	TestCommand string
}

// NewSWEEnv creates a SWE environment.
func NewSWEEnv(repoPath string) *SWEEnv {
	base := NewBaseEnv()
	base.Name = "swe"
	base.Tools = append(base.Tools, "code_exec", "patch", "terminal")
	return &SWEEnv{BaseEnv: *base, RepoPath: repoPath}
}
