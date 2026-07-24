// Package credentials provides runtime credential resolution.
package credentials

import (
	"os"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// Resolve resolves credentials for a provider from multiple sources.
func Resolve(prov *provider.ProviderConfig) (apiKey string, headers map[string]string) {
	headers = make(map[string]string)

	// Check env vars
	for _, envVar := range prov.AuthEnvVars {
		if val := os.Getenv(envVar); val != "" {
			apiKey = val
			break
		}
	}

	// Try stored credential
	if apiKey == "" {
		configDir, _ := os.UserConfigDir()
		store := provider.NewCredentialStore(configDir)
		if cred, _ := store.Load(prov.ID); cred != nil && cred.Key != "" {
			apiKey = cred.Key
		}
	}

	// Build headers
	if apiKey != "" {
		switch prov.API {
		case "openai-completions", "openai-responses":
			headers["Authorization"] = "Bearer " + apiKey
		case "anthropic-messages":
			headers["x-api-key"] = apiKey
			headers["anthropic-version"] = "2023-06-01"
		case "google-generative-ai":
			headers["x-goog-api-key"] = apiKey
		case "bedrock-converse-stream":
			headers["Authorization"] = "Bearer " + apiKey
		}
	}

	return
}

// MaskKey masks an API key for logging (show first 4 and last 4 chars).
func MaskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "..." + key[len(key)-4:]
}
