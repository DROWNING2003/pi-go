// Package modelruntime provides a runtime wrapper around the AI model.
package modelruntime

import (
	"context"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/credentials"
)

// Runtime wraps a provider model with credential resolution and streaming.
type Runtime struct {
	Model    *provider.ProviderModel
	Provider *provider.ProviderConfig
	Registry *provider.Registry
}

// New creates a model runtime from a model reference.
func New(ref string, reg *provider.Registry) (*Runtime, error) {
	m := reg.ResolveModel(ref)
	if m == nil {
		return nil, &ModelError{Ref: ref, Message: "model not found"}
	}
	prov := reg.GetProvider(m.Provider)
	if prov == nil {
		return nil, &ModelError{Ref: ref, Message: "provider not found"}
	}
	return &Runtime{Model: m, Provider: prov, Registry: reg}, nil
}

// Stream creates a stream for the model.
func (r *Runtime) Stream(ctx context.Context, c *provider.Context, opts *provider.StreamOptions) (<-chan model.StreamEvent, error) {
	apiKey, headers := credentials.Resolve(r.Provider)
	if apiKey == "" && r.Provider.ID != "faux" {
		return nil, &ModelError{Ref: r.Model.ID, Message: "no API key configured"}
	}

	client := protocol.NewHTTPClient(r.Provider.BaseURL, headers)

	switch r.Provider.API {
	case "openai-completions":
		return protocol.StreamChatCompletion(ctx, client, r.Model, c, opts), nil
	case "openai-responses":
		return protocol.StreamOpenAIResponses(ctx, client, r.Model, c, opts), nil
	case "anthropic-messages":
		return protocol.StreamAnthropicMessages(ctx, client, r.Model, c, opts), nil
	case "google-generative-ai":
		return protocol.StreamGoogleGenerate(ctx, client, r.Model, c, opts), nil
	case "bedrock-converse-stream":
		return protocol.StreamBedrockConverse(ctx, client, r.Model, c, opts), nil
	default:
		return nil, &ModelError{Ref: r.Provider.API, Message: "unsupported API"}
	}
}

// ModelError represents a model resolution or runtime error.
type ModelError struct {
	Ref     string
	Message string
}

func (e *ModelError) Error() string {
	return "model error: " + e.Message + " (" + e.Ref + ")"
}
