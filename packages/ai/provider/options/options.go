// Package options provides simple streaming options matching TS simple-options.ts.
package options

import (
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/ai/util"
)

const contextSafetyTokens = 4096
const minMaxTokens = 1

// BuildStreamOptions builds StreamOptions from SimpleStreamOptions with
// automatic context window clamping matching TS buildBaseOptions.
func BuildStreamOptions(m *provider.ProviderModel, c *provider.Context, maxTokens int, temperature float64) *provider.StreamOptions {
	clamped := clampMaxTokens(m, c, maxTokens)
	return &provider.StreamOptions{
		MaxTokens:   clamped,
		Temperature: temperature,
	}
}

func clampMaxTokens(m *provider.ProviderModel, c *provider.Context, maxTokens int) int {
	if m.ContextWindow <= 0 {
		return max(maxTokens, minMaxTokens)
	}
	ctxTokens := estimateContextTokens(c)
	available := int(m.ContextWindow) - ctxTokens - contextSafetyTokens
	result := maxTokens
	if result > available {
		result = available
	}
	if result < minMaxTokens {
		result = minMaxTokens
	}
	return result
}

func estimateContextTokens(c *provider.Context) int {
	total := 0
	if c.SystemPrompt != "" {
		total += util.EstimateTokens(c.SystemPrompt)
	}
	for _, msg := range c.Messages {
		total += util.EstimateTokens(string(msg))
	}
	return total
}

// ClampReasoning clamps xhigh/max to high.
func ClampReasoning(level model.ThinkingLevel) model.ThinkingLevel {
	if level == model.ThinkingXHigh || level == model.ThinkingMax {
		return model.ThinkingHigh
	}
	return level
}

// AdjustMaxTokensForThinking adjusts max tokens to accommodate thinking budget.
func AdjustMaxTokensForThinking(baseMax, modelMax int, level model.ThinkingLevel) (maxTokens, thinkingBudget int) {
	budgets := map[model.ThinkingLevel]int{
		model.ThinkingMinimal: 1024,
		model.ThinkingLow:     2048,
		model.ThinkingMedium:  8192,
		model.ThinkingHigh:    16384,
	}
	minOutput := 1024
	clampedLevel := ClampReasoning(level)
	thinkingBudget = budgets[clampedLevel]

	if baseMax == 0 {
		maxTokens = modelMax
	} else {
		maxTokens = baseMax + thinkingBudget
		if maxTokens > modelMax {
			maxTokens = modelMax
		}
	}
	if maxTokens <= thinkingBudget {
		thinkingBudget = max(0, maxTokens-minOutput)
	}
	return
}
