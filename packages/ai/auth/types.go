// Package auth provides auth type definitions matching TS auth/types.ts
package auth

import "github.com/DROWNING2003/pi-go/packages/ai/model"

// ModelAuth is per-request auth for a single model call.
type ModelAuth struct {
	APIKey  string
	Headers model.ProviderHeaders
	BaseURL string
}

// ApiKeyCredential is a stored API key credential.
type ApiKeyCredential struct {
	Type string            `json:"type"`
	Key  string            `json:"key,omitempty"`
	Env  model.ProviderEnv `json:"env,omitempty"`
}

// OAuthCredentials is OAuth token data.
type OAuthCredentials struct {
	Refresh string                 `json:"refresh"`
	Access  string                 `json:"access"`
	Expires int64                  `json:"expires"`
	Extra   map[string]interface{} `json:"-"`
}

// OAuthCredential is a stored OAuth credential.
type OAuthCredential struct {
	OAuthCredentials
	Type string `json:"type"`
}

// Credential is a type-tagged credential.
type Credential struct {
	Type string `json:"type"` // "api_key" or "oauth"
	Key  string `json:"key,omitempty"`
	// OAuth fields
	Refresh string            `json:"refresh,omitempty"`
	Access  string            `json:"access,omitempty"`
	Expires int64             `json:"expires,omitempty"`
	Env     model.ProviderEnv `json:"env,omitempty"`
}

// CredentialInfo is non-secret credential metadata.
type CredentialInfo struct {
	ProviderID string `json:"providerId"`
	Type       string `json:"type"`
}

// AuthType represents authentication method.
type AuthType string

const (
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOAuth  AuthType = "oauth"
)

// AuthCheck is the result of an auth availability check.
type AuthCheck struct {
	Available bool
	Provider  string
	Type      AuthType
	Label     string
}

// AuthResult is the result of resolving auth for a provider.
type AuthResult struct {
	APIKey  string
	Headers model.ProviderHeaders
	Source  string // "stored credential", "env", "options"
}
