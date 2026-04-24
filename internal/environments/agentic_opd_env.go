package environments

// AgenticOPDEnv provides an open-problem-driven agentic environment.
type AgenticOPDEnv struct {
	BaseEnv
	ProblemDescription string
	SuccessCriteria    []string
	MaxAttempts        int
}

// NewAgenticOPDEnv creates an OPD environment.
func NewAgenticOPDEnv(problem string) *AgenticOPDEnv {
	base := NewBaseEnv()
	base.Name = "agentic-opd"
	return &AgenticOPDEnv{BaseEnv: *base, ProblemDescription: problem, MaxAttempts: 5}
}
