// hello-pi-ai demonstrates using the pi-ai library standalone:
//   - Provider registry with built-in models
//   - Credential resolution from environment variables
//   - Streaming chat via OpenAI-compatible API
//   - Building and inspecting messages
//
// Usage:
//
//	export DEEPSEEK_API_KEY=sk-xxx
//	go run .
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ai "github.com/DROWNING2003/pi-go/packages/ai"
)

func main() {
	fmt.Println("=== pi-ai standalone example ===\n")

	// 1. Create provider registry
	reg := ai.NewRegistry(nil)
	ai.RegisterBuiltins(reg)

	fmt.Println("Registered providers:", strings.Join(reg.ListProviders(), ", "))

	// 2. Resolve a model
	m := reg.ResolveModel("deepseek/deepseek-chat")
	if m == nil {
		fmt.Println("Model not found, trying faux...")
		m = reg.ResolveModel("faux/faux-1")
	}
	fmt.Printf("Using model: %s/%s (reasoning=%v, context=%d)\n\n",
		m.Provider, m.ID, m.Reasoning, m.ContextWindow)

	// 3. Resolve API key
	key := reg.ResolveAPIKeyForProvider(m.Provider, "")
	useFaux := key == ""

	if useFaux {
		fmt.Println("No API key found, using faux provider for demo...")
		demoWithFaux(reg, m)
	} else {
		fmt.Println("API key found, streaming from real provider...")
		demoWithReal(reg, m, key)
	}
}

func demoWithReal(reg *ai.Registry, m *ai.UnifiedModel, key string) {
	prov := reg.GetProvider(m.Provider)
	headers := map[string]string{"Authorization": "Bearer " + key}
	client := ai.NewHTTPClient(prov.BaseURL, headers)

	ctx := ai.Context{
		SystemPrompt: "Be concise. Answer in one sentence.",
		Messages: []json.RawMessage{
			mustMarshal(ai.UserMessage{
				Role:    "user",
				Content: ai.UserContent{ai.NewTextContent("What is Go?")},
			}),
		},
	}

	for evt := range ai.StreamChatCompletion(context.Background(), client, m, &ctx, nil) {
		switch evt.Type {
		case ai.StreamEventTextDelta:
			fmt.Print(evt.Delta)
		case ai.StreamEventDone:
			fmt.Printf("\n\n---\nUsage: %d tokens, stop=%s\n",
				evt.Message.Usage.TotalTokens, evt.Message.StopReason)
		case ai.StreamEventError:
			fmt.Fprintf(os.Stderr, "Error: %s\n", evt.Error.ErrorMessage)
		}
	}
}

func demoWithFaux(reg *ai.Registry, m *ai.UnifiedModel) {
	faux := ai.NewFauxProvider()
	faux.SetResponses(
		ai.FauxMessage{
			Message: &ai.AssistantMessage{
				Role: "assistant",
				Content: []ai.ContentBlock{
					ai.NewTextContent("Go is a statically typed, compiled language designed for simplicity and concurrency."),
				},
				API: m.API, Provider: m.Provider, Model: m.ID,
				StopReason: ai.StopReasonStop,
				Usage:      ai.Usage{Input: 5, Output: 12, TotalTokens: 17},
			},
		},
	)

	ctx := ai.Context{
		SystemPrompt: "Be concise.",
		Messages: []json.RawMessage{
			mustMarshal(ai.UserMessage{
				Role:    "user",
				Content: ai.UserContent{ai.NewTextContent("What is Go?")},
			}),
		},
	}

	for evt := range faux.Stream(context.Background(), m, &ctx, nil) {
		switch evt.Type {
		case ai.StreamEventTextDelta:
			fmt.Print(evt.Delta)
		case ai.StreamEventDone:
			fmt.Printf("\n\n---\nUsage: %d tokens, stop=%s\n",
				evt.Message.Usage.TotalTokens, evt.Message.StopReason)
		}
	}

	// Also demo message construction
	fmt.Println("\n=== Message construction demo ===")
	msg := ai.AssistantMessage{
		Role: "assistant",
		Content: []ai.ContentBlock{
			ai.NewThinkingContent("Let me think about this..."),
			ai.NewTextContent("Here is the answer."),
			ai.NewToolCallContent("call-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
		},
		API: "faux", Provider: "faux", Model: "faux-1",
		StopReason: ai.StopReasonToolUse,
	}
	data, _ := json.MarshalIndent(msg, "", "  ")
	fmt.Println(string(data))
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
