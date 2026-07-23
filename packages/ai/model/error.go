package model

import (
	"encoding/json"
	"fmt"
)

// DiagnosticErrorInfo captures structured error information from providers
// and runtime failures for diagnostics.
type DiagnosticErrorInfo struct {
	Name    string      `json:"name,omitempty"`
	Message string      `json:"message"`
	Stack   string      `json:"stack,omitempty"`
	Code    interface{} `json:"code,omitempty"`
}

// AssistantMessageDiagnostic records a redacted provider/runtime diagnostic
// event for failure analysis.
type AssistantMessageDiagnostic struct {
	Type      string               `json:"type"`
	Timestamp int64                `json:"timestamp"`
	Error     *DiagnosticErrorInfo `json:"error,omitempty"`
	Details   json.RawMessage      `json:"details,omitempty"`
}

// ModelError wraps model-level errors with stable error types.
type ModelError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Common model error constructors.
func NewInvalidContentTypeError(contentType string) *ModelError {
	return &ModelError{
		Type:    "invalid_content_type",
		Message: fmt.Sprintf("unknown content type: %s", contentType),
	}
}

func NewInvalidRoleError(role string) *ModelError {
	return &ModelError{
		Type:    "invalid_role",
		Message: fmt.Sprintf("unknown message role: %s", role),
	}
}

// ErrInvalidContentType is returned when a content block has an unsupported type.
var ErrInvalidContentType = &ModelError{Type: "invalid_content_type", Message: "content block type is not supported"}

// ErrInvalidRole is returned when a message has an unknown role.
var ErrInvalidRole = &ModelError{Type: "invalid_role", Message: "message role is not recognized"}
