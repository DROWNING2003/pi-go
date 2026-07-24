// Package misc provides remaining small core utilities.
package misc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- auth-guidance.ts ---

// AuthGuidance provides user-facing auth setup instructions.
func AuthGuidance(providerID string) string {
	guidance := map[string]string{
		"deepseek":       "Set DEEPSEEK_API_KEY in your environment or run /login",
		"openai":         "Set OPENAI_API_KEY in your environment or run /login",
		"anthropic":      "Set ANTHROPIC_API_KEY in your environment or run /login",
		"google":         "Set GOOGLE_API_KEY or GEMINI_API_KEY in your environment",
		"mistral":        "Set MISTRAL_API_KEY in your environment",
		"groq":           "Set GROQ_API_KEY in your environment",
		"openrouter":     "Set OPENROUTER_API_KEY in your environment",
		"github-copilot": "Run /login to authenticate with GitHub",
	}
	if msg, ok := guidance[providerID]; ok {
		return msg
	}
	return fmt.Sprintf("Set %s_API_KEY in your environment", strings.ToUpper(strings.ReplaceAll(providerID, "-", "_")))
}

// --- remote-catalog-provider.ts ---

// RemoteCatalog fetches model catalogs from remote sources.
type RemoteCatalog struct {
	URL       string
	LastFetch time.Time
}

// FetchModels fetches the remote model catalog (placeholder).
func (c *RemoteCatalog) FetchModels() ([]map[string]interface{}, error) {
	// In production, this would fetch from a remote URL
	return nil, fmt.Errorf("remote catalog not configured")
}

// --- diagnostics.ts ---

// Diagnostic represents a startup or runtime diagnostic.
type Diagnostic struct {
	Type    string `json:"type"` // info, warning, error
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// GatherDiagnostics collects startup diagnostics.
func GatherDiagnostics(cwd, configDir string) []Diagnostic {
	var diags []Diagnostic

	// Check for common issues
	if _, err := os.Stat(filepath.Join(cwd, ".pi")); os.IsNotExist(err) {
		diags = append(diags, Diagnostic{
			Type: "info", Code: "no_pi_dir",
			Message: "No .pi directory found. Create one for project configuration.",
			Path:    cwd,
		})
	}

	return diags
}

// --- export-html/index.ts ---

// ExportHTML exports a session to HTML.
func ExportHTML(messages []json.RawMessage, outputPath string) error {
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>pi Session</title>
<style>body{font-family:system-ui;max-width:800px;margin:0 auto;padding:20px;background:#1a1a2e;color:#e0e0e0}
.user{color:#4fc3f7;margin:10px 0}.assistant{color:#fff;margin:10px 0;padding:10px;background:#16213e;border-radius:8px}
.tool{color:#ffd54f;font-size:0.9em;margin:5px 0}</style></head><body>`)
	html.WriteString(fmt.Sprintf("<h1>pi Session</h1><p>%s</p>", time.Now().Format(time.RFC3339)))

	for _, raw := range messages {
		var header struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
			Name    string          `json:"toolName,omitempty"`
		}
		if json.Unmarshal(raw, &header) != nil {
			continue
		}
		switch header.Role {
		case "user":
			html.WriteString(fmt.Sprintf(`<div class="user">👤 %s</div>`, escapeHTML(extractTextStr(header.Content))))
		case "assistant":
			html.WriteString(fmt.Sprintf(`<div class="assistant">🤖 %s</div>`, escapeHTML(extractTextStr(header.Content))))
		case "toolResult":
			html.WriteString(fmt.Sprintf(`<div class="tool">🔧 %s</div>`, escapeHTML(header.Name)))
		}
	}

	html.WriteString("</body></html>")

	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("pi-session-%d.html", time.Now().Unix()))
	}
	return os.WriteFile(outputPath, []byte(html.String()), 0644)
}

func extractTextStr(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) == nil {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" {
				texts = append(texts, b.Text)
			}
		}
		return strings.Join(texts, " ")
	}
	return string(content)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

var _ = json.Marshal
var _ = time.Now
