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

// Re-exports from model for backward compatibility.
type (
	Context       = model.Context
	ToolDef       = model.ToolDef
	StreamOptions = model.StreamOptions
	UnifiedModel  = model.UnifiedModel
	ModelCost     = model.ModelCost
)

// ProviderModel is a concrete model available from a provider (backward compat).
type ProviderModel = model.UnifiedModel

// Provider is the uniform interface for AI service providers.
type Provider interface {
	ID() string
	Stream(ctx context.Context, m *ProviderModel, c *Context, opts *StreamOptions) <-chan model.StreamEvent
}

// ModelConfig is a provider-specific model definition for registration.
type ModelConfig struct {
	ID            string
	Name          string
	Reasoning     bool
	Input         []string
	ContextWindow int
	MaxTokens     int
	Cost          ModelCost
}

// AuthConfig describes how a provider authenticates.
type AuthConfig struct {
	Type    string   // "api_key" or "oauth"
	Name    string   // human-readable name
	EnvVars []string // environment variable names
}

// ProviderConfig describes a registered AI provider.
type ProviderConfig struct {
	ID          string
	Name        string
	BaseURL     string
	API         string
	Auth        AuthConfig
	AuthEnvVars []string // deprecated, use Auth.EnvVars
	Models      []ModelConfig
}

var _ = json.RawMessage{}
