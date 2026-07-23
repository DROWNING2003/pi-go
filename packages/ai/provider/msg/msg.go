// Package msg provides cross-provider message transformation matching
// the TypeScript transform-messages.ts behavior.
package msg

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

const (
	nonVisionPlaceholder     = "(image omitted: model does not support images)"
	nonVisionToolPlaceholder = "(tool image omitted: model does not support images)"
)

var toolCallIDPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// NormalizeToolCallID sanitizes a tool call ID for cross-provider compatibility.
func NormalizeToolCallID(id string) string {
	sanitized := toolCallIDPattern.ReplaceAllString(id, "_")
	if len(sanitized) > 64 {
		sanitized = sanitized[:64]
	}
	return strings.TrimRight(sanitized, "_")
}

// TransformMessages prepares messages for a target model:
// 1. Downgrades images for non-vision models
// 2. Converts thinking blocks to text for cross-model
// 3. Normalizes tool call IDs
// 4. Drops thought signatures for cross-model
func TransformMessages(messages []json.RawMessage, m *provider.ProviderModel) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(messages))
	toolCallIDMap := make(map[string]string) // original → normalized

	for _, raw := range messages {
		transformed := transformOne(raw, m, toolCallIDMap)
		if transformed != nil {
			result = append(result, transformed)
		}
	}
	return result
}

func transformOne(raw json.RawMessage, m *provider.ProviderModel, idMap map[string]string) json.RawMessage {
	var header struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(raw, &header) != nil {
		return raw
	}

	switch header.Role {
	case "user":
		return transformUser(raw, m)
	case "assistant":
		return transformAssistant(raw, m, idMap)
	case "toolResult":
		return transformToolResult(raw, idMap)
	}
	return raw
}

func transformUser(raw json.RawMessage, m *provider.ProviderModel) json.RawMessage {
	if hasImageInput(m) {
		return raw
	}
	// Downgrade images to text placeholders
	var msg model.UserMessage
	if json.Unmarshal(raw, &msg) != nil {
		return raw
	}
	if len(msg.Content) == 0 {
		return raw
	}

	var newContent []model.ContentBlock
	prevPlaceholder := false
	for _, b := range msg.Content {
		if b.Type == model.ContentTypeImage {
			if !prevPlaceholder {
				newContent = append(newContent, model.NewTextContent(nonVisionPlaceholder))
				prevPlaceholder = true
			}
		} else {
			newContent = append(newContent, b)
			prevPlaceholder = false
		}
	}
	msg.Content = newContent
	data, _ := json.Marshal(msg)
	return data
}

func transformAssistant(raw json.RawMessage, m *provider.ProviderModel, idMap map[string]string) json.RawMessage {
	var msg model.AssistantMessage
	if json.Unmarshal(raw, &msg) != nil {
		return raw
	}

	isSameModel := msg.Provider == m.Provider && msg.API == m.API && msg.Model == m.ID

	var newContent []model.ContentBlock
	for _, b := range msg.Content {
		switch b.Type {
		case model.ContentTypeThinking:
			if b.Redacted {
				if isSameModel {
					newContent = append(newContent, b)
				}
				// Drop redacted thinking for cross-model
				continue
			}
			if isSameModel {
				newContent = append(newContent, b)
			} else {
				// Convert thinking to text for cross-model
				newContent = append(newContent, model.NewTextContent(b.Thinking))
			}

		case model.ContentTypeText:
			newContent = append(newContent, b)

		case model.ContentTypeToolCall:
			tc := b
			if !isSameModel {
				// Drop thought signature for cross-model
				tc.ThoughtSignature = ""
				// Normalize tool call ID
				normalized := NormalizeToolCallID(tc.ID)
				if normalized != tc.ID {
					idMap[tc.ID] = normalized
				}
				tc.ID = normalized
			}
			newContent = append(newContent, tc)
		}
	}
	msg.Content = newContent
	data, _ := json.Marshal(msg)
	return data
}

func transformToolResult(raw json.RawMessage, idMap map[string]string) json.RawMessage {
	var msg model.ToolResultMessage
	if json.Unmarshal(raw, &msg) != nil {
		return raw
	}
	if normalized, ok := idMap[msg.ToolCallID]; ok {
		msg.ToolCallID = normalized
	}
	data, _ := json.Marshal(msg)
	return data
}

func hasImageInput(m *provider.ProviderModel) bool {
	for _, input := range m.Input {
		if input == "image" {
			return true
		}
	}
	return false
}
