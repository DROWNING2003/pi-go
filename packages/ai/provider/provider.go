// Package provider defines the Provider interface, Model catalog, and
// authentication contracts for AI service providers.
//
// Providers encapsulate API-specific streaming logic and expose a uniform
// interface consumed by the Agent loop. The faux (scriptable in-memory)
// provider serves as the primary driver for all Agent tests.
package provider

import (
	"context"
	"encoding/json"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// ProviderModel describes a concrete model available from a provider.
type ProviderModel struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	API           string    `json:"api"`
	Provider      string    `json:"provider"`
	BaseURL       string    `json:"baseUrl"`
	Reasoning     bool      `json:"reasoning"`
	Input         []string  `json:"input"`
	ContextWindow int       `json:"contextWindow"`
	MaxTokens     int       `json:"maxTokens"`
	Cost          ModelCost `json:"cost"`
}

// ModelCost captures per-million-token pricing.
type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// Context is the conversation context sent with each provider request.
type Context struct {
	SystemPrompt string            `json:"systemPrompt,omitempty"`
	Messages     []json.RawMessage `json:"messages"`
	Tools        []ToolDef         `json:"tools,omitempty"`
}

// ToolDef describes a tool available to the model.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// StreamOptions carries optional parameters for a provider stream request.
type StreamOptions struct {
	Temperature    float64
	MaxTokens      int
	APIKey         string
	Transport      string
	CacheRetention string
	SessionID      string
}

// Provider is the uniform interface for AI service providers.
type Provider interface {
	ID() string
	Stream(ctx context.Context, m *ProviderModel, c *Context, opts *StreamOptions) <-chan model.StreamEvent
}
