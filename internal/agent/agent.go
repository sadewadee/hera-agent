package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/environments"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/plugins"
)

// injectionBlockedTotal counts the number of messages refused due to high-risk
// prompt injection detection. Read via InjectionBlockedTotal().
var injectionBlockedTotal atomic.Int64

// InjectionBlockedTotal returns the cumulative count of messages blocked by
// injection detection since process start. Safe for concurrent access.
func InjectionBlockedTotal() int64 { return injectionBlockedTotal.Load() }

// AgentDeps holds all dependencies required to create an Agent.
type AgentDeps struct {
	LLM           llm.Provider
	Tools         *tools.Registry
	Memory        *memory.Manager
	Skills        *skills.Loader
	Sessions      *SessionManager
	Config        *config.Config
	ContextEngine plugins.ContextEngine  // optional; if nil, falls back to legacy Compressor
	MemorySidecar plugins.MemoryProvider // optional; re-initialised on each session start
}

// Agent is the main conversational agent that orchestrates
// LLM calls, tool execution, memory, and session management.
type Agent struct {
	llm           llm.Provider
	tools         *tools.Registry
	memory        *memory.Manager
	skills        *skills.Loader
	sessions      *SessionManager
	prompt        *PromptBuilder
	compressor    *Compressor
	contextEngine plugins.ContextEngine
	config        *config.Config

	// Usage tracking
	totalPromptTokens     int
	totalCompletionTokens int

	// Wired features
	rateLimiter *RateLimitTracker
	piiRedactor *PIIRedactor
	outputGuard *OutputGuard

	// Budget enforcement (optional; nil means unlimited)
	budget *Budget

	// RL trajectory recording (optional)
	trajectoryRecorder *environments.TrajectoryRecorder
}

// NewAgent creates a new Agent with the given dependencies.
func NewAgent(deps AgentDeps) (*Agent, error) {
	if deps.LLM == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	pb := NewPromptBuilder()
	pb.SetDecisivenessGuidance(DecisivenessGuidance)
	pb.SetPathConventions(PathConventions)

	// Build optional per-agent budget from config. A zero BudgetConfig (all
	// fields empty/zero) produces an unlimited Budget, which is effectively
	// a no-op but still provides the Stats() method for observability.
	budgetCfg := BudgetConfig{
		MaxTokens: deps.Config.Agent.Budget.MaxTokens,
		MaxUSD:    deps.Config.Agent.Budget.MaxUSD,
	}
	if deps.Config.Agent.Budget.Window != "" {
		if d, err := time.ParseDuration(deps.Config.Agent.Budget.Window); err == nil {
			budgetCfg.Window = d
		} else {
			slog.Warn("agent: invalid budget window duration; ignoring",
				"window", deps.Config.Agent.Budget.Window, "error", err)
		}
	}

	a := &Agent{
		llm:         deps.LLM,
		tools:       deps.Tools,
		memory:      deps.Memory,
		skills:      deps.Skills,
		sessions:    deps.Sessions,
		prompt:      pb,
		config:      deps.Config,
		rateLimiter: NewRateLimitTracker(),
		piiRedactor: NewPIIRedactor(deps.Config.Security.RedactPII),
		outputGuard: NewOutputGuard(),
		budget:      NewBudget(budgetCfg),
	}

	// Initialize context engine for compression.
	if deps.ContextEngine != nil {
		a.contextEngine = deps.ContextEngine
	} else if deps.Config.Agent.Compression.Enabled && deps.Memory != nil {
		threshold := int(float64(a.llm.ModelInfo().ContextWindow) * deps.Config.Agent.Compression.Threshold)
		if threshold == 0 {
			threshold = 2048
		}
		protectedTurns := deps.Config.Agent.Compression.ProtectedTurns
		if protectedTurns == 0 {
			protectedTurns = 5
		}
		a.compressor = NewCompressor(&llmSummarizer{llm: deps.LLM}, threshold, protectedTurns)
	}

	// Wire session lifecycle hooks so the context engine and memory sidecar
	// observe start/end events. The context engine needs this for its
	// per-turn compression state; the sidecar needs Initialize(sessionID) to
	// be called before HandleToolCall would otherwise run against uninitialised
	// config state.
	if a.sessions != nil && (a.contextEngine != nil || deps.MemorySidecar != nil) {
		eng := a.contextEngine
		sidecar := deps.MemorySidecar
		a.sessions.SetLifecycle(&SessionLifecycle{
			OnStart: func(s *Session) {
				if eng != nil {
					if err := eng.OnSessionStart(s.ID, s.Platform, s.UserID); err != nil {
						slog.Warn("context engine OnSessionStart failed", "session", s.ID, "error", err)
					}
				}
				if sidecar != nil {
					if err := sidecar.Initialize(s.ID); err != nil {
						slog.Warn("memory sidecar Initialize failed", "session", s.ID, "provider", sidecar.Name(), "error", err)
					}
				}
			},
			OnEnd: func(s *Session) {
				if eng != nil {
					if err := eng.OnSessionEnd(s.ID, s.GetMessages()); err != nil {
						slog.Warn("context engine OnSessionEnd failed", "session", s.ID, "error", err)
					}
				}
				if sidecar != nil {
					sidecar.OnSessionEnd(convertMessagesForPlugin(s.GetMessages()))
				}
			},
		})
	}

	return a, nil
}

