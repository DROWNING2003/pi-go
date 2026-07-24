// Package rpc implements the JSON-RPC protocol matching TypeScript pi.
// Commands on stdin, responses + events on stdout.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	"github.com/DROWNING2003/pi-go/packages/agent/queue"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/jsonl"
)

// All TS RPC command types.
const (
	CmdPrompt               = "prompt"
	CmdSteer                = "steer"
	CmdFollowUp             = "follow_up"
	CmdAbort                = "abort"
	CmdNewSession           = "new_session"
	CmdGetState             = "get_state"
	CmdSetModel             = "set_model"
	CmdCycleModel           = "cycle_model"
	CmdGetAvailableModels   = "get_available_models"
	CmdSetThinkingLevel     = "set_thinking_level"
	CmdCycleThinkingLevel   = "cycle_thinking_level"
	CmdGetAvailableThinking = "get_available_thinking_levels"
	CmdSetSteeringMode      = "set_steering_mode"
	CmdSetFollowUpMode      = "set_follow_up_mode"
	CmdBash                 = "bash"
	CmdGetMessages          = "get_messages"
	CmdGetTree              = "get_tree"
	CmdGetEntries           = "get_entries"
	CmdGetSessionStats      = "get_session_stats"
	CmdGetLastAssistantText = "get_last_assistant_text"
	CmdSetSessionName       = "set_session_name"
	CmdGetCommands          = "get_commands"
	CmdQuit                 = "quit"
	CmdExit                 = "exit"
)

