package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// WebFetchTool fetches web page content via HTTP GET.
type WebFetchTool struct {
	timeout time.Duration
}

// NewWebFetchTool creates a web fetch tool.
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{timeout: 15 * time.Second}
}

func (t *WebFetchTool) Name() string { return "web_fetch" }
func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL. Returns the response body as text."
}
func (t *WebFetchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {"type": "string", "description": "URL to fetch"},
			"maxChars": {"type": "number", "description": "Maximum characters to return (default: 10000)"}
		},
		"required": ["url"]
	}`)
}

func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		URL      string `json:"url"`
		MaxChars int    `json:"maxChars"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("invalid url: " + err.Error())}, IsError: true}, nil
	}
	if params.MaxChars <= 0 {
		params.MaxChars = 10000
	}

	client := &http.Client{Timeout: t.timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("invalid request: " + err.Error())}, IsError: true}, nil
	}
	req.Header.Set("User-Agent", "pi-go/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("fetch error: " + err.Error())}, IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(params.MaxChars+1000)))
	if err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("read error: " + err.Error())}, IsError: true}, nil
	}

	text := stripHTML(string(body))
	if len(text) > params.MaxChars {
		text = text[:params.MaxChars] + fmt.Sprintf("\n... (truncated, %d bytes total)", len(body))
	}

	return &Result{
		Content: []model.ContentBlock{model.NewTextContent(
			fmt.Sprintf("URL: %s\nStatus: %d\n\n%s", params.URL, resp.StatusCode, text),
		)},
	}, nil
}

func stripHTML(s string) string {
	// Simple HTML tag stripping
	inTag := false
	var result strings.Builder
	for _, c := range s {
		if c == '<' {
			inTag = true
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(c)
		}
	}
	// Normalize whitespace
	lines := strings.Split(result.String(), "\n")
	var clean []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}
