package provider

var builtinProviders = []*ProviderConfig{
	{ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"DEEPSEEK_API_KEY"}}, AuthEnvVars: []string{"DEEPSEEK_API_KEY"}, Models: []ModelConfig{
		{ID: "deepseek-chat", Name: "DeepSeek Chat", Input: []string{"text"}, ContextWindow: 65536, MaxTokens: 8192},
		{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 16384},
		{ID: "deepseek-reasoner", Name: "DeepSeek Reasoner", Reasoning: true, Input: []string{"text"}, ContextWindow: 65536, MaxTokens: 8192, Cost: ModelCost{Input: 0.55, Output: 2.19}},
	}},
	{ID: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"OPENAI_API_KEY"}}, AuthEnvVars: []string{"OPENAI_API_KEY"}, Models: []ModelConfig{
		{ID: "gpt-4o", Name: "GPT-4o", Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
		{ID: "o1", Name: "o1", Reasoning: true, Input: []string{"text"}, ContextWindow: 200000, MaxTokens: 100000},
		{ID: "o3-mini", Name: "o3 Mini", Reasoning: true, Input: []string{"text"}, ContextWindow: 200000, MaxTokens: 100000},
	}},
	{ID: "openai-codex", Name: "OpenAI Codex", BaseURL: "https://api.openai.com/v1", API: "openai-responses", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"OPENAI_API_KEY"}}, AuthEnvVars: []string{"OPENAI_API_KEY"}, Models: []ModelConfig{
		{ID: "gpt-5.5", Name: "GPT-5.5", Reasoning: true, Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 128000},
	}},
	{ID: "azure-openai", Name: "Azure OpenAI", BaseURL: "https://{RESOURCE}.openai.azure.com/openai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"AZURE_OPENAI_API_KEY"}}, AuthEnvVars: []string{"AZURE_OPENAI_API_KEY"}, Models: []ModelConfig{
		{ID: "gpt-4o", Name: "GPT-4o", Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
	}},
	{ID: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1", API: "anthropic-messages", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"ANTHROPIC_API_KEY"}}, AuthEnvVars: []string{"ANTHROPIC_API_KEY"}, Models: []ModelConfig{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 16384},
		{ID: "claude-opus-4-8", Name: "Claude Opus 4.8", Reasoning: true, Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 32768},
	}},
	{ID: "google", Name: "Google", BaseURL: "https://generativelanguage.googleapis.com", API: "google-generative-ai", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"}}, AuthEnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"}, Models: []ModelConfig{
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Input: []string{"text", "image"}, ContextWindow: 1048576, MaxTokens: 8192},
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Reasoning: true, Input: []string{"text", "image"}, ContextWindow: 1048576, MaxTokens: 16384},
	}},
	{ID: "google-vertex", Name: "Google Vertex AI", BaseURL: "https://{LOCATION}-aiplatform.googleapis.com/v1beta1", API: "google-generative-ai", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"GOOGLE_VERTEX_API_KEY", "GOOGLE_API_KEY"}}, AuthEnvVars: []string{"GOOGLE_VERTEX_API_KEY", "GOOGLE_API_KEY"}, Models: []ModelConfig{
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Input: []string{"text", "image"}, ContextWindow: 1048576, MaxTokens: 8192},
	}},
	{ID: "mistral", Name: "Mistral", BaseURL: "https://api.mistral.ai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"MISTRAL_API_KEY"}}, AuthEnvVars: []string{"MISTRAL_API_KEY"}, Models: []ModelConfig{
		{ID: "mistral-large-latest", Name: "Mistral Large", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 16384},
	}},
	{ID: "groq", Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"GROQ_API_KEY"}}, AuthEnvVars: []string{"GROQ_API_KEY"}, Models: []ModelConfig{
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "together", Name: "Together AI", BaseURL: "https://api.together.xyz/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"TOGETHER_API_KEY"}}, AuthEnvVars: []string{"TOGETHER_API_KEY"}, Models: []ModelConfig{
		{ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "fireworks", Name: "Fireworks AI", BaseURL: "https://api.fireworks.ai/inference/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"FIREWORKS_API_KEY"}}, AuthEnvVars: []string{"FIREWORKS_API_KEY"}, Models: []ModelConfig{
		{ID: "accounts/fireworks/models/llama-v3p3-70b-instruct", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "cerebras", Name: "Cerebras", BaseURL: "https://api.cerebras.ai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"CEREBRAS_API_KEY"}}, AuthEnvVars: []string{"CEREBRAS_API_KEY"}, Models: []ModelConfig{
		{ID: "llama3.1-8b", Name: "Llama 3.1 8B", Input: []string{"text"}, ContextWindow: 8192, MaxTokens: 4096},
	}},
	{ID: "xai", Name: "xAI", BaseURL: "https://api.x.ai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"XAI_API_KEY"}}, AuthEnvVars: []string{"XAI_API_KEY"}, Models: []ModelConfig{
		{ID: "grok-2-1212", Name: "Grok 2", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 4096},
	}},
	{ID: "huggingface", Name: "Hugging Face", BaseURL: "https://api-inference.huggingface.co/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"HF_TOKEN"}}, AuthEnvVars: []string{"HF_TOKEN"}, Models: []ModelConfig{
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"OPENROUTER_API_KEY"}}, AuthEnvVars: []string{"OPENROUTER_API_KEY"}, Models: []ModelConfig{
		{ID: "openai/gpt-4o", Name: "GPT-4o", Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
	}},
	{ID: "nvidia", Name: "NVIDIA NIM", BaseURL: "https://integrate.api.nvidia.com/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"NVIDIA_API_KEY"}}, AuthEnvVars: []string{"NVIDIA_API_KEY"}, Models: []ModelConfig{
		{ID: "meta/llama-3.3-70b-instruct", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "github-copilot", Name: "GitHub Copilot", BaseURL: "https://api.githubcopilot.com/v1", API: "openai-completions", Auth: AuthConfig{Type: "oauth", EnvVars: []string{"GITHUB_COPILOT_TOKEN"}}, AuthEnvVars: []string{"GITHUB_COPILOT_TOKEN"}, Models: []ModelConfig{
		{ID: "claude-sonnet-4.6", Name: "Claude Sonnet 4.6", Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 16384},
	}},
	{ID: "cloudflare", Name: "Cloudflare Workers AI", BaseURL: "https://api.cloudflare.com/client/v4/accounts/{ACCOUNT_ID}/ai/v1", API: "openai-completions", Auth: AuthConfig{Type: "api_key", EnvVars: []string{"CLOUDFLARE_API_KEY"}}, AuthEnvVars: []string{"CLOUDFLARE_API_KEY"}, Models: []ModelConfig{
		{ID: "@cf/meta/llama-3.3-70b-instruct", Name: "Llama 3.3 70B", Input: []string{"text"}, ContextWindow: 131072, MaxTokens: 8192},
	}},
	{ID: "faux", Name: "Faux (testing)", BaseURL: "", API: "faux", Auth: AuthConfig{Type: "api_key", EnvVars: []string{}}, AuthEnvVars: []string{}, Models: []ModelConfig{
		{ID: "faux-1", Name: "Faux Model", Input: []string{"text", "image"}, ContextWindow: 128000, MaxTokens: 16384},
	}},
}

func RegisterBuiltins(r *Registry) {
	for _, p := range builtinProviders {
		r.Register(p)
	}
}
