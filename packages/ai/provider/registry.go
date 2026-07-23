package provider

import (
	"strings"
	"sync"
)

// ProviderConfig describes a registered AI provider.
type ProviderConfig struct {
	ID          string
	Name        string
	BaseURL     string
	API         string // "openai-completions", "anthropic-messages", etc.
	AuthEnvVars []string
	Models      []ModelConfig
}

// ModelConfig is a provider-specific model definition.
type ModelConfig struct {
	ID            string
	Name          string
	Reasoning     bool
	Input         []string
	ContextWindow int
	MaxTokens     int
	Cost          ModelCost
}

// Registry holds all registered providers and their models.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]*ProviderConfig
	store     *CredentialStore
}

// NewRegistry creates a provider registry with an optional credential store.
func NewRegistry(store *CredentialStore) *Registry {
	r := &Registry{
		providers: make(map[string]*ProviderConfig),
		store:     store,
	}
	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p *ProviderConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID] = p
}

// GetProvider returns a provider by ID.
func (r *Registry) GetProvider(id string) *ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[id]
}

// ListProviders returns all registered provider IDs.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

// GetModel resolves a model by provider and model ID.
func (r *Registry) GetModel(providerID, modelID string) *ProviderModel {
	prov := r.GetProvider(providerID)
	if prov == nil {
		return nil
	}
	for _, mc := range prov.Models {
		if mc.ID == modelID {
			return &ProviderModel{
				ID:            mc.ID,
				Name:          mc.Name,
				API:           prov.API,
				Provider:      prov.ID,
				BaseURL:       prov.BaseURL,
				Reasoning:     mc.Reasoning,
				Input:         mc.Input,
				ContextWindow: mc.ContextWindow,
				MaxTokens:     mc.MaxTokens,
				Cost:          mc.Cost,
			}
		}
	}
	return nil
}

// ResolveModel finds a model by "<provider>/<model>" or just "<model>" (first match).
func (r *Registry) ResolveModel(ref string) *ProviderModel {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		return r.GetModel(parts[0], parts[1])
	}
	// Search all providers
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, prov := range r.providers {
		for _, mc := range prov.Models {
			if mc.ID == parts[0] {
				return &ProviderModel{
					ID:            mc.ID,
					Name:          mc.Name,
					API:           prov.API,
					Provider:      prov.ID,
					BaseURL:       prov.BaseURL,
					Reasoning:     mc.Reasoning,
					Input:         mc.Input,
					ContextWindow: mc.ContextWindow,
					MaxTokens:     mc.MaxTokens,
					Cost:          mc.Cost,
				}
			}
		}
	}
	return nil
}

// ResolveAPIKeyForProvider resolves the API key for a registered provider.
func (r *Registry) ResolveAPIKeyForProvider(providerID string, optsKey string) string {
	prov := r.GetProvider(providerID)
	if prov == nil {
		return ""
	}
	return ResolveAPIKey(r.store, providerID, prov.AuthEnvVars, optsKey)
}

// RegisterBuiltins registers all built-in providers.
func RegisterBuiltins(r *Registry) {
	for _, p := range builtinProviders {
		r.Register(p)
	}
}
