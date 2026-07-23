package rpc

import (
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
	"github.com/DROWNING2003/pi-go/packages/coding-agent/jsonl"
)

// Command types.
const (
	CmdPrompt   = "prompt"
	CmdAbort    = "abort"
	CmdQuit     = "quit"
	CmdExit     = "exit"
	CmdGetState = "get_state"
	CmdSetModel = "set_model"
)

type Command struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Message  string `json:"message,omitempty"`
	Provider string `json:"provider,omitempty"`
	ModelID  string `json:"modelId,omitempty"`
}

type Response struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RunRPC starts the RPC loop.
func RunRPC(reg *provider.Registry, m *provider.ProviderModel) error {
	cwd, _ := os.Getwd()
	configDir, _ := os.UserConfigDir()
	configDir = filepath.Join(configDir, "pi-go")

	// Setup tools
	tools := tool.NewRegistry()
	tools.Register(tool.NewReadTool(cwd))
	tools.Register(tool.NewWriteTool(cwd))
	tools.Register(tool.NewEditTool(cwd))
	tools.Register(tool.NewBashTool(cwd))

	h := &Handler{
		reader:    jsonl.NewReader(os.Stdin),
		writer:    jsonl.NewWriter(os.Stdout),
		reg:       reg,
		model:     m,
		cwd:       cwd,
		configDir: configDir,
		tools:     tools,
	}
	return h.loop()
}

type Handler struct {
	reader    *jsonl.Reader
	writer    *jsonl.Writer
	reg       *provider.Registry
	model     *provider.ProviderModel
	cwd       string
	configDir string
	tools     *tool.Registry
	aborted   bool
}

func (h *Handler) loop() error {
	for {
		var raw json.RawMessage
		if err := h.reader.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var cmd Command
		if err := json.Unmarshal(raw, &cmd); err != nil {
			h.error("", "", fmt.Sprintf("invalid: %v", err))
			continue
		}

		switch cmd.Type {
		case CmdQuit, CmdExit:
			h.success(cmd.ID, cmd.Type, nil)
			return nil
		case CmdPrompt:
			h.handlePrompt(cmd)
		case CmdAbort:
			h.aborted = true
			h.success(cmd.ID, cmd.Type, nil)
		case CmdGetState:
			h.success(cmd.ID, cmd.Type, map[string]interface{}{
				"provider": h.model.Provider, "model": h.model.ID, "aborted": h.aborted,
			})
		case CmdSetModel:
			if cmd.Provider != "" && cmd.ModelID != "" {
				if m := h.reg.ResolveModel(cmd.Provider + "/" + cmd.ModelID); m != nil {
					h.model = m
					h.success(cmd.ID, cmd.Type, map[string]string{"provider": m.Provider, "model": m.ID})
				} else {
					h.error(cmd.ID, cmd.Type, "model not found")
				}
			} else {
				h.error(cmd.ID, cmd.Type, "provider and modelId required")
			}
		default:
			h.error(cmd.ID, cmd.Type, fmt.Sprintf("unknown: %s", cmd.Type))
		}
	}
}

func (h *Handler) handlePrompt(cmd Command) {
	if cmd.Message == "" {
		h.error(cmd.ID, cmd.Type, "message required")
		return
	}

	prov := h.reg.GetProvider(h.model.Provider)
	if prov == nil {
		h.error(cmd.ID, cmd.Type, "provider not found")
		return
	}

	apiKey := h.reg.ResolveAPIKeyForProvider(h.model.Provider, "")
	if apiKey == "" && h.model.Provider != "faux" {
		h.error(cmd.ID, cmd.Type, fmt.Sprintf("no API key (set %s)", strings.Join(prov.AuthEnvVars, " or ")))
		return
	}

	headers := map[string]string{}
	switch prov.API {
	case "openai-completions", "openai-responses":
		headers["Authorization"] = "Bearer " + apiKey
	case "anthropic-messages":
		headers["x-api-key"] = apiKey
		headers["anthropic-version"] = "2023-06-01"
	case "google-generative-ai":
		headers["x-goog-api-key"] = apiKey
	}

	client := protocol.NewHTTPClient(prov.BaseURL, headers)

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
		default:
			ch := make(chan model.StreamEvent, 1)
			ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported: " + prov.API})
			close(ch)
			return ch
		}
	}

	config := &loop.Config{
		Model:    h.model,
		Tools:    h.tools,
		MaxTurns: 10,
		StreamFn: streamFn,
	}

	userMsg := &model.UserMessage{
		Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
		Timestamp: time.Now().UnixMilli(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
	if err != nil {
		h.error(cmd.ID, cmd.Type, err.Error())
		return
	}

	// Collect assistant text
	var text strings.Builder
	var toolCalls []map[string]interface{}
	for _, msg := range messages {
		if msg.Assistant != nil {
			for _, block := range msg.Assistant.Content {
				switch block.Type {
				case model.ContentTypeText:
					text.WriteString(block.Text)
				case model.ContentTypeToolCall:
					toolCalls = append(toolCalls, map[string]interface{}{
						"id": block.ID, "name": block.Name, "arguments": json.RawMessage(block.Arguments),
					})
				}
			}
		}
	}

	h.success(cmd.ID, cmd.Type, map[string]interface{}{
		"message":   text.String(),
		"toolCalls": toolCalls,
		"done":      true,
	})
}

func (h *Handler) success(id, cmd string, data interface{}) {
	h.writer.Write(Response{ID: id, Type: "response", Command: cmd, Success: true, Data: data})
}

func (h *Handler) error(id, cmd, msg string) {
	h.writer.Write(Response{ID: id, Type: "response", Command: cmd, Success: false, Error: msg})
}
