package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// WebSearchTool searches the web using the Exa API.
type WebSearchTool struct {
	apiKey string
	client *http.Client
}

type webSearchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

func (w *WebSearchTool) Name() string {
	return "web_search"
}

func (w *WebSearchTool) Description() string {
	return "Searches the web for current information. Returns relevant web results for a query. Use this for real-time data like prices, news, weather, exchange rates, etc."
}

func (w *WebSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query."
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results to return. Defaults to 5."
			}
		},
		"required": ["query"]
	}`)
}

// exaSearchRequest is the payload for POST https://api.exa.ai/search
type exaSearchRequest struct {
	Query      string `json:"query"`
	NumResults int    `json:"numResults,omitempty"`
	Type       string `json:"type,omitempty"`
	Contents   *struct {
		Text *struct {
			MaxCharacters int `json:"maxCharacters,omitempty"`
		} `json:"text,omitempty"`
	} `json:"contents,omitempty"`
}

type exaSearchResponse struct {
	Results []exaResult `json:"results"`
}

type exaResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
	PubDate string  `json:"publishedDate,omitempty"`
}

func (w *WebSearchTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params webSearchArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Query == "" {
		return &tools.Result{Content: "query is required", IsError: true}, nil
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}

	// Fallback to DuckDuckGo if no Exa API key.
	if w.apiKey == "" {
		return w.searchDuckDuckGo(ctx, params.Query, maxResults)
	}

	reqBody := exaSearchRequest{
		Query:      params.Query,
		NumResults: maxResults,
		Type:       "auto",
		Contents: &struct {
			Text *struct {
				MaxCharacters int `json:"maxCharacters,omitempty"`
			} `json:"text,omitempty"`
		}{
			Text: &struct {
				MaxCharacters int `json:"maxCharacters,omitempty"`
			}{MaxCharacters: 1000},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal request: %v", err), IsError: true}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.exa.ai/search", bytes.NewReader(data))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", w.apiKey)

	resp, err := w.client.Do(httpReq)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("search request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &tools.Result{
			Content: fmt.Sprintf("Exa API error (status %d): %s", resp.StatusCode, truncate(string(body), 500)),
			IsError: true,
		}, nil
	}

	var exaResp exaSearchResponse
	if err := json.Unmarshal(body, &exaResp); err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse response: %v", err), IsError: true}, nil
	}

	// Format results as readable text.
	var output string
	for i, r := range exaResp.Results {
		output += fmt.Sprintf("## Result %d: %s\n", i+1, r.Title)
		output += fmt.Sprintf("URL: %s\n", r.URL)
		if r.PubDate != "" {
			output += fmt.Sprintf("Published: %s\n", r.PubDate)
		}
		if r.Text != "" {
			output += fmt.Sprintf("\n%s\n", truncate(r.Text, 500))
		}
		output += "\n---\n\n"
	}

	if output == "" {
		output = "No results found."
	}

	return &tools.Result{Content: output}, nil
}

// WebScrapeTool scrapes a web page using the Firecrawl API.
type WebScrapeTool struct {
	apiKey string
	client *http.Client
}

type webScrapeArgs struct {
	URL    string `json:"url"`
	Format string `json:"format,omitempty"`
}

func (w *WebScrapeTool) Name() string {
	return "web_scrape"
}

func (w *WebScrapeTool) Description() string {
	return "Scrapes the content of a web page using the Firecrawl API. Returns the page content in markdown or plain text."
}

func (w *WebScrapeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL of the web page to scrape."
			},
			"format": {
				"type": "string",
				"description": "Output format: 'markdown' or 'text'. Defaults to 'markdown'."
			}
		},
		"required": ["url"]
	}`)
}

// firecrawlRequest is the payload for POST https://api.firecrawl.dev/v1/scrape
type firecrawlRequest struct {
	URL     string   `json:"url"`
	Formats []string `json:"formats,omitempty"`
}

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Markdown string            `json:"markdown"`
		Content  string            `json:"content"`
		Metadata map[string]string `json:"metadata"`
	} `json:"data"`
}

func (w *WebScrapeTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params webScrapeArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.URL == "" {
		return &tools.Result{Content: "url is required", IsError: true}, nil
	}

	if w.apiKey == "" {
		return &tools.Result{
			Content: "web_scrape is not configured: set FIRECRAWL_API_KEY environment variable",
			IsError: true,
		}, nil
	}

	format := params.Format
	if format == "" {
		format = "markdown"
	}

	reqBody := firecrawlRequest{
		URL:     params.URL,
		Formats: []string{format},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal request: %v", err), IsError: true}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.firecrawl.dev/v1/scrape", bytes.NewReader(data))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+w.apiKey)

	resp, err := w.client.Do(httpReq)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("scrape request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &tools.Result{
			Content: fmt.Sprintf("Firecrawl API error (status %d): %s", resp.StatusCode, truncate(string(body), 500)),
			IsError: true,
		}, nil
	}

	var fcResp firecrawlResponse
	if err := json.Unmarshal(body, &fcResp); err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse response: %v", err), IsError: true}, nil
	}

	if !fcResp.Success {
		return &tools.Result{Content: "Firecrawl scrape failed", IsError: true}, nil
	}

	content := fcResp.Data.Markdown
	if content == "" {
		content = fcResp.Data.Content
	}
	if content == "" {
		content = "No content extracted from page."
	}

	// Add metadata header.
	var output string
	if title, ok := fcResp.Data.Metadata["title"]; ok && title != "" {
		output = fmt.Sprintf("# %s\n\n", title)
	}
	output += truncate(content, 50000)

	return &tools.Result{Content: output}, nil
}