// Command from stdin.
type Command struct {
	ID             string `json:"id,omitempty"`
	Type           string `json:"type"`
	Message        string `json:"message,omitempty"`
	Provider       string `json:"provider,omitempty"`
	ModelID        string `json:"modelId,omitempty"`
	Level          string `json:"level,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Command_       string `json:"command,omitempty"`
	Name           string `json:"name,omitempty"`
	ExcludeFromCtx bool   `json:"excludeFromContext,omitempty"`
}

// Response to stdout. Matches TS RpcResponse union exactly.
type Response struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func successResp(id, cmd string, data interface{}) Response {
	return Response{ID: id, Type: "response", Command: cmd, Success: true, Data: data}
}
func successRespOK(id, cmd string) Response {
	return Response{ID: id, Type: "response", Command: cmd, Success: true}
}
func errorResp(id, cmd, msg string) Response {
	return Response{ID: id, Type: "response", Command: cmd, Success: false, Error: msg}
}

// RunRPC starts the RPC loop.
func RunRPC(reg *provider.Registry, m *provider.ProviderModel, cwd string) error {
	tools := tool.NewRegistry()
	tools.Register(tool.NewReadTool(cwd))
	tools.Register(tool.NewWriteTool(cwd))
	tools.Register(tool.NewEditTool(cwd))
	tools.Register(tool.NewBashTool(cwd))
	tools.Register(tool.NewWebFetchTool())

	h := &Handler{
		reader: jsonl.NewReader(os.Stdin),
		writer: jsonl.NewWriter(os.Stdout),
		reg:    reg,
		model:  m,
		cwd:    cwd,
		tools:  tools,
		qm:     queue.NewManager(queue.QueueModeAll),
	}
	return h.loop()
}

type Handler struct {
	reader        *jsonl.Reader
	writer        *jsonl.Writer
	reg           *provider.Registry
	model         *provider.ProviderModel
	cwd           string
	tools         *tool.Registry
	qm            *queue.Manager
	messages      []json.RawMessage
	thinkingLevel model.ThinkingLevel
}

func (h *Handler) loop() error {
	for {
		var cmd Command
		if err := h.reader.Decode(&cmd); err != nil {
			if err == io.EOF {
				return nil
			}
			h.write(errorResp("", "", fmt.Sprintf("invalid: %v", err)))
			continue
		}

		switch cmd.Type {
		case CmdQuit, CmdExit:
			h.write(successRespOK(cmd.ID, cmd.Type))
			return nil

		case CmdPrompt:
			h.handlePrompt(cmd)

		case CmdSteer:
			h.qm.PushSteering(&model.UserMessage{
				Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
				Timestamp: time.Now().UnixMilli(),
			})
			h.write(successRespOK(cmd.ID, cmd.Type))

		case CmdFollowUp:
			h.qm.PushFollowUp(&model.UserMessage{
				Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
				Timestamp: time.Now().UnixMilli(),
			})
			h.write(successRespOK(cmd.ID, cmd.Type))

		case CmdAbort:
			h.qm.Abort("user aborted")
			h.write(successRespOK(cmd.ID, cmd.Type))

		case CmdNewSession:
			h.messages = nil
			h.qm = queue.NewManager(queue.QueueModeAll)
			h.write(successResp(cmd.ID, cmd.Type, map[string]bool{"cancelled": false}))

		case CmdGetState:
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{
				"model":                 map[string]string{"provider": h.model.Provider, "model": h.model.ID},
				"thinkingLevel":         string(h.thinkingLevel),
				"isStreaming":           false,
				"steeringMode":          "all",
				"followUpMode":          "all",
				"sessionId":             "",
				"autoCompactionEnabled": false,
				"messageCount":          len(h.messages),
				"pendingMessageCount":   0,
			}))

		case CmdSetModel:
			if cmd.Provider != "" && cmd.ModelID != "" {
				if m := h.reg.ResolveModel(cmd.Provider + "/" + cmd.ModelID); m != nil {
					h.model = m
					h.write(successResp(cmd.ID, cmd.Type, m))
				} else {
					h.write(errorResp(cmd.ID, cmd.Type, "model not found"))
				}
			} else {
				h.write(errorResp(cmd.ID, cmd.Type, "provider and modelId required"))
			}

		case CmdCycleModel:
			models := h.allModels()
			if len(models) == 0 {
				h.write(errorResp(cmd.ID, cmd.Type, "no models"))
				return nil
			}
			idx := -1
			for i, mm := range models {
				if mm.ID == h.model.ID && mm.Provider == h.model.Provider {
					idx = i
					break
				}
			}
			h.model = models[(idx+1)%len(models)]
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{
				"model": h.model, "thinkingLevel": string(h.thinkingLevel), "isScoped": false,
			}))

		case CmdGetAvailableModels:
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"models": h.allModels()}))

		case CmdSetThinkingLevel:
			h.thinkingLevel = model.ThinkingLevel(cmd.Level)
			h.write(successRespOK(cmd.ID, cmd.Type))

		case CmdCycleThinkingLevel:
			levels := []model.ThinkingLevel{"off", "minimal", "low", "medium", "high"}
			idx := -1
			for i, l := range levels {
				if l == h.thinkingLevel {
					idx = i
					break
				}
			}
			h.thinkingLevel = levels[(idx+1)%len(levels)]
			h.write(successResp(cmd.ID, cmd.Type, map[string]string{"level": string(h.thinkingLevel)}))

		case CmdSetSteeringMode, CmdSetFollowUpMode:
			h.write(successRespOK(cmd.ID, cmd.Type))

		case CmdBash:
			result, err := h.tools.Execute(context.Background(), "", "bash", json.RawMessage(fmt.Sprintf(`{"command":%q}`, cmd.Command_)))
			if err != nil {
				h.write(errorResp(cmd.ID, cmd.Type, err.Error()))
			} else {
				text := ""
				for _, b := range result.Content {
					if b.Type == model.ContentTypeText {
						text += b.Text
					}
				}
				h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"output": text}))
			}

		case CmdGetMessages:
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"messages": h.messages}))

		case CmdGetLastAssistantText:
			text := h.lastAssistantText()
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"text": text}))

		case CmdGetCommands:
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"commands": allCommands()}))

		case CmdGetAvailableThinking:
			h.write(successResp(cmd.ID, cmd.Type, map[string]interface{}{"levels": []string{"off", "minimal", "low", "medium", "high"}}))

		default:
			h.write(successRespOK(cmd.ID, cmd.Type))
		}
	}
}

func (h *Handler) handlePrompt(cmd Command) {
	if cmd.Message == "" {
		h.write(errorResp(cmd.ID, cmd.Type, "message required"))
		return
	}

	prov := h.reg.GetProvider(h.model.Provider)
	if prov == nil {
		h.write(errorResp(cmd.ID, cmd.Type, "provider not found"))
		return
	}

	apiKey := h.reg.ResolveAPIKeyForProvider(h.model.Provider, "")
	if apiKey == "" && h.model.Provider != "faux" {
		h.write(errorResp(cmd.ID, cmd.Type, fmt.Sprintf("no API key (set %s)", strings.Join(prov.AuthEnvVars, " or "))))
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
			ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported"})
			close(ch)
			return ch
		}
	}

	// TS protocol: respond immediately with success, then stream events
	h.write(successRespOK(cmd.ID, cmd.Type))

	config := &loop.Config{
		Model:        h.model,
		Tools:        h.tools,
		MaxTurns:     10,
		StreamFn:     streamFn,
		QueueManager: h.qm,
	}

	userMsg := &model.UserMessage{
		Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
		Timestamp: time.Now().UnixMilli(),
	}

	ctx := context.Background()
	messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
	if err != nil {
		// Send error as agent event
		h.write(map[string]interface{}{"type": "agent_end", "messages": []interface{}{}})
		return
	}

	for _, msg := range messages {
		data, _ := json.Marshal(msg)
		h.messages = append(h.messages, data)
	}

	// TS protocol: emit agent_end event with messages
	h.write(map[string]interface{}{
		"type":     "agent_end",
		"messages": h.messages,
	})
}

func (h *Handler) allModels() []*provider.ProviderModel {
	var all []*provider.ProviderModel
	for _, pid := range h.reg.ListProviders() {
		prov := h.reg.GetProvider(pid)
		if prov == nil {
			continue
		}
		for _, mc := range prov.Models {
			all = append(all, &provider.ProviderModel{
				ID: mc.ID, Provider: pid, API: prov.API, BaseURL: prov.BaseURL,
				Name: mc.Name, Reasoning: mc.Reasoning, Input: mc.Input,
				ContextWindow: int64(mc.ContextWindow), MaxTokens: int64(mc.MaxTokens),
			})
		}
	}
	return all
}

func (h *Handler) lastAssistantText() string {
	for i := len(h.messages) - 1; i >= 0; i-- {
		var msg struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal(h.messages[i], &msg) == nil && msg.Role == "assistant" {
			for _, b := range msg.Content {
				if b.Type == "text" {
					return b.Text
				}
			}
		}
	}
	return ""
}

func (h *Handler) write(v interface{}) {
	h.writer.Write(v)
}

func allCommands() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "prompt", "description": "Send a message", "source": "prompt", "sourceInfo": map[string]string{}},
		{"name": "steer", "description": "Inject steering", "source": "prompt", "sourceInfo": map[string]string{}},
		{"name": "follow_up", "description": "Queue follow-up", "source": "prompt", "sourceInfo": map[string]string{}},
		{"name": "set_model", "description": "Switch model", "source": "prompt", "sourceInfo": map[string]string{}},
		{"name": "bash", "description": "Run command", "source": "prompt", "sourceInfo": map[string]string{}},
	}
}
