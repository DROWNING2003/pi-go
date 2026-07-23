package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// NewDiagnostic creates a diagnostic entry for an error.
func NewDiagnostic(diagType string, err error) AssistantMessageDiagnostic {
	diag := AssistantMessageDiagnostic{
		Type:      diagType,
		Timestamp: time.Now().UnixMilli(),
	}
	if err != nil {
		diag.Error = &DiagnosticErrorInfo{
			Message: err.Error(),
		}
	}
	return diag
}

// FormatError formats an error for provider error messages.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ContentText extracts text from content blocks.
func ContentText(blocks []ContentBlock) string {
	var result string
	for _, b := range blocks {
		if b.Type == ContentTypeText {
			result += b.Text
		}
	}
	return result
}

// ParseStreamingJSON attempts to parse partial JSON, returning what's valid.
func ParseStreamingJSON(partial string) json.RawMessage {
	if json.Valid([]byte(partial)) {
		return json.RawMessage(partial)
	}
	// Try to close incomplete JSON
	for _, suffix := range []string{"}", "]", `"]`, `"}`, `"}}`, `"]}`, `}]`, `"}]`, `]}`} {
		candidate := partial + suffix
		if json.Valid([]byte(candidate)) {
			return json.RawMessage(candidate)
		}
	}
	return json.RawMessage(partial)
}

// ValidateToolArguments validates JSON arguments against a JSON Schema.
// This is a simplified validation; returns nil if arguments are valid JSON.
func ValidateToolArguments(args json.RawMessage) error {
	if !json.Valid(args) {
		return fmt.Errorf("invalid JSON arguments")
	}
	return nil
}
