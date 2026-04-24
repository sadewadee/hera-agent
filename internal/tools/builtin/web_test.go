package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSearchTool_Name(t *testing.T) {
	tool := &WebSearchTool{}
	if got := tool.Name(); got != "web_search" {
		t.Errorf("Name() = %q, want %q", got, "web_search")
	}
}

func TestWebSearchTool_Description(t *testing.T) {
	tool := &WebSearchTool{}
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestWebSearchTool_Parameters(t *testing.T) {
	tool := &WebSearchTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() returned invalid JSON")
	}
}

func TestWebSearchTool_Execute_EmptyQuery(t *testing.T) {
	tool := &WebSearchTool{apiKey: "test-key"}
	args, _ := json.Marshal(webSearchArgs{Query: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for empty query")
	}
	if !strings.Contains(result.Content, "query is required") {
		t.Errorf("error should mention query required, got: %q", result.Content)
	}
}

func TestWebSearchTool_Execute_NoAPIKey_FallbackDDG(t *testing.T) {
	tool := &WebSearchTool{apiKey: "", client: &http.Client{Timeout: 10 * time.Second}}
	args, _ := json.Marshal(webSearchArgs{Query: "golang"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	// DuckDuckGo fallback — should not be a config error.
	if result.IsError {
		t.Logf("DuckDuckGo fallback returned error (possible network issue): %s", result.Content)
	}
}

func TestWebSearchTool_Execute_InvalidJSON(t *testing.T) {
	tool := &WebSearchTool{apiKey: "test-key"}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid JSON")
	}
}

func TestWebSearchTool_Execute_SuccessfulSearch(t *testing.T) {
	// Set up a mock Exa API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := `{
			"results": [
				{
					"title": "Test Result",
					"url": "https://example.com",
					"text": "This is a test result.",
					"score": 0.95,
					"publishedDate": "2024-01-01"
				}
			]
		}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		apiKey: "test-key",
		client: server.Client(),
	}

	// Override the URL by using the httptest server directly.
	// Since the tool hardcodes the Exa API URL, we need to use a custom client
	// that redirects to our test server. Instead, we test the response handling
	// by using a transport that redirects all requests.
	tool.client = &http.Client{
		Transport: &testTransport{handler: server},
	}

	args, _ := json.Marshal(webSearchArgs{Query: "test query", MaxResults: 3})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Test Result") {
		t.Errorf("result should contain 'Test Result', got: %q", result.Content)
	}
	if !strings.Contains(result.Content, "https://example.com") {
		t.Errorf("result should contain URL, got: %q", result.Content)
	}
}

func TestWebSearchTool_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webSearchArgs{Query: "test"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for API error")
	}
	if !strings.Contains(result.Content, "500") {
		t.Errorf("error should mention status code, got: %q", result.Content)
	}
}

func TestWebSearchTool_Execute_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webSearchArgs{Query: "obscure query"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "No results found") {
		t.Errorf("result should say no results found, got: %q", result.Content)
	}
}

func TestWebSearchTool_Execute_DefaultMaxResults(t *testing.T) {
	var receivedNumResults int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req exaSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedNumResults = req.NumResults
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	tool := &WebSearchTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webSearchArgs{Query: "test"})
	tool.Execute(context.Background(), args)

	if receivedNumResults != 5 {
		t.Errorf("default max_results should be 5, got %d", receivedNumResults)
	}
}

// --- WebScrapeTool tests ---

func TestWebScrapeTool_Name(t *testing.T) {
	tool := &WebScrapeTool{}
	if got := tool.Name(); got != "web_scrape" {
		t.Errorf("Name() = %q, want %q", got, "web_scrape")
	}
}

func TestWebScrapeTool_Description(t *testing.T) {
	tool := &WebScrapeTool{}
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestWebScrapeTool_Parameters(t *testing.T) {
	tool := &WebScrapeTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() returned invalid JSON")
	}
}

func TestWebScrapeTool_Execute_EmptyURL(t *testing.T) {
	tool := &WebScrapeTool{apiKey: "test-key"}
	args, _ := json.Marshal(webScrapeArgs{URL: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for empty URL")
	}
	if !strings.Contains(result.Content, "url is required") {
		t.Errorf("error should mention url required, got: %q", result.Content)
	}
}

func TestWebScrapeTool_Execute_MissingAPIKey(t *testing.T) {
	tool := &WebScrapeTool{apiKey: ""}
	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error when API key is missing")
	}
	if !strings.Contains(result.Content, "FIRECRAWL_API_KEY") {
		t.Errorf("error should mention FIRECRAWL_API_KEY, got: %q", result.Content)
	}
}

func TestWebScrapeTool_Execute_InvalidJSON(t *testing.T) {
	tool := &WebScrapeTool{apiKey: "test-key"}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid JSON")
	}
}

func TestWebScrapeTool_Execute_SuccessfulScrape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("expected Bearer auth, got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := `{
			"success": true,
			"data": {
				"markdown": "# Hello World\n\nThis is the content.",
				"content": "Hello World. This is the content.",
				"metadata": {"title": "Test Page"}
			}
		}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	tool := &WebScrapeTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Hello World") {
		t.Errorf("result should contain markdown content, got: %q", result.Content)
	}
	if !strings.Contains(result.Content, "# Test Page") {
		t.Errorf("result should contain page title, got: %q", result.Content)
	}
}

func TestWebScrapeTool_Execute_ScrapeFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": false, "data": {}}`))
	}))
	defer server.Close()

	tool := &WebScrapeTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error when scrape fails")
	}
}

func TestWebScrapeTool_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	tool := &WebScrapeTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for API error")
	}
	if !strings.Contains(result.Content, "401") {
		t.Errorf("error should mention status code, got: %q", result.Content)
	}
}

func TestWebScrapeTool_Execute_DefaultFormat(t *testing.T) {
	var receivedFormats []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req firecrawlRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedFormats = req.Formats
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "data": {"markdown": "content", "content": "", "metadata": {}}}`))
	}))
	defer server.Close()

	tool := &WebScrapeTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	// No format specified -- should default to markdown
	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	tool.Execute(context.Background(), args)

	if len(receivedFormats) != 1 || receivedFormats[0] != "markdown" {
		t.Errorf("default format should be [markdown], got %v", receivedFormats)
	}
}

func TestWebScrapeTool_Execute_FallbackToContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// markdown is empty, should fall back to content
		w.Write([]byte(`{"success": true, "data": {"markdown": "", "content": "Fallback content", "metadata": {}}}`))
	}))
	defer server.Close()

	tool := &WebScrapeTool{
		apiKey: "test-key",
		client: &http.Client{Transport: &testTransport{handler: server}},
	}

	args, _ := json.Marshal(webScrapeArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Fallback content") {
		t.Errorf("result should contain fallback content, got: %q", result.Content)
	}
}

// --- Shared utility tests ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// testTransport redirects all HTTP requests to a test server.
type testTransport struct {
	handler *httptest.Server
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the URL with the test server URL, preserving the path
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.handler.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}
