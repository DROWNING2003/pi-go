// Package api provides provider-specific API utilities matching TS api/ files.
package api

import (
	"encoding/json"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// Copilot headers (github-copilot-headers.ts)

// InferCopilotInitiator returns the X-Initiator header value.
func InferCopilotInitiator(messages []json.RawMessage) string {
	if len(messages) == 0 {
		return "user"
	}
	var last struct {
		Role string `json:"role"`
	}
	json.Unmarshal(messages[len(messages)-1], &last)
	if last.Role != "user" {
		return "agent"
	}
	return "user"
}

// HasCopilotVisionInput checks if any message contains images.
func HasCopilotVisionInput(messages []json.RawMessage) bool {
	for _, raw := range messages {
		var msg struct {
			Role    string            `json:"role"`
			Content []json.RawMessage `json:"content"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		if msg.Role != "user" && msg.Role != "toolResult" {
			continue
		}
		for _, c := range msg.Content {
			var block struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(c, &block) == nil && block.Type == model.ContentTypeImage {
				return true
			}
		}
	}
	return false
}

// BuildCopilotDynamicHeaders builds Copilot-specific request headers.
func BuildCopilotDynamicHeaders(messages []json.RawMessage, hasImages bool) map[string]string {
	headers := map[string]string{
		"X-Initiator":   InferCopilotInitiator(messages),
		"Openai-Intent": "conversation-edits",
	}
	if hasImages {
		headers["Copilot-Vision-Request"] = "true"
	}
	return headers
}

// Cloudflare base URLs (cloudflare.ts)
const (
	CloudflareWorkersAIBaseURL          = "https://api.cloudflare.com/client/v4/accounts/{CF_ACCOUNT_ID}/ai/v1"
	CloudflareAIGatewayCompatBaseURL    = "https://gateway.ai.cloudflare.com/v1/{CF_ACCOUNT_ID}/{CF_GATEWAY_ID}/compat"
	CloudflareAIGatewayOpenAIBaseURL    = "https://gateway.ai.cloudflare.com/v1/{CF_ACCOUNT_ID}/{CF_GATEWAY_ID}/openai"
	CloudflareAIGatewayAnthropicBaseURL = "https://gateway.ai.cloudflare.com/v1/{CF_ACCOUNT_ID}/{CF_GATEWAY_ID}/anthropic"
)

// LazyStream creates a stream that runs async setup before forwarding. (lazy.ts equivalent)
// In Go this is simplified - we run setup synchronously in the goroutine.
func LazyStream(setup func() (<-chan model.StreamEvent, error)) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)
	go func() {
		defer close(ch)
		inner, err := setup()
		if err != nil {
			ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{
				ErrorMessage: err.Error(),
			})
			return
		}
		for evt := range inner {
			ch <- evt
		}
	}()
	return ch
}
