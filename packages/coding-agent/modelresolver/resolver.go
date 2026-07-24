// Package modelresolver provides model resolution with fuzzy matching.
package modelresolver

import (
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// DefaultModelPerProvider maps known providers to their default models.
var DefaultModelPerProvider = map[string]string{
	"amazon-bedrock":  "us.anthropic.claude-opus-4-6-v1",
	"anthropic":       "claude-opus-4-8",
	"openai":          "gpt-5.5",
	"openai-codex":    "gpt-5.5",
	"deepseek":        "deepseek-v4-pro",
	"google":          "gemini-3.1-pro-preview",
	"google-vertex":   "gemini-3.1-pro-preview",
	"github-copilot":  "gpt-5.4",
	"openrouter":      "moonshotai/kimi-k2.6",
	"xai":             "grok-4.5",
	"groq":            "openai/gpt-oss-120b",
	"cerebras":        "zai-glm-4.7",
	"zai":             "glm-5.1",
	"mistral":         "devstral-medium-latest",
	"minimax":         "MiniMax-M2.7",
	"moonshotai":      "kimi-k2.6",
	"huggingface":     "moonshotai/Kimi-K2.6",
	"fireworks":       "accounts/fireworks/models/kimi-k2p6",
	"together":        "moonshotai/Kimi-K2.6",
	"kimi-coding":     "kimi-for-coding",
	"qwen-token-plan": "qwen3.7-max",
	"xiaomi":          "mimo-v2.5-pro",
	"nvidia":          "nvidia/nemotron-3-super-120b-a12b",
}

// ScopedModel holds a resolved model with optional thinking level.
type ScopedModel struct {
	Model         *provider.ProviderModel
	ThinkingLevel model.ThinkingLevel
}

// ResolvePattern resolves a model pattern (e.g., "deepseek/deepseek-chat:high") to a ScopedModel.
func ResolvePattern(pattern string, reg *provider.Registry) *ScopedModel {
	if pattern == "" {
		return nil
	}

	// Parse thinking level suffix: model:level
	thinkingLevel := model.ThinkingOff
	if idx := strings.LastIndex(pattern, ":"); idx > 0 {
		suffix := pattern[idx+1:]
		switch suffix {
		case "off", "minimal", "low", "medium", "high", "xhigh", "max":
			thinkingLevel = model.ThinkingLevel(suffix)
			pattern = pattern[:idx]
		}
	}

	// Try exact match
	if m := reg.ResolveModel(pattern); m != nil {
		return &ScopedModel{Model: m, ThinkingLevel: thinkingLevel}
	}

	// Try fuzzy: just provider name → get default model
	if !strings.Contains(pattern, "/") {
		if prov := reg.GetProvider(pattern); prov != nil {
			if defaultID, ok := DefaultModelPerProvider[pattern]; ok {
				if m := reg.GetModel(pattern, defaultID); m != nil {
					return &ScopedModel{Model: m, ThinkingLevel: thinkingLevel}
				}
			}
			// Use first model of provider
			if len(prov.Models) > 0 {
				m := reg.ResolveModel(pattern + "/" + prov.Models[0].ID)
				return &ScopedModel{Model: m, ThinkingLevel: thinkingLevel}
			}
		}
	}

	// Try fuzzy matching: search all models for partial match
	pattern = strings.ToLower(pattern)
	for _, pid := range reg.ListProviders() {
		prov := reg.GetProvider(pid)
		if prov == nil {
			continue
		}
		for _, mc := range prov.Models {
			fullID := strings.ToLower(pid + "/" + mc.ID)
			if strings.Contains(fullID, pattern) || strings.Contains(strings.ToLower(mc.Name), pattern) {
				m := reg.ResolveModel(pid + "/" + mc.ID)
				return &ScopedModel{Model: m, ThinkingLevel: thinkingLevel}
			}
		}
	}

	return nil
}

// GetDefaultModel returns the default model for a provider.
func GetDefaultModel(providerID string, reg *provider.Registry) *provider.ProviderModel {
	if defaultID, ok := DefaultModelPerProvider[providerID]; ok {
		if m := reg.GetModel(providerID, defaultID); m != nil {
			return m
		}
	}
	return reg.ResolveModel(providerID)
}

// CycleModel returns the next model after the current one.
func CycleModel(current *provider.ProviderModel, reg *provider.Registry) *provider.ProviderModel {
	prov := reg.GetProvider(current.Provider)
	if prov == nil {
		return current
	}

	models := prov.Models
	for i, mc := range models {
		if mc.ID == current.ID {
			next := models[(i+1)%len(models)]
			return reg.ResolveModel(current.Provider + "/" + next.ID)
		}
	}

	return current
}