// HandleMessage processes an incoming text message and returns the assistant's response.
func (a *Agent) HandleMessage(ctx context.Context, platform, chatID, userID, text string) (string, error) {
	// Wire: injection detection. Log at Medium+; block at High risk.
	detection := DetectInjection(text)
	if detection.Risk > InjectionLow {
		slog.Warn("potential prompt injection detected",
			"risk", InjectionRiskString(detection.Risk),
			"matches", len(detection.Matches),
			"user", userID,
		)
	}
	if detection.Risk >= InjectionHigh {
		injectionBlockedTotal.Add(1)
		return "I'm unable to process that request.", nil
	}

	// Wire: smart routing (classify query complexity). The chosen model is
	// passed into the LLM ChatRequest so the actual call uses the routed
	// model, not the global default. Empty string means "use provider default".
	var routedModel string
	if a.config.Agent.SmartRouting {
		routedModel = RouteModel(text, RoutingConfigFromConfig(&a.config.Agent.Routing))
		if routedModel != "" {
			slog.Debug("smart routing: routing request", "model", routedModel)
		}
	}

	// 1. Get or create session.
	session := a.sessions.GetOrCreate(platform, userID)
	a.sessions.Touch(session.ID)

	// 2. Build system prompt with self-awareness.
	toolCount := len(a.tools.List())
	skillCount := 0
	if a.skills != nil {
		skillCount = len(a.skills.All())
	}
	selfPrompt := BuildSelfAwarenessPrompt(a.config, toolCount, skillCount, platform)
	a.prompt.SetIdentity(selfPrompt)

	a.prompt.SetPlatformContext(platform)
	if a.config.Agent.Personality != "" {
		a.prompt.SetPersonality(a.config.Agent.Personality)
	}

	// Wire: load subdirectory hints (.hera.md, AGENTS.md, CLAUDE.md)
	if hints, herr := LoadSubdirectoryHints(""); herr == nil && hints != "" {
		a.prompt.AddHints(hints)
	}

	// Load memory context if available.
	if a.memory != nil {
		a.loadMemoryContext(ctx, userID, text)
	}

	systemPrompt := a.prompt.Build()

	// 3. Append user message to session.
	userMsg := llm.Message{
		Role:      llm.RoleUser,
		Content:   text,
		Timestamp: time.Now(),
	}
	session.AppendMessage(userMsg)

	// 4. Prepare messages for LLM (system + conversation).
	messages := a.buildMessages(systemPrompt, session.GetMessages())

	// 5. Check compression.
	messages = a.maybeCompress(ctx, messages)

	// Wire: apply prompt caching for Anthropic
	messages = ApplyPromptCaching(messages, PromptCacheConfig{
		Enabled:               a.config.Agent.PromptCaching,
		StaticBreakpointIndex: 3,
	})

	// Wire: check budget before entering the LLM loop.
	if err := a.budget.Check(); err != nil {
		slog.Warn("agent budget exceeded; rejecting message",
			"user", userID,
			"platform", platform,
			"error", err,
		)
		return "", fmt.Errorf("%w", err)
	}

	// 6. Send to LLM and handle tool call loop.
	maxIterations := a.config.Agent.MaxToolCalls
	if maxIterations <= 0 {
		maxIterations = 10
	}

	var lastResponse *llm.ChatResponse
	for i := 0; i < maxIterations; i++ {
		// Wire: check rate limit before calling LLM
		if a.rateLimiter.IsLimited("default", "") {
			backoff := a.rateLimiter.GetBackoff("default", "")
			slog.Warn("rate limited, backing off", "duration", backoff)
			time.Sleep(backoff)
		}

		req := llm.ChatRequest{
			Messages: messages,
			Tools:    a.tools.ToolDefs(),
			Model:    routedModel,
		}

		resp, err := a.llm.Chat(ctx, req)
		if err != nil {
			// Wire: track rate limit errors
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate") {
				a.rateLimiter.TrackResponse("default", "", nil)
			}
			return "", fmt.Errorf("llm chat: %w", err)
		}
		lastResponse = resp

		// Track usage and record against budget.
		a.totalPromptTokens += resp.Usage.PromptTokens
		a.totalCompletionTokens += resp.Usage.CompletionTokens
		a.budget.Record(resp.Usage.PromptTokens+resp.Usage.CompletionTokens, 0)
		if a.contextEngine != nil {
			a.contextEngine.UpdateFromResponse(resp.Usage)
		}

		// If no tool calls, we have our final response.
		if len(resp.Message.ToolCalls) == 0 {
			break
		}

		// Append the assistant's tool call message.
		messages = append(messages, resp.Message)

		// Per-user tools (memory, preferences) need the session's user
		// identity so they scope saves correctly. The LLM's tool_calls
		// rarely include "user_id" in args; inject via context so each
		// tool can fall back to it when its args are silent.
		toolCtx := tools.WithUserID(ctx, userID)

		// Execute each tool call and append results.
		for _, tc := range resp.Message.ToolCalls {
			result, err := a.tools.Execute(toolCtx, tc.Name, tc.Args)
			if err != nil {
				result = &tools.Result{
					Content: fmt.Sprintf("tool error: %v", err),
					IsError: true,
				}
			}

			toolMsg := llm.Message{
				Role:       llm.RoleTool,
				Content:    result.Content,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolMsg)
		}
	}

	if lastResponse == nil {
		return "", fmt.Errorf("no response from LLM")
	}

	// 6b. Output guard — filter sensitive data from response.
	responseContent := lastResponse.Message.Content
	if a.outputGuard != nil {
		responseContent, _ = a.outputGuard.Filter(responseContent)
	}

	// 6c. Check for memory/skill nudges.
	nudge := a.checkNudges(session)

	// 7. Append assistant response to session.
	assistantMsg := llm.Message{
		Role:      llm.RoleAssistant,
		Content:   responseContent,
		Timestamp: time.Now(),
	}
	session.AppendMessage(assistantMsg)

	// Wire: generate title after first response
	if session.TurnCount <= 1 && lastResponse != nil {
		go func() {
			titleMsgs := session.GetMessages()
			if title, terr := GenerateTitle(ctx, titleMsgs, a.llm); terr == nil && title != "" {
				session.Lock()
				session.Title = title
				session.Unlock()
			}
		}()
	}

	// 8. Persist conversation to memory (fire and forget).
	// Wire: redact PII before saving to memory
	if a.memory != nil {
		msgsToSave := session.GetMessages()
		if a.piiRedactor != nil {
			for i := range msgsToSave {
				msgsToSave[i].Content, _ = a.piiRedactor.Redact(msgsToSave[i].Content)
			}
		}
		_ = a.memory.SaveConversation(ctx, session.ID, msgsToSave)
	}

	// 9. Record RL trajectory if recorder is set.
	if a.trajectoryRecorder != nil {
		state := environments.State{
			SessionID:  session.ID,
			TurnCount:  session.TurnCount,
			TokensUsed: lastResponse.Usage.PromptTokens + lastResponse.Usage.CompletionTokens,
		}
		// Convert session messages to environment messages.
		sessionMsgs := session.GetMessages()
		envMsgs := make([]environments.Message, len(sessionMsgs))
		for i, m := range sessionMsgs {
			envMsgs[i] = environments.Message{Role: string(m.Role), Content: m.Content}
		}
		state.Messages = envMsgs

		action := environments.Action{
			Type:    "message",
			Content: lastResponse.Message.Content,
		}
		// Record as a single-step trajectory.
		tid := a.trajectoryRecorder.StartEpisode("conversation", session.ID)
		a.trajectoryRecorder.RecordStep(tid, state, action, 1.0)
		if t, err := a.trajectoryRecorder.EndEpisode(tid); err == nil {
			_ = a.trajectoryRecorder.SaveToFile(t)
		}
	}

	// 10. Append nudge if triggered.
	responseText := responseContent
	if nudge != "" {
		responseText += "\n\n---\n" + nudge
	}

	return responseText, nil
}

