// Package rpc implements the JSON-RPC headless protocol matching the
// TypeScript pi RPC protocol. Commands on stdin, responses/events on stdout.
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
	CmdCompact              = "compact"
	CmdBash                 = "bash"
	CmdAbortBash            = "abort_bash"
	CmdGetMessages          = "get_messages"
	CmdGetTree              = "get_tree"
	CmdGetEntries           = "get_entries"
	CmdGetSessionStats      = "get_session_stats"
	CmdGetLastAssistantText = "get_last_assistant_text"
	CmdFork                 = "fork"
	CmdClone                = "clone"
	CmdSwitchSession        = "switch_session"
	CmdSetSessionName       = "set_session_name"
	CmdExportHTML           = "export_html"
	CmdGetCommands          = "get_commands"
	CmdQuit                 = "quit"
	CmdExit                 = "exit"
)

type Command struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	// set_model
	Provider string `json:"provider,omitempty"`
	ModelID  string `json:"modelId,omitempty"`
	// set_thinking_level
	Level string `json:"level,omitempty"`
	// set_steering_mode / set_follow_up_mode
	Mode string `json:"mode,omitempty"`
	// bash
	Command_ string `json:"command,omitempty"`
	// compact
	CustomInstructions string `json:"customInstructions,omitempty"`
	// fork
	EntryID string `json:"entryId,omitempty"`
	// switch_session
	SessionPath string `json:"sessionPath,omitempty"`
	// set_session_name
	Name string `json:"name,omitempty"`
	// export_html
	OutputPath string `json:"outputPath,omitempty"`
	// new_session
	ParentSession string `json:"parentSession,omitempty"`
}

