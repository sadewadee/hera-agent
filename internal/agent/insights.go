package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"
)

// InsightsEngine analyzes session history and produces usage insights.
// Works directly with a *sql.DB connection to query session and message data.
type InsightsEngine struct {
	db *sql.DB
}

// NewInsightsEngine creates a new InsightsEngine with the given database connection.
func NewInsightsEngine(db *sql.DB) *InsightsEngine {
	return &InsightsEngine{db: db}
}

// InsightsReport holds all computed insights for a time period.
type InsightsReport struct {
	Days         int                 `json:"days"`
	SourceFilter string              `json:"source_filter,omitempty"`
	Empty        bool                `json:"empty"`
	GeneratedAt  float64             `json:"generated_at,omitempty"`
	Overview     InsightsOverview    `json:"overview"`
	Models       []ModelBreakdown    `json:"models"`
	Platforms    []PlatformBreakdown `json:"platforms"`
	Tools        []ToolBreakdown     `json:"tools"`
	Activity     ActivityPatterns    `json:"activity"`
	TopSessions  []NotableSession    `json:"top_sessions"`
}

// InsightsOverview holds high-level overview statistics.
type InsightsOverview struct {
	TotalSessions       int     `json:"total_sessions"`
	TotalMessages       int     `json:"total_messages"`
	TotalToolCalls      int     `json:"total_tool_calls"`
	TotalInputTokens    int64   `json:"total_input_tokens"`
	TotalOutputTokens   int64   `json:"total_output_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	EstimatedCost       float64 `json:"estimated_cost"`
	TotalHours          float64 `json:"total_hours"`
	AvgSessionDuration  float64 `json:"avg_session_duration"`
	AvgMsgsPerSession   float64 `json:"avg_msgs_per_session"`
	AvgTokensPerSession float64 `json:"avg_tokens_per_session"`
	UserMessages        int     `json:"user_messages"`
	AssistantMessages   int     `json:"assistant_messages"`
	ToolMessages        int     `json:"tool_messages"`
	DateRangeStart      float64 `json:"date_range_start,omitempty"`
	DateRangeEnd        float64 `json:"date_range_end,omitempty"`
}

// ModelBreakdown holds usage statistics for a single model.
type ModelBreakdown struct {
	Model        string  `json:"model"`
	Sessions     int     `json:"sessions"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	ToolCalls    int     `json:"tool_calls"`
	Cost         float64 `json:"cost"`
}

// PlatformBreakdown holds usage statistics for a single platform.
type PlatformBreakdown struct {
	Platform     string `json:"platform"`
	Sessions     int    `json:"sessions"`
	Messages     int    `json:"messages"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
	ToolCalls    int    `json:"tool_calls"`
}

// ToolBreakdown holds usage statistics for a single tool.
type ToolBreakdown struct {
	Tool       string  `json:"tool"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ActivityPatterns holds activity pattern analysis.
type ActivityPatterns struct {
	ByDay       []DayCount  `json:"by_day"`
	ByHour      []HourCount `json:"by_hour"`
	BusiestDay  *DayCount   `json:"busiest_day,omitempty"`
	BusiestHour *HourCount  `json:"busiest_hour,omitempty"`
	ActiveDays  int         `json:"active_days"`
	MaxStreak   int         `json:"max_streak"`
}

// DayCount holds session count for a day of the week.
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// HourCount holds session count for an hour of the day.
type HourCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// NotableSession describes a remarkable session.
type NotableSession struct {
	Label     string `json:"label"`
	SessionID string `json:"session_id"`
	Value     string `json:"value"`
	Date      string `json:"date"`
}

type sessionRow struct {
	ID            string
	Source        string
	Model         string
	StartedAt     float64
	EndedAt       float64
	MessageCount  int
	ToolCallCount int
	InputTokens   int64
	OutputTokens  int64
}

type toolUsageRow struct {
	ToolName string
	Count    int
}

type messageStats struct {
	TotalMessages     int
	UserMessages      int
	AssistantMessages int
	ToolMessages      int
}

// Generate produces a complete insights report.
func (e *InsightsEngine) Generate(days int, source string) (*InsightsReport, error) {
	cutoff := float64(time.Now().Unix()) - float64(days*86400)

	sessions, err := e.getSessions(cutoff, source)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}
	toolUsage, err := e.getToolUsage(cutoff, source)
	if err != nil {
		return nil, fmt.Errorf("get tool usage: %w", err)
	}
	msgStats, err := e.getMessageStats(cutoff, source)
	if err != nil {
		return nil, fmt.Errorf("get message stats: %w", err)
	}

	if len(sessions) == 0 {
		return &InsightsReport{
			Days:         days,
			SourceFilter: source,
			Empty:        true,
		}, nil
	}

	overview := e.computeOverview(sessions, msgStats)
	models := e.computeModelBreakdown(sessions)
	platforms := e.computePlatformBreakdown(sessions)
	tools := e.computeToolBreakdown(toolUsage)
	activity := e.computeActivityPatterns(sessions)
	topSessions := e.computeTopSessions(sessions)

	return &InsightsReport{
		Days:         days,
		SourceFilter: source,
		Empty:        false,
		GeneratedAt:  float64(time.Now().Unix()),
		Overview:     overview,
		Models:       models,
		Platforms:    platforms,
		Tools:        tools,
		Activity:     activity,
		TopSessions:  topSessions,
	}, nil
}