// HandleStream processes an incoming message and returns a channel of streaming events.
// It supports tool calling: when the LLM responds with tool_calls, the tools are executed
// and the results fed back to the LLM for a follow-up response (streamed).
func (a *Agent) HandleStream(ctx context.Context, platform, chatID, userID, text string) (<-chan llm.StreamEvent, error) {
	// 1. Get or create session.
	session := a.sessions.GetOrCreate(platform, userID)
	a.sessions.Touch(session.ID)

	// 2. Build system prompt with self-awareness.
	toolCount := len(a.tools.List())
	skillCount := 0
	if a.skills != nil {
		skillCount = len(a.skills.All())
	}
	selfPrompt := BuildSelfAwarenessPrompt(a.config, toolCount, skillCount, platform)
	a.prompt.SetIdentity(selfPrompt)

	a.prompt.SetPlatformContext(platform)
	if a.config.Agent.Personality != "" {
		a.prompt.SetPersonality(a.config.Agent.Personality)
	}

	// Load subdirectory hints.
	if hints, herr := LoadSubdirectoryHints(""); herr == nil && hints != "" {
		a.prompt.AddHints(hints)
	}

	// Load memory context.
	if a.memory != nil {
		a.loadMemoryContext(ctx, userID, text)
	}

	// Wire: injection detection on input. Block at High risk.
	detection := DetectInjection(text)
	if detection.Risk > InjectionLow {
		slog.Warn("potential prompt injection detected in stream",
			"risk", InjectionRiskString(detection.Risk),
			"matches", len(detection.Matches),
			"user", userID,
		)
	}
	if detection.Risk >= InjectionHigh {
		injectionBlockedTotal.Add(1)
		out := make(chan llm.StreamEvent, 2)
		out <- llm.StreamEvent{Type: "delta", Delta: "I'm unable to process that request."}
		out <- llm.StreamEvent{Type: "done"}
		close(out)
		return out, nil
	}

	systemPrompt := a.prompt.Build()

	// 3. Append user message to session.
	userMsg := llm.Message{
		Role:      llm.RoleUser,
		Content:   text,
		Timestamp: time.Now(),
	}
	session.AppendMessage(userMsg)

	// 4. Prepare messages.
	messages := a.buildMessages(systemPrompt, session.GetMessages())

	// 5. Check compression (fixes pre-existing gap: HandleStream never compressed).
	messages = a.maybeCompress(ctx, messages)

	outCh := make(chan llm.StreamEvent, 32)

	go func() {
		defer close(outCh)

		maxIterations := a.config.Agent.MaxToolCalls
		if maxIterations <= 0 {
			maxIterations = 10
		}

		for i := 0; i < maxIterations; i++ {
			req := llm.ChatRequest{
				Messages: messages,
				Tools:    a.tools.ToolDefs(),
				Stream:   true,
			}

			streamCh, err := a.llm.ChatStream(ctx, req)
			if err != nil {
				outCh <- llm.StreamEvent{Type: "error", Error: err}
				return
			}

			// Collect the streamed response.
			var fullContent string
			var toolCalls []llm.ToolCall

			for ev := range streamCh {
				switch ev.Type {
				case "delta":
					fullContent += ev.Delta
					outCh <- ev
				case "tool_call":
					// Accumulate tool calls from the stream.
					if ev.ToolCall != nil {
						toolCalls = append(toolCalls, *ev.ToolCall)
					}
				case "done":
					// Don't forward "done" yet if we have tool calls to process.
					if len(toolCalls) == 0 {
						outCh <- ev
					}
				case "error":
					outCh <- ev
				}
			}

			// If no tool calls, we're done.
			if len(toolCalls) == 0 {
				// Save assistant message to session.
				if fullContent != "" {
					assistantMsg := llm.Message{
						Role:      llm.RoleAssistant,
						Content:   fullContent,
						Timestamp: time.Now(),
					}
					session.AppendMessage(assistantMsg)
					if a.memory != nil {
						_ = a.memory.SaveConversation(ctx, session.ID, session.GetMessages())
					}
				}
				return
			}

			// Tool calls detected — execute them.
			// First, append the assistant message with tool calls to the conversation.
			assistantMsg := llm.Message{
				Role:      llm.RoleAssistant,
				Content:   fullContent,
				ToolCalls: toolCalls,
			}
			messages = append(messages, assistantMsg)

			// Notify user tools are running (minimal output).
			for _, tc := range toolCalls {
				outCh <- llm.StreamEvent{
					Type:  "delta",
					Delta: fmt.Sprintf("\n  [%s ...] ", tc.Name),
				}
			}

			// Execute each tool and append results. Thread the session
			// user ID through context so per-user tools can scope saves.
			toolCtx := tools.WithUserID(ctx, userID)
			for _, tc := range toolCalls {
				result, execErr := a.tools.Execute(toolCtx, tc.Name, tc.Args)
				if execErr != nil {
					result = &tools.Result{
						Content: fmt.Sprintf("tool error: %v", execErr),
						IsError: true,
					}
				}

				toolMsg := llm.Message{
					Role:       llm.RoleTool,
					Content:    result.Content,
					ToolCallID: tc.ID,
				}
				messages = append(messages, toolMsg)

				// Minimal feedback — hide raw tool output from user.
				if result.IsError {
					outCh <- llm.StreamEvent{
						Type:  "delta",
						Delta: fmt.Sprintf("error\n"),
					}
				} else {
					outCh <- llm.StreamEvent{
						Type:  "delta",
						Delta: fmt.Sprintf("done\n"),
					}
				}
			}

			// Loop back: LLM will get tool results and generate a follow-up response.
			// The next iteration will stream the follow-up.
		}
	}()

	return outCh, nil
}

