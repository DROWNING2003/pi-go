package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

const MaxProviderErrorBodyChars = 4000

// NormalizedProviderError matches TS NormalizedProviderError.
type NormalizedProviderError struct {
	Status             int
	Body               string
	Message            string
	MessageCarriesBody bool
}

// NormalizeError normalizes a provider error for display.
func NormalizeError(err error) NormalizedProviderError {
	if err == nil {
		return NormalizedProviderError{Message: "unknown error"}
	}
	msg := err.Error()
	return NormalizedProviderError{
		Message:            msg,
		MessageCarriesBody: false,
	}
}

// FormatProviderError formats a normalized error for display.
func FormatProviderError(norm NormalizedProviderError, prefix string) string {
	if norm.MessageCarriesBody || norm.Status == 0 || norm.Body == "" {
		if prefix != "" && norm.Status != 0 {
			return fmt.Sprintf("%s (%d): %s", prefix, norm.Status, norm.Message)
		}
		return norm.Message
	}
	if prefix != "" {
		return fmt.Sprintf("%s (%d): %s", prefix, norm.Status, norm.Body)
	}
	return fmt.Sprintf("%d: %s", norm.Status, norm.Body)
}

// TruncateErrorText truncates error text to maxChars.
func TruncateErrorText(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return fmt.Sprintf("%s... [truncated %d chars]", text[:maxChars], len(text)-maxChars)
}

// SafeJSONStringify safely converts a value to JSON string.
func SafeJSONStringify(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

// Ensure fmt and strings are used.
var _ = fmt.Sprintf
var _ = strings.TrimSpace
