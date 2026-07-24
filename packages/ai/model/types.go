package model

import "encoding/json"

// --- Thinking levels ---

// ThinkingLevel maps to the TS ThinkingLevel type.
type ThinkingLevel string

const (
	ThinkingOff     ThinkingLevel = "off"
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
	ThinkingXHigh   ThinkingLevel = "xhigh"
	ThinkingMax     ThinkingLevel = "max"
)

// ThinkingLevelMap maps pi thinking levels to provider-specific values.
type ThinkingLevelMap map[ThinkingLevel]string

// ThinkingBudgets holds token budgets per thinking level for token-based providers.
type ThinkingBudgets struct {
	Minimal *int `json:"minimal,omitempty"`
	Low     *int `json:"low,omitempty"`
	Medium  *int `json:"medium,omitempty"`
	High    *int `json:"high,omitempty"`
}

// --- Cache & transport ---

// CacheRetention controls prompt cache retention policy.
type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

// Transport selects the protocol transport.
type Transport string

const (
	TransportSSE             Transport = "sse"
	TransportWebSocket       Transport = "websocket"
	TransportWebSocketCached Transport = "websocket-cached"
	TransportAuto            Transport = "auto"
)

// --- Model (unified) ---

// ModelCost holds pricing with optional tiers.
type ModelCost struct {
	Input      float64         `json:"input"`
	Output     float64         `json:"output"`
	CacheRead  float64         `json:"cacheRead"`
	CacheWrite float64         `json:"cacheWrite"`
	Tiers      []ModelCostTier `json:"tiers,omitempty"`
}

// ModelCostTier is a volume-based pricing tier.
type ModelCostTier struct {
	Input            float64 `json:"input"`
	Output           float64 `json:"output"`
	CacheRead        float64 `json:"cacheRead"`
	CacheWrite       float64 `json:"cacheWrite"`
	InputTokensAbove int     `json:"inputTokensAbove"`
}

// UnifiedModel describes a concrete model with all TS-equivalent fields.
type UnifiedModel struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	API              string            `json:"api"`
	Provider         string            `json:"provider"`
	BaseURL          string            `json:"baseUrl"`
	Reasoning        bool              `json:"reasoning"`
	ThinkingLevelMap ThinkingLevelMap  `json:"thinkingLevelMap,omitempty"`
	Input            []string          `json:"input"`
	Cost             ModelCost         `json:"cost"`
	ContextWindow    int64             `json:"contextWindow"`
	MaxTokens        int64             `json:"maxTokens"`
	Headers          map[string]string `json:"headers,omitempty"`
}

// --- Tool (public contract) ---

// ToolDef describes a tool available to the model with a JSON Schema.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// --- Context ---

// Context is the full conversation context sent to a provider.
type Context struct {
	SystemPrompt string            `json:"systemPrompt,omitempty"`
	Messages     []json.RawMessage `json:"messages"`
	Tools        []ToolDef         `json:"tools,omitempty"`
}

// --- Stream options ---

// StreamOptions carries all optional parameters for a provider stream request.
type StreamOptions struct {
	Temperature    float64
	MaxTokens      int
	APIKey         string
	Transport      Transport
	CacheRetention CacheRetention
	SessionID      string
	Signal         interface{} // context.Context in Go
	// TODO: onPayload, onResponse callbacks
}

// Ensure json is used.
var _ = json.RawMessage{}

// --- Provider headers & env ---

type ProviderHeaders map[string]string
type ProviderEnv map[string]string
type SessionAffinityFormat string

const (
	SessionAffinityOpenAI          SessionAffinityFormat = "openai"
	SessionAffinityOpenAINoSession SessionAffinityFormat = "openai-nosession"
	SessionAffinityOpenRouter      SessionAffinityFormat = "openrouter"
)

// --- Text signature (OpenAI responses) ---

type TextSignatureV1 struct {
	V     int    `json:"v"`
	ID    string `json:"id"`
	Phase string `json:"phase,omitempty"`
}

// --- Chat template kwargs ---

