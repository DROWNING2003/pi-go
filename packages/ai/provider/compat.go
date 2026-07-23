package provider

import "strings"

// CompatConfig holds provider-specific compatibility settings for
// OpenAI-compatible Completions APIs. These control payload formatting
// and feature detection that vary across providers.
type CompatConfig struct {
	SupportsStore                               bool
	SupportsDeveloperRole                       bool
	SupportsReasoningEffort                     bool
	SupportsUsageInStreaming                    bool
	MaxTokensField                              string // "max_completion_tokens" or "max_tokens"
	RequiresToolResultName                      bool
	RequiresAssistantAfterToolResult            bool
	RequiresThinkingAsText                      bool
	RequiresReasoningContentOnAssistantMessages bool
	ThinkingFormat                              string
	SupportsStrictMode                          bool
	SupportsLongCacheRetention                  bool
}

// DefaultCompat returns default openai compat settings.
func DefaultCompat() CompatConfig {
	return CompatConfig{
		SupportsStore:              true,
		SupportsDeveloperRole:      true,
		SupportsReasoningEffort:    true,
		SupportsUsageInStreaming:   true,
		MaxTokensField:             "max_completion_tokens",
		SupportsStrictMode:         true,
		SupportsLongCacheRetention: true,
	}
}

// DetectCompat auto-detects compat settings from provider ID and base URL.
func DetectCompat(providerID, baseURL string) CompatConfig {
	c := DefaultCompat()

	// Non-standard providers that don't support store/developerRole/reasoningEffort
	nonStandard := isNonStandardProvider(providerID, baseURL)

	isOpenRouter := providerID == "openrouter" || strings.Contains(baseURL, "openrouter.ai")
	isDeepSeek := providerID == "deepseek" || strings.Contains(baseURL, "deepseek.com")
	isTogether := providerID == "together" || strings.Contains(baseURL, "together.xyz")
	isZai := providerID == "zai" || strings.Contains(baseURL, "z.ai")
	isMoonshot := providerID == "moonshotai" || strings.Contains(baseURL, "moonshot.ai")
	isCloudflare := providerID == "cloudflare-workers-ai" || strings.Contains(baseURL, "cloudflare.com")
	isNvidia := providerID == "nvidia" || strings.Contains(baseURL, "nvidia.com")
	isAntLing := providerID == "ant-ling" || strings.Contains(baseURL, "ant-ling.com")
	isGrok := providerID == "xai" || strings.Contains(baseURL, "api.x.ai")
	isCerebras := providerID == "cerebras" || strings.Contains(baseURL, "cerebras.ai")
	isFireworks := providerID == "fireworks" || strings.Contains(baseURL, "fireworks.ai")
	isGroq := providerID == "groq" || strings.Contains(baseURL, "groq.com")
	isHuggingFace := providerID == "huggingface" || strings.Contains(baseURL, "huggingface.co")
	isMistral := providerID == "mistral" || strings.Contains(baseURL, "mistral.ai")

	if nonStandard {
		c.SupportsStore = false
		c.SupportsDeveloperRole = isOpenRouter && !isMistral
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false
	}

	// Provider-specific overrides
	switch {
	case isDeepSeek:
		c.SupportsReasoningEffort = false
		c.SupportsDeveloperRole = false
		c.RequiresReasoningContentOnAssistantMessages = true
		c.ThinkingFormat = "deepseek"
		c.SupportsStore = false
		c.SupportsStrictMode = false

	case isTogether:
		c.MaxTokensField = "max_tokens"
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false
		c.SupportsLongCacheRetention = false
		c.ThinkingFormat = "together"

	case isZai:
		c.SupportsReasoningEffort = false
		c.ThinkingFormat = "zai"

	case isMoonshot:
		c.MaxTokensField = "max_tokens"
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false

	case isCloudflare:
		c.MaxTokensField = "max_tokens"
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false
		c.SupportsLongCacheRetention = false

	case isNvidia:
		c.MaxTokensField = "max_tokens"
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false
		c.SupportsLongCacheRetention = false

	case isGrok:
		c.SupportsReasoningEffort = false

	case isCerebras:
		c.SupportsStore = false
		c.SupportsDeveloperRole = false
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false

	case isFireworks:
		c.SupportsStore = false
		c.SupportsDeveloperRole = false
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false

	case isGroq:
		c.SupportsStore = false
		c.SupportsDeveloperRole = false
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false

	case isHuggingFace:
		c.SupportsStore = false
		c.SupportsDeveloperRole = false
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false

	case isMistral:
		c.SupportsDeveloperRole = false

	case isAntLing:
		c.MaxTokensField = "max_tokens"
		c.SupportsReasoningEffort = false
		c.SupportsStrictMode = false
		c.ThinkingFormat = "ant-ling"

	case isOpenRouter:
		c.ThinkingFormat = "openrouter"
	}

	return c
}

func isNonStandardProvider(providerID, baseURL string) bool {
	nonStandardIDs := map[string]bool{
		"deepseek": true, "zai": true, "zai-coding-cn": true,
		"moonshotai": true, "moonshotai-cn": true,
		"minimax": true, "minimax-cn": true,
		"fireworks": true, "together": true, "nvidia": true,
		"groq": true, "cerebras": true, "xai": true,
		"huggingface": true, "opencode": true, "opencode-go": true,
		"cloudflare-workers-ai": true, "cloudflare-ai-gateway": true,
		"kimi-coding": true, "qwen-token-plan": true, "qwen-token-plan-cn": true,
		"xiaomi": true, "xiaomi-token-plan-cn": true, "xiaomi-token-plan-ams": true, "xiaomi-token-plan-sgp": true,
		"ant-ling": true,
	}
	if nonStandardIDs[providerID] {
		return true
	}
	nonStandardURLs := []string{
		"deepseek.com", "z.ai", "moonshot.ai",
		"cerebras.ai", "api.x.ai", "chutes.ai",
		"fireworks.ai", "together.xyz", "groq.com",
		"huggingface.co", "opencode.ai", "cloudflare.com",
		"nvidia.com", "minimax", "ant-ling.com",
	}
	for _, u := range nonStandardURLs {
		if strings.Contains(baseURL, u) {
			return true
		}
	}
	return false
}