func (e *InsightsEngine) getSessions(cutoff float64, source string) ([]sessionRow, error) {
	var query string
	var args []interface{}

	cols := "id, source, model, started_at, ended_at, message_count, tool_call_count, input_tokens, output_tokens"
	if source != "" {
		query = fmt.Sprintf("SELECT %s FROM sessions WHERE started_at >= ? AND source = ? ORDER BY started_at DESC", cols)
		args = []interface{}{cutoff, source}
	} else {
		query = fmt.Sprintf("SELECT %s FROM sessions WHERE started_at >= ? ORDER BY started_at DESC", cols)
		args = []interface{}{cutoff}
	}

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []sessionRow
	for rows.Next() {
		var s sessionRow
		var endedAt sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.Source, &s.Model, &s.StartedAt, &endedAt,
			&s.MessageCount, &s.ToolCallCount, &s.InputTokens, &s.OutputTokens); err != nil {
			slog.Debug("scan session row", "error", err)
			continue
		}
		if endedAt.Valid {
			s.EndedAt = endedAt.Float64
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (e *InsightsEngine) getToolUsage(cutoff float64, source string) ([]toolUsageRow, error) {
	var query string
	var args []interface{}

	if source != "" {
		query = `SELECT m.tool_name, COUNT(*) as count
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE s.started_at >= ? AND s.source = ?
				AND m.role = 'tool' AND m.tool_name IS NOT NULL
			GROUP BY m.tool_name
			ORDER BY count DESC`
		args = []interface{}{cutoff, source}
	} else {
		query = `SELECT m.tool_name, COUNT(*) as count
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE s.started_at >= ?
				AND m.role = 'tool' AND m.tool_name IS NOT NULL
			GROUP BY m.tool_name
			ORDER BY count DESC`
		args = []interface{}{cutoff}
	}

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tool usage: %w", err)
	}
	defer rows.Close()

	toolCounts := make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		toolCounts[name] += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Also extract from tool_calls JSON on assistant messages
	var query2 string
	if source != "" {
		query2 = `SELECT m.tool_calls
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE s.started_at >= ? AND s.source = ?
				AND m.role = 'assistant' AND m.tool_calls IS NOT NULL`
	} else {
		query2 = `SELECT m.tool_calls
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE s.started_at >= ?
				AND m.role = 'assistant' AND m.tool_calls IS NOT NULL`
	}

	rows2, err := e.db.Query(query2, args...)
	if err != nil {
		slog.Debug("query tool_calls JSON", "error", err)
	} else {
		defer rows2.Close()
		tcCounts := make(map[string]int)
		for rows2.Next() {
			var raw string
			if err := rows2.Scan(&raw); err != nil {
				continue
			}
			var calls []map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &calls); err != nil {
				continue
			}
			for _, call := range calls {
				fn, _ := call["function"].(map[string]interface{})
				if fn == nil {
					continue
				}
				name, _ := fn["name"].(string)
				if name != "" {
					tcCounts[name]++
				}
			}
		}
		// Merge: take max of each tool's count
		if len(toolCounts) == 0 {
			toolCounts = tcCounts
		} else {
			for tool, count := range tcCounts {
				if count > toolCounts[tool] {
					toolCounts[tool] = count
				}
			}
		}
	}

	var result []toolUsageRow
	for name, count := range toolCounts {
		result = append(result, toolUsageRow{ToolName: name, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	return result, nil
}

func (e *InsightsEngine) getMessageStats(cutoff float64, source string) (messageStats, error) {
	var query string
	var args []interface{}

	if source != "" {
		query = `SELECT
			COUNT(*) as total_messages,
			SUM(CASE WHEN m.role = 'user' THEN 1 ELSE 0 END) as user_messages,
			SUM(CASE WHEN m.role = 'assistant' THEN 1 ELSE 0 END) as assistant_messages,
			SUM(CASE WHEN m.role = 'tool' THEN 1 ELSE 0 END) as tool_messages
		FROM messages m
		JOIN sessions s ON s.id = m.session_id
		WHERE s.started_at >= ? AND s.source = ?`
		args = []interface{}{cutoff, source}
	} else {
		query = `SELECT
			COUNT(*) as total_messages,
			SUM(CASE WHEN m.role = 'user' THEN 1 ELSE 0 END) as user_messages,
			SUM(CASE WHEN m.role = 'assistant' THEN 1 ELSE 0 END) as assistant_messages,
			SUM(CASE WHEN m.role = 'tool' THEN 1 ELSE 0 END) as tool_messages
		FROM messages m
		JOIN sessions s ON s.id = m.session_id
		WHERE s.started_at >= ?`
		args = []interface{}{cutoff}
	}

	var ms messageStats
	err := e.db.QueryRow(query, args...).Scan(
		&ms.TotalMessages, &ms.UserMessages, &ms.AssistantMessages, &ms.ToolMessages)
	if err != nil {
		return messageStats{}, fmt.Errorf("query message stats: %w", err)
	}
	return ms, nil
}

func (e *InsightsEngine) computeOverview(sessions []sessionRow, ms messageStats) InsightsOverview {
	var totalInput, totalOutput int64
	var totalToolCalls, totalMessages int

	for _, s := range sessions {
		totalInput += s.InputTokens
		totalOutput += s.OutputTokens
		totalToolCalls += s.ToolCallCount
		totalMessages += s.MessageCount
	}
	totalTokens := totalInput + totalOutput

	var durations []float64
	for _, s := range sessions {
		if s.StartedAt > 0 && s.EndedAt > s.StartedAt {
			durations = append(durations, s.EndedAt-s.StartedAt)
		}
	}

	var totalHours, avgDuration float64
	if len(durations) > 0 {
		var sum float64
		for _, d := range durations {
			sum += d
		}
		totalHours = sum / 3600
		avgDuration = sum / float64(len(durations))
	}

	var dateStart, dateEnd float64
	for _, s := range sessions {
		if s.StartedAt > 0 {
			if dateStart == 0 || s.StartedAt < dateStart {
				dateStart = s.StartedAt
			}
			if s.StartedAt > dateEnd {
				dateEnd = s.StartedAt
			}
		}
	}

	n := len(sessions)
	var avgMsgs, avgTokens float64
	if n > 0 {
		avgMsgs = float64(totalMessages) / float64(n)
		avgTokens = float64(totalTokens) / float64(n)
	}

	return InsightsOverview{
		TotalSessions:       n,
		TotalMessages:       totalMessages,
		TotalToolCalls:      totalToolCalls,
		TotalInputTokens:    totalInput,
		TotalOutputTokens:   totalOutput,
		TotalTokens:         totalTokens,
		TotalHours:          totalHours,
		AvgSessionDuration:  avgDuration,
		AvgMsgsPerSession:   avgMsgs,
		AvgTokensPerSession: avgTokens,
		UserMessages:        ms.UserMessages,
		AssistantMessages:   ms.AssistantMessages,
		ToolMessages:        ms.ToolMessages,
		DateRangeStart:      dateStart,
		DateRangeEnd:        dateEnd,
	}
}

func (e *InsightsEngine) computeModelBreakdown(sessions []sessionRow) []ModelBreakdown {
	type modelData struct {
		Sessions     int
		InputTokens  int64
		OutputTokens int64
		TotalTokens  int64
		ToolCalls    int
	}
	data := make(map[string]*modelData)

	for _, s := range sessions {
		model := s.Model
		if model == "" {
			model = "unknown"
		}
		if idx := strings.LastIndex(model, "/"); idx >= 0 {
			model = model[idx+1:]
		}
		d, ok := data[model]
		if !ok {
			d = &modelData{}
			data[model] = d
		}
		d.Sessions++
		d.InputTokens += s.InputTokens
		d.OutputTokens += s.OutputTokens
		d.TotalTokens += s.InputTokens + s.OutputTokens
		d.ToolCalls += s.ToolCallCount
	}

	var result []ModelBreakdown
	for model, d := range data {
		result = append(result, ModelBreakdown{
			Model:        model,
			Sessions:     d.Sessions,
			InputTokens:  d.InputTokens,
			OutputTokens: d.OutputTokens,
			TotalTokens:  d.TotalTokens,
			ToolCalls:    d.ToolCalls,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalTokens != result[j].TotalTokens {
			return result[i].TotalTokens > result[j].TotalTokens
		}
		return result[i].Sessions > result[j].Sessions
	})
	return result
}

func (e *InsightsEngine) computePlatformBreakdown(sessions []sessionRow) []PlatformBreakdown {
	type platData struct {
		Sessions     int
		Messages     int
		InputTokens  int64
		OutputTokens int64
		TotalTokens  int64
		ToolCalls    int
	}
	data := make(map[string]*platData)

	for _, s := range sessions {
		source := s.Source
		if source == "" {
			source = "unknown"
		}
		d, ok := data[source]
		if !ok {
			d = &platData{}
			data[source] = d
		}
		d.Sessions++
		d.Messages += s.MessageCount
		d.InputTokens += s.InputTokens
		d.OutputTokens += s.OutputTokens
		d.TotalTokens += s.InputTokens + s.OutputTokens
		d.ToolCalls += s.ToolCallCount
	}

	var result []PlatformBreakdown
	for plat, d := range data {
		result = append(result, PlatformBreakdown{
			Platform:     plat,
			Sessions:     d.Sessions,
			Messages:     d.Messages,
			InputTokens:  d.InputTokens,
			OutputTokens: d.OutputTokens,
			TotalTokens:  d.TotalTokens,
			ToolCalls:    d.ToolCalls,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sessions > result[j].Sessions })
	return result
}

func (e *InsightsEngine) computeToolBreakdown(usage []toolUsageRow) []ToolBreakdown {
	var totalCalls int
	for _, t := range usage {
		totalCalls += t.Count
	}

	var result []ToolBreakdown
	for _, t := range usage {
		pct := 0.0
		if totalCalls > 0 {
			pct = float64(t.Count) / float64(totalCalls) * 100
		}
		result = append(result, ToolBreakdown{
			Tool:       t.ToolName,
			Count:      t.Count,
			Percentage: pct,
		})
	}
	return result
}

func (e *InsightsEngine) computeActivityPatterns(sessions []sessionRow) ActivityPatterns {
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	dayCounts := make([]int, 7)
	hourCounts := make([]int, 24)
	dailyCounts := make(map[string]int)

	for _, s := range sessions {
		if s.StartedAt <= 0 {
			continue
		}
		t := time.Unix(int64(s.StartedAt), 0)
		wd := int(t.Weekday())
		// Convert from Sunday=0 to Monday=0
		wd = (wd + 6) % 7
		dayCounts[wd]++
		hourCounts[t.Hour()]++
		dailyCounts[t.Format("2006-01-02")]++
	}

	byDay := make([]DayCount, 7)
	for i := 0; i < 7; i++ {
		byDay[i] = DayCount{Day: dayNames[i], Count: dayCounts[i]}
	}
	byHour := make([]HourCount, 24)
	for i := 0; i < 24; i++ {
		byHour[i] = HourCount{Hour: i, Count: hourCounts[i]}
	}

	var busiestDay *DayCount
	for i := range byDay {
		if busiestDay == nil || byDay[i].Count > busiestDay.Count {
			busiestDay = &byDay[i]
		}
	}
	var busiestHour *HourCount
	for i := range byHour {
		if busiestHour == nil || byHour[i].Count > busiestHour.Count {
			busiestHour = &byHour[i]
		}
	}

	// Streak calculation
	var maxStreak int
	if len(dailyCounts) > 0 {
		var dates []string
		for d := range dailyCounts {
			dates = append(dates, d)
		}
		sort.Strings(dates)
		currentStreak := 1
		maxStreak = 1
		for i := 1; i < len(dates); i++ {
			d1, _ := time.Parse("2006-01-02", dates[i-1])
			d2, _ := time.Parse("2006-01-02", dates[i])
			if d2.Sub(d1).Hours() == 24 {
				currentStreak++
				if currentStreak > maxStreak {
					maxStreak = currentStreak
				}
			} else {
				currentStreak = 1
			}
		}
	}

	return ActivityPatterns{
		ByDay:       byDay,
		ByHour:      byHour,
		BusiestDay:  busiestDay,
		BusiestHour: busiestHour,
		ActiveDays:  len(dailyCounts),
		MaxStreak:   maxStreak,
	}
}

func (e *InsightsEngine) computeTopSessions(sessions []sessionRow) []NotableSession {
	var top []NotableSession

	// Longest by duration
	var longest *sessionRow
	var longestDur float64
	for i, s := range sessions {
		if s.StartedAt > 0 && s.EndedAt > s.StartedAt {
			dur := s.EndedAt - s.StartedAt
			if dur > longestDur {
				longestDur = dur
				longest = &sessions[i]
			}
		}
	}
	if longest != nil {
		top = append(top, NotableSession{
			Label:     "Longest session",
			SessionID: truncateID(longest.ID, 16),
			Value:     formatDurationCompact(longestDur),
			Date:      time.Unix(int64(longest.StartedAt), 0).Format("Jan 02"),
		})
	}

	// Most messages
	var mostMsgs *sessionRow
	for i, s := range sessions {
		if mostMsgs == nil || s.MessageCount > mostMsgs.MessageCount {
			mostMsgs = &sessions[i]
		}
	}
	if mostMsgs != nil && mostMsgs.MessageCount > 0 {
		top = append(top, NotableSession{
			Label:     "Most messages",
			SessionID: truncateID(mostMsgs.ID, 16),
			Value:     fmt.Sprintf("%d msgs", mostMsgs.MessageCount),
			Date:      formatSessionDate(mostMsgs.StartedAt),
		})
	}

	// Most tokens
	var mostTokens *sessionRow
	var maxTokens int64
	for i, s := range sessions {
		total := s.InputTokens + s.OutputTokens
		if total > maxTokens {
			maxTokens = total
			mostTokens = &sessions[i]
		}
	}
	if mostTokens != nil && maxTokens > 0 {
		top = append(top, NotableSession{
			Label:     "Most tokens",
			SessionID: truncateID(mostTokens.ID, 16),
			Value:     fmt.Sprintf("%d tokens", maxTokens),
			Date:      formatSessionDate(mostTokens.StartedAt),
		})
	}

	// Most tool calls
	var mostTools *sessionRow
	for i, s := range sessions {
		if mostTools == nil || s.ToolCallCount > mostTools.ToolCallCount {
			mostTools = &sessions[i]
		}
	}
	if mostTools != nil && mostTools.ToolCallCount > 0 {
		top = append(top, NotableSession{
			Label:     "Most tool calls",
			SessionID: truncateID(mostTools.ID, 16),
			Value:     fmt.Sprintf("%d calls", mostTools.ToolCallCount),
			Date:      formatSessionDate(mostTools.StartedAt),
		})
	}

	return top
}

// FormatTerminal formats the insights report for terminal display.
func (e *InsightsEngine) FormatTerminal(report *InsightsReport) string {
	if report.Empty {
		src := ""
		if report.SourceFilter != "" {
			src = fmt.Sprintf(" (source: %s)", report.SourceFilter)
		}
		return fmt.Sprintf("  No sessions found in the last %d days%s.", report.Days, src)
	}

	var b strings.Builder
	o := report.Overview

	b.WriteString("\n")
	b.WriteString("  Session Insights\n")
	b.WriteString(fmt.Sprintf("  Last %d days", report.Days))
	if report.SourceFilter != "" {
		b.WriteString(fmt.Sprintf(" (%s)", report.SourceFilter))
	}
	b.WriteString("\n\n")

	// Overview
	b.WriteString("  Overview\n")
	b.WriteString("  " + strings.Repeat("-", 56) + "\n")
	b.WriteString(fmt.Sprintf("  Sessions:          %-12d  Messages:        %d\n", o.TotalSessions, o.TotalMessages))
	b.WriteString(fmt.Sprintf("  Tool calls:        %-12d  User messages:   %d\n", o.TotalToolCalls, o.UserMessages))
	b.WriteString(fmt.Sprintf("  Input tokens:      %-12d  Output tokens:   %d\n", o.TotalInputTokens, o.TotalOutputTokens))
	b.WriteString(fmt.Sprintf("  Total tokens:      %-12d\n", o.TotalTokens))
	if o.TotalHours > 0 {
		b.WriteString(fmt.Sprintf("  Active time:       ~%-11s  Avg session:     ~%s\n",
			formatDurationCompact(o.TotalHours*3600), formatDurationCompact(o.AvgSessionDuration)))
	}
	b.WriteString(fmt.Sprintf("  Avg msgs/session:  %.1f\n\n", o.AvgMsgsPerSession))

	// Models
	if len(report.Models) > 0 {
		b.WriteString("  Models Used\n")
		b.WriteString("  " + strings.Repeat("-", 56) + "\n")
		b.WriteString(fmt.Sprintf("  %-30s %8s %12s\n", "Model", "Sessions", "Tokens"))
		for _, m := range report.Models {
			name := m.Model
			if len(name) > 28 {
				name = name[:28]
			}
			b.WriteString(fmt.Sprintf("  %-30s %8d %12d\n", name, m.Sessions, m.TotalTokens))
		}
		b.WriteString("\n")
	}

	// Platforms
	if len(report.Platforms) > 1 || (len(report.Platforms) == 1 && report.Platforms[0].Platform != "cli") {
		b.WriteString("  Platforms\n")
		b.WriteString("  " + strings.Repeat("-", 56) + "\n")
		b.WriteString(fmt.Sprintf("  %-14s %8s %10s %14s\n", "Platform", "Sessions", "Messages", "Tokens"))
		for _, p := range report.Platforms {
			b.WriteString(fmt.Sprintf("  %-14s %8d %10d %14d\n", p.Platform, p.Sessions, p.Messages, p.TotalTokens))
		}
		b.WriteString("\n")
	}

	// Tools
	if len(report.Tools) > 0 {
		b.WriteString("  Top Tools\n")
		b.WriteString("  " + strings.Repeat("-", 56) + "\n")
		b.WriteString(fmt.Sprintf("  %-28s %8s %8s\n", "Tool", "Calls", "%"))
		limit := len(report.Tools)
		if limit > 15 {
			limit = 15
		}
		for _, t := range report.Tools[:limit] {
			b.WriteString(fmt.Sprintf("  %-28s %8d %7.1f%%\n", t.Tool, t.Count, t.Percentage))
		}
		if len(report.Tools) > 15 {
			b.WriteString(fmt.Sprintf("  ... and %d more tools\n", len(report.Tools)-15))
		}
		b.WriteString("\n")
	}

	// Activity
	if len(report.Activity.ByDay) > 0 {
		b.WriteString("  Activity Patterns\n")
		b.WriteString("  " + strings.Repeat("-", 56) + "\n")
		dayValues := make([]int, len(report.Activity.ByDay))
		for i, d := range report.Activity.ByDay {
			dayValues[i] = d.Count
		}
		bars := barChart(dayValues, 15)
		for i, d := range report.Activity.ByDay {
			b.WriteString(fmt.Sprintf("  %s  %-15s %d\n", d.Day, bars[i], d.Count))
		}
		b.WriteString("\n")

		if report.Activity.ActiveDays > 0 {
			b.WriteString(fmt.Sprintf("  Active days: %d\n", report.Activity.ActiveDays))
		}
		if report.Activity.MaxStreak > 1 {
			b.WriteString(fmt.Sprintf("  Best streak: %d consecutive days\n", report.Activity.MaxStreak))
		}
		b.WriteString("\n")
	}

	// Notable sessions
	if len(report.TopSessions) > 0 {
		b.WriteString("  Notable Sessions\n")
		b.WriteString("  " + strings.Repeat("-", 56) + "\n")
		for _, ts := range report.TopSessions {
			b.WriteString(fmt.Sprintf("  %-20s %-18s (%s, %s)\n", ts.Label, ts.Value, ts.Date, ts.SessionID))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func barChart(values []int, maxWidth int) []string {
	peak := 0
	for _, v := range values {
		if v > peak {
			peak = v
		}
	}
	if peak == 0 {
		bars := make([]string, len(values))
		return bars
	}
	bars := make([]string, len(values))
	for i, v := range values {
		if v > 0 {
			width := int(math.Max(1, float64(v)/float64(peak)*float64(maxWidth)))
			bars[i] = strings.Repeat("#", width)
		}
	}
	return bars
}

func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen]
}

func formatSessionDate(ts float64) string {
	if ts <= 0 {
		return "?"
	}
	return time.Unix(int64(ts), 0).Format("Jan 02")
}

func formatDurationCompact(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.0fm", seconds/60)
	}
	h := int(seconds / 3600)
	m := int(math.Mod(seconds, 3600) / 60)
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}
