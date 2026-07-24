// Package providercomposer composes providers from config and extensions.
package providercomposer

import (
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/credentials"
)

// ComposedProvider wraps a provider with resolved credentials and headers.
type ComposedProvider struct {
	Provider *provider.ProviderConfig
	Model    *provider.ProviderModel
	APIKey   string
	Headers  map[string]string
}

// Compose resolves a model from a reference and builds credential headers.
func Compose(ref string, reg *provider.Registry) (*ComposedProvider, error) {
	m := reg.ResolveModel(ref)
	if m == nil {
		return nil, &Error{Ref: ref, Message: "model not found"}
	}

	prov := reg.GetProvider(m.Provider)
	if prov == nil {
		return nil, &Error{Ref: ref, Message: "provider not found"}
	}

	apiKey, headers := credentials.Resolve(prov)

	return &ComposedProvider{
		Provider: prov,
		Model:    m,
		APIKey:   apiKey,
		Headers:  headers,
	}, nil
}

// Validate checks that the provider has credentials configured.
func (c *ComposedProvider) Validate() error {
	if c.APIKey == "" && c.Provider.ID != "faux" {
		return &Error{Ref: c.Provider.ID, Message: "no API key configured"}
	}
	return nil
}

// Error is a provider composition error.
type Error struct {
	Ref     string
	Message string
}

func (e *Error) Error() string {
	return e.Message + " (" + e.Ref + ")"
}
