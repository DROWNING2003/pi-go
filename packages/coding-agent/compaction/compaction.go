// Package compaction provides context compaction for long sessions.
package compaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	storagesession "github.com/DROWNING2003/pi-go/packages/storage/session"
)

// Result holds the result of a compaction operation.
type Result struct {
	Success      bool
	TokensBefore int
	TokensAfter  int
	Summary      string
	NewSession   *storagesession.Session
}

// Prepare compacts old messages into a summary to save context window space.
func Prepare(messages []json.RawMessage, maxTokens int) (*Result, error) {
	if len(messages) == 0 {
		return &Result{Success: true}, nil
	}

	// Estimate current tokens
	tokensBefore := estimateTokens(messages)
	if tokensBefore <= maxTokens {
		return &Result{Success: true, TokensBefore: tokensBefore, TokensAfter: tokensBefore}, nil
	}

	// Build a summary of the conversation
	summary := buildSummary(messages)

	// Create compaction entry
	entry := storagesession.CreateCompactionEntry(summary, tokensBefore, "", nil)

	result := &Result{
		Success:      true,
		TokensBefore: tokensBefore,
		TokensAfter:  len(summary) / 4,
		Summary:      summary,
	}
	_ = entry

	return result, nil
}

// ShouldCompact checks if messages exceed the token budget and should be compacted.
func ShouldCompact(messages []json.RawMessage, maxTokens int) bool {
	return estimateTokens(messages) > maxTokens
}

func estimateTokens(messages []json.RawMessage) int {
	total := 0
	for _, m := range messages {
		total += len(m) / 4
	}
	return total
}

func buildSummary(messages []json.RawMessage) string {
	var parts []string
	parts = append(parts, "The conversation history before this point was compacted into the following summary:")
	parts = append(parts, "")
	parts = append(parts, "<summary>")

	userCount := 0
	asstCount := 0
	toolCount := 0

	for _, raw := range messages {
		var header struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(raw, &header) != nil {
			continue
		}
		switch header.Role {
		case "user":
			userCount++
			if text := extractText(header.Content); text != "" {
				if len(text) > 200 {
					text = text[:200] + "..."
				}
				parts = append(parts, "User: "+text)
			}
		case "assistant":
			asstCount++
			if text := extractText(header.Content); text != "" {
				if len(text) > 300 {
					text = text[:300] + "..."
				}
				parts = append(parts, "Assistant: "+text)
			}
		case "toolResult":
			toolCount++
		}
	}

	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("(Compacted %d messages: %d user, %d assistant, %d tool results)",
		userCount+asstCount+toolCount, userCount, asstCount, toolCount))
	parts = append(parts, "</summary>")

	return strings.Join(parts, "\n")
}

func extractText(content json.RawMessage) string {
	// Try string first
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	// Try array of content blocks
	var blocks []model.ContentBlock
	if json.Unmarshal(content, &blocks) != nil {
		return ""
	}
	var texts []string
	for _, b := range blocks {
		if b.Type == model.ContentTypeText && b.Text != "" {
			texts = append(texts, b.Text)
		}
	}
	return strings.Join(texts, " ")
}

// --- Automatic compaction ---

// AutoConfig holds auto-compaction settings.
type AutoConfig struct {
	Enabled    bool
	MaxTokens  int
	TriggerPct float64 // Trigger when context reaches this % of max
}

// DefaultAutoConfig returns sensible defaults.
func DefaultAutoConfig() AutoConfig {
	return AutoConfig{
		Enabled:    true,
		MaxTokens:  50000,
		TriggerPct: 0.8,
	}
}

// AutoCompaction runs compaction if the context is getting too full.
func AutoCompaction(ctx context.Context, messages []json.RawMessage, cfg AutoConfig, streamFn func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent) (*Result, error) {
	if !cfg.Enabled {
		return &Result{Success: true}, nil
	}

	currentTokens := estimateTokens(messages)
	threshold := int(float64(cfg.MaxTokens) * cfg.TriggerPct)

	if currentTokens < threshold {
		return &Result{Success: true, TokensBefore: currentTokens, TokensAfter: currentTokens}, nil
	}

	return Prepare(messages, cfg.MaxTokens)
}

// Ensure imports used
var _ = context.Background
var _ = time.Now
var _ = fmt.Sprintf
