package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Credential represents a stored API credential.
type Credential struct {
	Key string            `json:"key,omitempty"`
	Env map[string]string `json:"env,omitempty"`
}

// CredentialStore loads and saves credentials from the config directory.
type CredentialStore struct {
	dir string
}

// NewCredentialStore creates a store rooted at the given directory.
func NewCredentialStore(configDir string) *CredentialStore {
	return &CredentialStore{dir: configDir}
}

// Load reads the stored credential for a provider.
func (s *CredentialStore) Load(providerID string) (*Credential, error) {
	path := filepath.Join(s.dir, "credentials", providerID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read credential: %w", err)
	}
	var cred Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("decode credential: %w", err)
	}
	return &cred, nil
}

// Save persists a credential for a provider with 0600 permissions.
func (s *CredentialStore) Save(providerID string, cred *Credential) error {
	dir := filepath.Join(s.dir, "credentials")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("encode credential: %w", err)
	}
	path := filepath.Join(dir, providerID+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write credential: %w", err)
	}
	return nil
}

// Delete removes a stored credential.
func (s *CredentialStore) Delete(providerID string) error {
	path := filepath.Join(s.dir, "credentials", providerID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete credential: %w", err)
	}
	return nil
}

// ResolveAPIKey looks up an API key for a provider. It checks:
// 1. Stored credential file
// 2. Environment variables (provider-specific, in order)
// 3. APIKey from StreamOptions
func ResolveAPIKey(store *CredentialStore, providerID string, envVars []string, optsKey string) string {
	// 1. Stored credential
	if store != nil {
		if cred, _ := store.Load(providerID); cred != nil && cred.Key != "" {
			return cred.Key
		}
	}

	// 2. Environment variables
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}

	// 3. Options override
	return optsKey
}