type Response struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// RunRPC starts the RPC loop on stdin/stdout.
func RunRPC(reg *provider.Registry, m *provider.ProviderModel, cmdCwd string) error {
	cwd := cmdCwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	tools := tool.NewRegistry()
	tools.Register(tool.NewReadTool(cwd))
	tools.Register(tool.NewWriteTool(cwd))
	tools.Register(tool.NewEditTool(cwd))
	tools.Register(tool.NewBashTool(cwd))

	h := &Handler{
		reader: jsonl.NewReader(os.Stdin),
		writer: jsonl.NewWriter(os.Stdout),
		reg:    reg,
		model:  m,
		cwd:    cwd,
		tools:  tools,
		qm:     queue.NewManager(queue.QueueModeAll),
		models: reg.ListProviders(),
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
	models        []string
	messages      []json.RawMessage
	thinkingLevel model.ThinkingLevel
	steeringMode  queue.QueueMode
	followUpMode  queue.QueueMode
}

func (h *Handler) loop() error {
	for {
		var cmd Command
		if err := h.reader.Decode(&cmd); err != nil {
			if err == io.EOF {
				return nil
			}
			h.errorCmd("", "", fmt.Sprintf("invalid: %v", err))
			continue
		}

		switch cmd.Type {
		case CmdQuit, CmdExit:
			h.successCmd(cmd.ID, cmd.Type, nil)
			return nil

		case CmdPrompt:
			h.handlePrompt(cmd)

		case CmdSteer:
			h.qm.PushSteering(&model.UserMessage{
				Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
				Timestamp: time.Now().UnixMilli(),
			})
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdFollowUp:
			h.qm.PushFollowUp(&model.UserMessage{
				Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
				Timestamp: time.Now().UnixMilli(),
			})
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdAbort:
			h.qm.Abort("user aborted")
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdNewSession:
			h.messages = nil
			h.qm = queue.NewManager(queue.QueueModeAll)
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdGetState:
			h.successCmd(cmd.ID, cmd.Type, map[string]interface{}{
				"provider":      h.model.Provider,
				"model":         h.model.ID,
				"thinkingLevel": string(h.thinkingLevel),
				"steeringMode":  string(h.steeringMode),
				"followUpMode":  string(h.followUpMode),
			})

		case CmdSetModel:
			h.handleSetModel(cmd)

		case CmdCycleModel:
			h.handleCycleModel(cmd)

		case CmdGetAvailableModels:
			var infos []map[string]interface{}
			for _, pid := range h.models {
				prov := h.reg.GetProvider(pid)
				if prov == nil {
					continue
				}
				for _, mc := range prov.Models {
					infos = append(infos, map[string]interface{}{
						"provider": pid, "model": mc.ID, "reasoning": mc.Reasoning,
					})
				}
			}
			h.successCmd(cmd.ID, cmd.Type, infos)

		case CmdSetThinkingLevel:
			h.thinkingLevel = model.ThinkingLevel(cmd.Level)
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdCycleThinkingLevel:
			levels := []model.ThinkingLevel{model.ThinkingOff, model.ThinkingMinimal, model.ThinkingLow, model.ThinkingMedium, model.ThinkingHigh}
			idx := -1
			for i, l := range levels {
				if l == h.thinkingLevel {
					idx = i
					break
				}
			}
			h.thinkingLevel = levels[(idx+1)%len(levels)]
			h.successCmd(cmd.ID, cmd.Type, map[string]string{"level": string(h.thinkingLevel)})

		case CmdSetSteeringMode:
			if cmd.Mode == "one-at-a-time" {
				h.steeringMode = queue.QueueModeOneAtATime
			} else {
				h.steeringMode = queue.QueueModeAll
			}
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdSetFollowUpMode:
			if cmd.Mode == "one-at-a-time" {
				h.followUpMode = queue.QueueModeOneAtATime
			} else {
				h.followUpMode = queue.QueueModeAll
			}
			h.successCmd(cmd.ID, cmd.Type, nil)

		case CmdBash:
			h.handleBash(cmd)

		case CmdGetMessages:
			h.successCmd(cmd.ID, cmd.Type, h.messages)

		case CmdGetTree, CmdGetEntries, CmdGetSessionStats:
			h.successCmd(cmd.ID, cmd.Type, []interface{}{})

		case CmdGetLastAssistantText:
			text := h.lastAssistantText()
			h.successCmd(cmd.ID, cmd.Type, map[string]string{"text": text})

		case CmdGetCommands:
			h.successCmd(cmd.ID, cmd.Type, allCommands())

		case CmdGetAvailableThinking:
			h.successCmd(cmd.ID, cmd.Type, []string{"off", "minimal", "low", "medium", "high"})

		default:
			// fork, clone, switch_session, set_session_name, export_html, compact,
			// abort_bash, set_auto_compaction, set_auto_retry, abort_retry
			h.successCmd(cmd.ID, cmd.Type, map[string]string{"status": "not implemented"})
		}
	}
}

func (h *Handler) handlePrompt(cmd Command) {
	if cmd.Message == "" {
		h.errorCmd(cmd.ID, cmd.Type, "message required")
		return
	}

	prov := h.reg.GetProvider(h.model.Provider)
	if prov == nil {
		h.errorCmd(cmd.ID, cmd.Type, "provider not found")
		return
	}

	apiKey := h.reg.ResolveAPIKeyForProvider(h.model.Provider, "")
	if apiKey == "" && h.model.Provider != "faux" {
		h.errorCmd(cmd.ID, cmd.Type, fmt.Sprintf("no API key (set %s)", strings.Join(prov.AuthEnvVars, " or ")))
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
		Model:        h.model,
		Tools:        h.tools,
		MaxTurns:     10,
		StreamFn:     streamFn,
		QueueManager: h.qm,
		OnEvent: func(evt model.StreamEvent) {
			switch evt.Type {
			case model.StreamEventTextDelta:
				h.emit(Event{Type: "text_delta", Data: evt.Delta})
			case model.StreamEventThinkingDelta:
				h.emit(Event{Type: "thinking_delta", Data: evt.Delta})
			case model.StreamEventToolCallDelta:
				h.emit(Event{Type: "toolcall_delta", Data: evt.Delta})
			case model.StreamEventToolCallStart:
				h.emit(Event{Type: "toolcall_start", Data: map[string]interface{}{
					"index": evt.ContentIndex,
				}})
			case model.StreamEventToolCallEnd:
				if evt.ToolCall != nil {
					h.emit(Event{Type: "toolcall_end", Data: map[string]interface{}{
						"id": evt.ToolCall.ID, "name": evt.ToolCall.Name,
						"arguments": json.RawMessage(evt.ToolCall.Arguments),
					}})
				}
			}
		},
	}

	userMsg := &model.UserMessage{
		Role: "user", Content: model.UserContent{model.NewTextContent(cmd.Message)},
		Timestamp: time.Now().UnixMilli(),
	}

	// Emit streaming events
	h.emit(Event{Type: "stream_start"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
	if err != nil {
		h.emit(Event{Type: "stream_error", Data: err.Error()})
		h.errorCmd(cmd.ID, cmd.Type, err.Error())
		return
	}

	// Store messages
	for _, msg := range messages {
		data, _ := json.Marshal(msg)
		h.messages = append(h.messages, data)
	}

	// Collect response
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

	h.emit(Event{Type: "stream_end"})
	h.successCmd(cmd.ID, cmd.Type, map[string]interface{}{
		"message":   text.String(),
		"toolCalls": toolCalls,
		"done":      true,
	})
}

func (h *Handler) handleSetModel(cmd Command) {
	if cmd.Provider == "" || cmd.ModelID == "" {
		h.errorCmd(cmd.ID, cmd.Type, "provider and modelId required")
		return
	}
	m := h.reg.ResolveModel(cmd.Provider + "/" + cmd.ModelID)
	if m == nil {
		h.errorCmd(cmd.ID, cmd.Type, "model not found")
		return
	}
	h.model = m
	h.successCmd(cmd.ID, cmd.Type, map[string]string{"provider": m.Provider, "model": m.ID})
}

func (h *Handler) handleCycleModel(cmd Command) {
	var allModels []*provider.ProviderModel
	for _, pid := range h.models {
		prov := h.reg.GetProvider(pid)
		if prov == nil {
			continue
		}
		for _, mc := range prov.Models {
			allModels = append(allModels, &provider.ProviderModel{
				ID: mc.ID, Provider: pid, API: prov.API, BaseURL: prov.BaseURL,
			})
		}
	}
	if len(allModels) == 0 {
		h.errorCmd(cmd.ID, cmd.Type, "no models available")
		return
	}
	idx := -1
	for i, m := range allModels {
		if m.ID == h.model.ID && m.Provider == h.model.Provider {
			idx = i
			break
		}
	}
	h.model = allModels[(idx+1)%len(allModels)]
	h.successCmd(cmd.ID, cmd.Type, map[string]string{"provider": h.model.Provider, "model": h.model.ID})
}

func (h *Handler) handleBash(cmd Command) {
	if cmd.Command_ == "" {
		h.errorCmd(cmd.ID, cmd.Type, "command required")
		return
	}
	result, err := h.tools.Execute(context.Background(), "", "bash", json.RawMessage(fmt.Sprintf(`{"command":%q}`, cmd.Command_)))
	if err != nil {
		h.errorCmd(cmd.ID, cmd.Type, err.Error())
		return
	}
	text := ""
	for _, b := range result.Content {
		if b.Type == model.ContentTypeText {
			text += b.Text
		}
	}
	h.successCmd(cmd.ID, cmd.Type, map[string]interface{}{"output": text, "isError": result.IsError})
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

func (h *Handler) emit(evt Event) {
	h.writer.Write(evt)
}

func (h *Handler) successCmd(id, cmd string, data interface{}) {
	h.writer.Write(Response{ID: id, Type: "response", Command: cmd, Success: true, Data: data})
}

func (h *Handler) errorCmd(id, cmd, msg string) {
	h.writer.Write(Response{ID: id, Type: "response", Command: cmd, Success: false, Error: msg})
}

func allCommands() []map[string]interface{} {
	return []map[string]interface{}{
		{"command": "prompt", "description": "Send a user message"},
		{"command": "steer", "description": "Inject a steering message"},
		{"command": "follow_up", "description": "Queue a follow-up message"},
		{"command": "abort", "description": "Abort current operation"},
		{"command": "new_session", "description": "Start a new session"},
		{"command": "get_state", "description": "Get current state"},
		{"command": "set_model", "description": "Switch model"},
		{"command": "cycle_model", "description": "Cycle to next model"},
		{"command": "get_available_models", "description": "List available models"},
		{"command": "set_thinking_level", "description": "Set thinking level"},
		{"command": "cycle_thinking_level", "description": "Cycle thinking level"},
		{"command": "bash", "description": "Execute a shell command"},
		{"command": "get_messages", "description": "Get message history"},
		{"command": "get_last_assistant_text", "description": "Get last assistant response text"},
		{"command": "get_commands", "description": "List available commands"},
		{"command": "quit", "description": "Exit"},
	}
}