type ChatTemplateKwargValue struct {
	Var         string `json:"$var,omitempty"`
	OmitWhenOff bool   `json:"omitWhenOff,omitempty"`
	Value       string `json:"value,omitempty"`
}

// --- Simple stream options ---

type SimpleStreamOptions struct {
	Temperature     float64
	MaxTokens       int
	Reasoning       ThinkingLevel
	ThinkingBudgets *ThinkingBudgets
	APIKey          string
	Transport       Transport
	CacheRetention  CacheRetention
	SessionID       string
	Headers         ProviderHeaders
	Env             ProviderEnv
}

// --- Image generation types ---

type ImagesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type ImagesOutputContent = ImagesInputContent

type ImagesStopReason string

const (
	ImagesStopStop    ImagesStopReason = "stop"
	ImagesStopError   ImagesStopReason = "error"
	ImagesStopAborted ImagesStopReason = "aborted"
)

type ImagesContext struct {
	Input []ImagesInputContent `json:"input"`
}

type ImagesOptions struct {
	APIKey          string
	Signal          interface{}
	Env             ProviderEnv
	Headers         ProviderHeaders
	TimeoutMs       int
	MaxRetries      int
	MaxRetryDelayMs int
}

type AssistantImages struct {
	API          string              `json:"api"`
	Provider     string              `json:"provider"`
	Model        string              `json:"model"`
	Output       []ImagesOutputContent `json:"output"`
	ResponseID   string              `json:"responseId,omitempty"`
	Usage        *Usage              `json:"usage,omitempty"`
	StopReason   ImagesStopReason    `json:"stopReason"`
	ErrorMessage string              `json:"errorMessage,omitempty"`
	Timestamp    int64               `json:"timestamp"`
}

type ImagesModel struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	API      string   `json:"api"`
	Provider string   `json:"provider"`
	BaseURL  string   `json:"baseUrl"`
	Output   []string `json:"output"`
	Cost     ModelCost `json:"cost"`
}

// --- TextHelpers ---

// ContentText extracts all text from content blocks.
func ContentText(blocks []ContentBlock) string {
	var result string
	for _, b := range blocks {
		if b.Type == ContentTypeText {
			result += b.Text
		}
	}
	return result
}

// --- API & Provider identifiers ---

type KnownAPI string
type KnownProvider string

const (
	APIOpenAICompletions     KnownAPI = "openai-completions"
	APIMistralConversations  KnownAPI = "mistral-conversations"
	APIOpenAIResponses       KnownAPI = "openai-responses"
	APIAzureOpenAIResponses  KnownAPI = "azure-openai-responses"
	APIOpenAICodexResponses  KnownAPI = "openai-codex-responses"
	APIAnthropicMessages     KnownAPI = "anthropic-messages"
	APIBedrockConverse       KnownAPI = "bedrock-converse-stream"
	APIGoogleGenerative      KnownAPI = "google-generative-ai"
	APIGoogleVertex          KnownAPI = "google-vertex"
	APIPiMessages            KnownAPI = "pi-messages"
)

const (
	ProviderAmazonBedrock     KnownProvider = "amazon-bedrock"
	ProviderAnthropic         KnownProvider = "anthropic"
	ProviderGoogle            KnownProvider = "google"
	ProviderGoogleVertex      KnownProvider = "google-vertex"
	ProviderOpenAI            KnownProvider = "openai"
	ProviderAzureOpenAI       KnownProvider = "azure-openai-responses"
	ProviderOpenAICodex       KnownProvider = "openai-codex"
	ProviderDeepSeek          KnownProvider = "deepseek"
	ProviderXAI               KnownProvider = "xai"
	ProviderGroq              KnownProvider = "groq"
	ProviderCerebras          KnownProvider = "cerebras"
	ProviderOpenRouter        KnownProvider = "openrouter"
	ProviderMistral           KnownProvider = "mistral"
	ProviderTogether          KnownProvider = "together"
	ProviderFireworks         KnownProvider = "fireworks"
	ProviderHuggingFace       KnownProvider = "huggingface"
)