// searchDuckDuckGo performs a web search using DuckDuckGo's HTML interface (no API key needed).
func (w *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, maxResults int) (*tools.Result, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + urlEncode(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Hera/1.0)")

	resp, err := w.client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("search request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	html := string(body)
	results := parseDDGResults(html, maxResults)

	if len(results) == 0 {
		return &tools.Result{Content: "No results found for: " + query}, nil
	}

	var output string
	for i, r := range results {
		output += fmt.Sprintf("## Result %d: %s\n", i+1, r.title)
		output += fmt.Sprintf("URL: %s\n", r.url)
		if r.snippet != "" {
			output += fmt.Sprintf("\n%s\n", r.snippet)
		}
		output += "\n---\n\n"
	}

	return &tools.Result{Content: output}, nil
}

type ddgResult struct {
	title   string
	url     string
	snippet string
}

// parseDDGResults extracts search results from DuckDuckGo HTML response.
func parseDDGResults(html string, max int) []ddgResult {
	var results []ddgResult

	// DuckDuckGo HTML results are in <a class="result__a" ...> tags
	remaining := html
	for len(results) < max {
		// Find result link
		linkIdx := strings.Index(remaining, `class="result__a"`)
		if linkIdx == -1 {
			break
		}
		remaining = remaining[linkIdx:]

		// Extract href
		hrefStart := strings.Index(remaining, `href="`)
		if hrefStart == -1 {
			break
		}
		remaining = remaining[hrefStart+6:]
		hrefEnd := strings.Index(remaining, `"`)
		if hrefEnd == -1 {
			break
		}
		rawURL := remaining[:hrefEnd]
		remaining = remaining[hrefEnd:]

		// Clean DuckDuckGo redirect URL
		actualURL := rawURL
		if udIdx := strings.Index(rawURL, "uddg="); udIdx != -1 {
			actualURL = rawURL[udIdx+5:]
			if ampIdx := strings.Index(actualURL, "&"); ampIdx != -1 {
				actualURL = actualURL[:ampIdx]
			}
			actualURL = urlDecode(actualURL)
		}

		// Extract title (text between > and </a>)
		titleStart := strings.Index(remaining, ">")
		if titleStart == -1 {
			break
		}
		remaining = remaining[titleStart+1:]
		titleEnd := strings.Index(remaining, "</a>")
		if titleEnd == -1 {
			break
		}
		title := stripHTMLTags(remaining[:titleEnd])
		remaining = remaining[titleEnd:]

		// Extract snippet from result__snippet
		snippet := ""
		snippetIdx := strings.Index(remaining, `class="result__snippet"`)
		if snippetIdx != -1 && snippetIdx < 500 {
			snipRemaining := remaining[snippetIdx:]
			snipStart := strings.Index(snipRemaining, ">")
			if snipStart != -1 {
				snipRemaining = snipRemaining[snipStart+1:]
				snipEnd := strings.Index(snipRemaining, "</")
				if snipEnd != -1 {
					snippet = stripHTMLTags(snipRemaining[:snipEnd])
				}
			}
		}

		if title != "" && actualURL != "" {
			results = append(results, ddgResult{
				title:   strings.TrimSpace(title),
				url:     strings.TrimSpace(actualURL),
				snippet: strings.TrimSpace(snippet),
			})
		}
	}
	return results
}

// stripHTMLTags is defined in browser.go (shared across package).

func urlEncode(s string) string {
	var out strings.Builder
	for _, b := range []byte(s) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' || b == '~' {
			out.WriteByte(b)
		} else if b == ' ' {
			out.WriteByte('+')
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}

func urlDecode(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi := unhex(s[i+1])
			lo := unhex(s[i+2])
			if hi >= 0 && lo >= 0 {
				out.WriteByte(byte(hi<<4 | lo))
				i += 2
				continue
			}
		} else if s[i] == '+' {
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// WebFetchTool fetches the content of any URL directly (no API key needed).
type WebFetchTool struct {
	client *http.Client
}

func (w *WebFetchTool) Name() string { return "web_fetch" }

func (w *WebFetchTool) Description() string {
	return "Fetches the content of a web page by URL. Returns the raw HTML or text content. Use this to read web pages, APIs, or any URL directly."
}

func (w *WebFetchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL to fetch."
			}
		},
		"required": ["url"]
	}`)
}

func (w *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.URL == "" {
		return &tools.Result{Content: "url is required", IsError: true}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Hera/1.0)")
	req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")

	resp, err := w.client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("fetch failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	content := string(body)

	// If it looks like HTML, strip tags for readability.
	if strings.Contains(content, "<html") || strings.Contains(content, "<HTML") {
		content = stripHTMLTags(content)
		// Collapse whitespace.
		lines := strings.Split(content, "\n")
		var clean []string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" {
				clean = append(clean, l)
			}
		}
		content = strings.Join(clean, "\n")
	}

	return &tools.Result{Content: truncate(content, 50000)}, nil
}

// RegisterWeb registers web_search, web_scrape, and web_fetch tools.
func RegisterWeb(registry *tools.Registry, exaKey, firecrawlKey string) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	registry.Register(&WebSearchTool{apiKey: exaKey, client: httpClient})
	registry.Register(&WebScrapeTool{apiKey: firecrawlKey, client: httpClient})
	registry.Register(&WebFetchTool{client: httpClient})
}

// truncate limits a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
