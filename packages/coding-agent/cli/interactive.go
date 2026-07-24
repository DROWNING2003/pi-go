package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// runInteractiveShell starts a readline-based interactive shell.
func runInteractiveShell(stdout, stderr io.Writer, m *provider.ProviderModel, prov *provider.ProviderConfig, client *protocol.HTTPClient, tools *tool.Registry, cwd string, reg *provider.Registry) int {
	fmt.Fprintf(stderr, "pi ● %s/%s\n/help /quit /clear /model /list /save\n\n", m.Provider, m.ID)
	scanner := bufio.NewScanner(os.Stdin)
	var history []json.RawMessage

	for {
		fmt.Fprint(stderr, "▸ ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch {
		case input == "/quit" || input == "/exit":
			return 0
		case input == "/help":
			fmt.Fprintln(stderr, "  /quit, /exit  - quit")
			fmt.Fprintln(stderr, "  /clear        - new session")
			fmt.Fprintln(stderr, "  /model <ref>  - switch model (e.g. openai/gpt-4o)")
			fmt.Fprintln(stderr, "  /list         - list available models")
			fmt.Fprintln(stderr, "  /save         - save session to disk")
			continue
		case input == "/clear":
			history = nil
			fmt.Fprintln(stderr, "  ✨ session cleared")
			continue
		case strings.HasPrefix(input, "/model "):
			ref := strings.TrimPrefix(input, "/model ")
			if mm := reg.ResolveModel(ref); mm != nil {
				m = mm
				prov = reg.GetProvider(mm.Provider)
				fmt.Fprintf(stderr, "  ✓ switched to %s/%s\n", mm.Provider, mm.ID)
			} else {
				fmt.Fprintf(stderr, "  ✗ model not found: %s\n", ref)
			}
			continue
		case input == "/list":
			for _, pid := range reg.ListProviders() {
				p := reg.GetProvider(pid)
				if p == nil {
					continue
				}
				for _, mc := range p.Models {
					fmt.Fprintf(stderr, "  %s/%s\n", pid, mc.ID)
				}
			}
			continue
		case input == "/save":
			if len(history) > 0 {
				saveSessionRaw(history, cwd)
				fmt.Fprintln(stderr, "  ✓ session saved")
			} else {
				fmt.Fprintln(stderr, "  nothing to save")
			}
			continue
		}

		// Build user message and add to history
		userData, _ := json.Marshal(model.UserMessage{
			Role: "user", Content: model.UserContent{model.NewTextContent(input)},
			Timestamp: time.Now().UnixMilli(),
		})
		history = append(history, userData)

		// Build stream function
		streamFn := func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
			switch prov.API {
			case "openai-completions":
				return protocol.StreamChatCompletion(ctx, client, pm, c, so)
			case "openai-responses":
				return protocol.StreamOpenAIResponses(ctx, client, pm, c, so)
			case "anthropic-messages":
				return protocol.StreamAnthropicMessages(ctx, client, pm, c, so)
			case "google-generative-ai":
				return protocol.StreamGoogleGenerate(ctx, client, pm, c, so)
			case "bedrock-converse-stream":
				return protocol.StreamBedrockConverse(ctx, client, pm, c, so)
			default:
				ch := make(chan model.StreamEvent, 1)
				ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported: " + prov.API})
				close(ch)
				return ch
			}
		}

		config := &loop.Config{
			Model:    m,
			Tools:    tools,
			MaxTurns: 10,
			StreamFn: streamFn,
		}

		userMsg := &model.UserMessage{
			Role: "user", Content: model.UserContent{model.NewTextContent(input)},
			Timestamp: time.Now().UnixMilli(),
		}

		ctx := context.Background()
		msgs, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
		if err != nil {
			fmt.Fprintf(stderr, "  ✗ %v\n", err)
			continue
		}

		for _, msg := range msgs {
			data, _ := json.Marshal(msg)
			history = append(history, data)

			if msg.Assistant != nil {
				for _, block := range msg.Assistant.Content {
					switch block.Type {
					case model.ContentTypeText:
						fmt.Fprint(stdout, block.Text)
					case model.ContentTypeToolCall:
						fmt.Fprintf(stderr, "  🔧 %s %s\n", block.Name, string(block.Arguments))
					}
				}
			}
			if msg.ToolResult != nil {
				text := ""
				for _, b := range msg.ToolResult.Content {
					if b.Type == model.ContentTypeText {
						text += b.Text
					}
				}
				if len(text) > 200 {
					text = text[:200] + "..."
				}
				fmt.Fprintf(stderr, "  [%s] %s\n", msg.ToolResult.ToolName, text)
			}
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func saveSessionRaw(messages []json.RawMessage, cwd string) {
	d, _ := os.UserConfigDir()
	d = filepath.Join(d, "pi-go", "sessions")
	os.MkdirAll(d, 0700)
	id := fmt.Sprintf("chat-%d", time.Now().Unix())
	path := filepath.Join(d, id+".jsonl")

	// Write header
	header := fmt.Sprintf(`{"type":"session","version":3,"id":"%s","timestamp":"%s","cwd":"%s"}`, id, time.Now().UTC().Format(time.RFC3339), cwd)
	lines := []string{header}
	for _, m := range messages {
		lines = append(lines, string(m))
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
