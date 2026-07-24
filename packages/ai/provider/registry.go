package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// StreamDispatcher is a function that streams a model response.
// It is registered per API type to avoid circular imports with the protocol package.
type StreamDispatcher func(ctx context.Context, m *ProviderModel, c *Context, opts *StreamOptions) <-chan model.StreamEvent

// Registry holds all registered providers and their models.
type Registry struct {
	mu          sync.RWMutex
	providers   map[string]*ProviderConfig
	store       *CredentialStore
	dispatchers map[string]StreamDispatcher
}

// NewRegistry creates a provider registry with an optional credential store.
func NewRegistry(store *CredentialStore) *Registry {
	return &Registry{
		providers:   make(map[string]*ProviderConfig),
		store:       store,
		dispatchers: make(map[string]StreamDispatcher),
	}
}

// SetDispatcher registers a stream dispatcher for an API type.
func (r *Registry) SetDispatcher(api string, d StreamDispatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dispatchers[api] = d
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
				ContextWindow: int64(mc.ContextWindow),
				MaxTokens:     int64(mc.MaxTokens),
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
					ContextWindow: int64(mc.ContextWindow),
					MaxTokens:     int64(mc.MaxTokens),
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

// Stream dispatches a model request using the registered dispatcher for the model's API.
func (r *Registry) Stream(ctx context.Context, modelRef string, c *Context, opts *StreamOptions) (<-chan model.StreamEvent, error) {
	m := r.ResolveModel(modelRef)
	if m == nil {
		return nil, fmt.Errorf("model not found: %s", modelRef)
	}

	prov := r.GetProvider(m.Provider)
	if prov == nil {
		return nil, fmt.Errorf("provider not found: %s", m.Provider)
	}

	r.mu.RLock()
	dispatch := r.dispatchers[prov.API]
	r.mu.RUnlock()

	if dispatch == nil {
		return nil, fmt.Errorf("no dispatcher for API: %s", prov.API)
	}

	return dispatch(ctx, m, c, opts), nil
}
