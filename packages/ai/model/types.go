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
