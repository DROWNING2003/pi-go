package provider

var builtinProviders = []*ProviderConfig{
	{
		ID:          "deepseek",
		Name:        "DeepSeek",
		BaseURL:     "https://api.deepseek.com",
		API:         "openai-completions",
		AuthEnvVars: []string{"DEEPSEEK_API_KEY"},
		Models: []ModelConfig{
			{ID: "deepseek-chat", Name: "DeepSeek Chat", Reasoning: false, Input: []string{"text"}, ContextWindow: 65536, MaxTokens: 8192},
			{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash", Reasoning: false, Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 16384},
			{ID: "deepseek-reasoner", Name: "DeepSeek Reasoner", Reasoning: true, Input: []string{"text"}, ContextWindow: 65536, MaxTokens: 8192,
				Cost: ModelCost{Input: 0.55, Output: 2.19, CacheRead: 0.14, CacheWrite: 0.55}},
		},
	},
	{
		ID:          "openai",
		Name:        "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
		API:         "openai-completions",
		AuthEnvVars: []string{"OPENAI_API_KEY"},
		Models: []ModelConfig{
			{ID: "gpt-4o", Name: "GPT-4o", Reasoning: false, Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Reasoning: false, Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
			{ID: "o1", Name: "o1", Reasoning: true, Input: []string{"text"}, ContextWindow: 200000, MaxTokens: 100000},
			{ID: "o3-mini", Name: "o3 Mini", Reasoning: true, Input: []string{"text"}, ContextWindow: 200000, MaxTokens: 100000},
		},
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic",
		BaseURL:     "https://api.anthropic.com/v1",
		API:         "anthropic-messages",
		AuthEnvVars: []string{"ANTHROPIC_API_KEY"},
		Models: []ModelConfig{
			{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Reasoning: false, Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 16384},
			{ID: "claude-opus-4-8", Name: "Claude Opus 4.8", Reasoning: true, Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 32768},
		},
	},
	{
		ID:          "google",
		Name:        "Google",
		BaseURL:     "https://generativelanguage.googleapis.com",
		API:         "google-generative-ai",
		AuthEnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"},
		Models: []ModelConfig{
			{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Reasoning: false, Input: []string{"text", "image"}, ContextWindow: 1048576, MaxTokens: 8192},
			{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Reasoning: true, Input: []string{"text", "image"}, ContextWindow: 1048576, MaxTokens: 16384},
		},
	},
	{
		ID:          "faux",
		Name:        "Faux (testing)",
		BaseURL:     "",
		API:         "faux",
		AuthEnvVars: []string{},
		Models: []ModelConfig{
			{ID: "faux-1", Name: "Faux Model", Reasoning: false, Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
		},
	},
}