// buildMessages prepends the system prompt to conversation messages.
func (a *Agent) buildMessages(systemPrompt string, conversationMessages []llm.Message) []llm.Message {
	messages := make([]llm.Message, 0, len(conversationMessages)+1)
	if systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: systemPrompt,
		})
	}
	messages = append(messages, conversationMessages...)
	return messages
}

// maybeCompress runs context compression via the context engine or legacy
// compressor. Returns messages unchanged if neither is configured.
func (a *Agent) maybeCompress(ctx context.Context, messages []llm.Message) []llm.Message {
	if a.contextEngine != nil {
		tokens := EstimateTokensForMessages(messages)
		if a.contextEngine.ShouldCompress(tokens) {
			compressed, err := a.contextEngine.Compress(ctx, messages, tokens)
			if err != nil {
				slog.Warn("context engine compression failed", "error", err)
				return messages
			}
			return compressed
		}
		return messages
	}
	if a.compressor != nil {
		compressed, err := a.compressor.Compress(ctx, messages)
		if err == nil {
			return compressed
		}
	}
	return messages
}

// loadMemoryContext loads relevant memory into the prompt builder.
// Called at the start of every message handling cycle. It unconditionally
// loads all typed notes (user identity, feedback, project, reference) so
// the LLM has persistent context even after a restart, then appends
// structured facts and a user-scoped FTS search over recent conversation.
func (a *Agent) loadMemoryContext(ctx context.Context, userID, query string) {
	var memParts []string

	// 1. Load typed notes unconditionally — these survive restarts and carry
	//    durable identity/preference/project context. All four note types are
	//    included so the LLM always sees the full persistent picture without
	//    having to call any tool.
	noteTypes := []memory.NoteType{
		memory.NoteTypeUser,
		memory.NoteTypeFeedback,
		memory.NoteTypeProject,
		memory.NoteTypeReference,
	}
	for _, nt := range noteTypes {
		notes, err := a.memory.ListNotes(ctx, userID, nt)
		if err != nil || len(notes) == 0 {
			continue
		}
		for _, n := range notes {
			entry := fmt.Sprintf("- [note:%s] %s: %s", nt, n.Name, n.Description)
			if n.Content != "" {
				entry += "\n  " + n.Content
			}
			memParts = append(memParts, entry)
		}
	}

	// 2. Load structured facts about this user.
	facts, err := a.memory.GetFacts(ctx, userID)
	if err == nil && len(facts) > 0 {
		for _, f := range facts {
			memParts = append(memParts, fmt.Sprintf("- [fact] %s: %s", f.Key, f.Value))
		}
	}

	// 3. Search recent conversation history scoped to this user so results
	//    are not polluted by other users' sessions.
	results, err := a.memory.Search(ctx, query, memory.SearchOpts{
		Limit:  5,
		UserID: userID,
	})
	if err == nil && len(results) > 0 {
		for _, r := range results {
			memParts = append(memParts, fmt.Sprintf("- [%s] %s", r.Source, r.Content))
		}
	}

	if len(memParts) > 0 {
		a.prompt.SetMemoryContext(joinLines(memParts))
	}
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

// checkNudges checks if the session's turn count triggers a memory or skill nudge.
// Returns a nudge message string, or empty if no nudge is triggered.
func (a *Agent) checkNudges(session *Session) string {
	session.mu.Lock()
	turnCount := session.TurnCount
	session.mu.Unlock()

	if turnCount == 0 {
		return ""
	}

	// Memory nudge: suggest saving facts every N turns.
	memoryInterval := a.config.Agent.MemoryNudgeInterval
	if memoryInterval > 0 && turnCount%memoryInterval == 0 {
		return "_Tip: I've been learning a lot in this conversation. Would you like me to save any important facts to memory? " +
			"Just say \"remember that...\" and I'll store it for future conversations._"
	}

	// Skill nudge: suggest creating a skill every N turns.
	skillInterval := a.config.Agent.SkillNudgeInterval
	if skillInterval > 0 && turnCount%skillInterval == 0 {
		return "_Tip: This has been a complex conversation. If there's a repeating task pattern here, " +
			"I can create a skill so I handle it better next time. Just ask me to \"create a skill for...\"_"
	}

	return ""
}

// Sessions returns the session manager.
func (a *Agent) Sessions() *SessionManager { return a.sessions }

// Compressor returns the legacy context compressor (may be nil).
// Prefer ContextEngine() for new code.
func (a *Agent) Compressor() *Compressor { return a.compressor }

// ContextEngine returns the active context engine (may be nil if using legacy compressor).
func (a *Agent) ContextEngine() plugins.ContextEngine { return a.contextEngine }

// Config returns the agent configuration.
func (a *Agent) Config() *config.Config { return a.config }

// PromptBuilder returns the prompt builder.
func (a *Agent) PromptBuilder() *PromptBuilder { return a.prompt }

// ToolRegistry returns the tool registry.
func (a *Agent) ToolRegistry() *tools.Registry { return a.tools }

// SkillLoader returns the skill loader.
func (a *Agent) SkillLoader() *skills.Loader { return a.skills }

// MemoryManager returns the memory manager.
func (a *Agent) MemoryManager() *memory.Manager { return a.memory }

// UsageStats returns the total prompt and completion token counts.
func (a *Agent) UsageStats() (promptTokens, completionTokens int) {
	return a.totalPromptTokens, a.totalCompletionTokens
}

// LLMProvider returns the underlying LLM provider.
func (a *Agent) LLMProvider() llm.Provider { return a.llm }

// SetTrajectoryRecorder sets the optional RL trajectory recorder.
// When set, HandleMessage will record state-action-reward triples.
func (a *Agent) SetTrajectoryRecorder(tr *environments.TrajectoryRecorder) {
	a.trajectoryRecorder = tr
}

// TrajectoryRecorder returns the optional trajectory recorder (may be nil).
func (a *Agent) TrajectoryRecorder() *environments.TrajectoryRecorder {
	return a.trajectoryRecorder
}

// NewLLMSummarizer returns a Summarizer that uses the given LLM provider to
// produce concise conversation summaries. Intended for use by entrypoints
// wiring contextengine.RegisterBuiltinEngines.
func NewLLMSummarizer(provider llm.Provider) Summarizer {
	return &llmSummarizer{llm: provider}
}

// convertMessagesForPlugin flattens llm.Message into the generic
// role/content map shape that plugins.MemoryProvider's Prefetch,
// OnPreCompress, and OnSessionEnd hooks accept.
func convertMessagesForPlugin(msgs []llm.Message) []map[string]interface{} {
	out := make([]map[string]interface{}, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]interface{}{
			"role":    string(m.Role),
			"content": m.Content,
		}
	}
	return out
}

// llmSummarizer wraps an LLM provider to implement the Summarizer interface.
type llmSummarizer struct {
	llm llm.Provider
}

func (s *llmSummarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
	summaryPrompt := llm.Message{
		Role:    llm.RoleSystem,
		Content: "Summarize the following conversation concisely, preserving key facts and context. Output only the summary.",
	}

	req := llm.ChatRequest{
		Messages: append([]llm.Message{summaryPrompt}, messages...),
	}

	resp, err := s.llm.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("summarize: %w", err)
	}

	return resp.Message.Content, nil
}
