package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAPIKey_EnvVar(t *testing.T) {
	os.Setenv("TEST_PROVIDER_KEY", "sk-test123")
	defer os.Unsetenv("TEST_PROVIDER_KEY")

	key := ResolveAPIKey(nil, "test", []string{"TEST_PROVIDER_KEY"}, "")
	if key != "sk-test123" {
		t.Errorf("expected sk-test123, got %q", key)
	}
}

func TestResolveAPIKey_StoreFallback(t *testing.T) {
	dir := t.TempDir()
	store := NewCredentialStore(dir)
	store.Save("test", &Credential{Key: "stored-key"})

	// No env var set, should use stored credential
	key := ResolveAPIKey(store, "test", []string{"NONEXISTENT_VAR"}, "")
	if key != "stored-key" {
		t.Errorf("expected stored-key, got %q", key)
	}
}

func TestResolveAPIKey_StoreFirstThenEnv(t *testing.T) {
	dir := t.TempDir()
	store := NewCredentialStore(dir)
	store.Save("test", &Credential{Key: "stored-key"})
	os.Setenv("TEST_KEY", "env-key")
	defer os.Unsetenv("TEST_KEY")

	// Stored credential takes priority over env
	key := ResolveAPIKey(store, "test", []string{"TEST_KEY"}, "")
	if key != "stored-key" {
		t.Errorf("expected stored-key (stored takes priority), got %q", key)
	}
}

func TestCredentialStore_SaveLoadDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewCredentialStore(dir)

	// Save
	if err := store.Save("deepseek", &Credential{Key: "sk-abc"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file permissions
	path := filepath.Join(dir, "credentials", "deepseek.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}

	// Load
	cred, err := store.Load("deepseek")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cred.Key != "sk-abc" {
		t.Errorf("key: %q", cred.Key)
	}

	// Delete
	if err := store.Delete("deepseek"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	cred, err = store.Load("deepseek")
	if err != nil || cred != nil {
		t.Error("should be nil after delete")
	}
}

func TestDetectCompat_DeepSeek(t *testing.T) {
	c := DetectCompat("deepseek", "https://api.deepseek.com")
	if c.SupportsStore {
		t.Error("deepseek should not support store")
	}
	if c.SupportsDeveloperRole {
		t.Error("deepseek should not support developer role")
	}
	if c.RequiresReasoningContentOnAssistantMessages != true {
		t.Error("deepseek should require reasoning content on assistant messages")
	}
	if c.ThinkingFormat != "deepseek" {
		t.Errorf("thinking format: %q", c.ThinkingFormat)
	}
}

func TestDetectCompat_OpenAI(t *testing.T) {
	c := DetectCompat("openai", "https://api.openai.com/v1")
	if !c.SupportsStore {
		t.Error("openai should support store")
	}
	if !c.SupportsDeveloperRole {
		t.Error("openai should support developer role")
	}
	if c.ThinkingFormat != "" {
		t.Errorf("thinking format should be empty, got %q", c.ThinkingFormat)
	}
}

func TestDetectCompat_Together(t *testing.T) {
	c := DetectCompat("together", "https://api.together.xyz")
	if c.MaxTokensField != "max_tokens" {
		t.Errorf("maxTokensField: %q", c.MaxTokensField)
	}
	if c.SupportsLongCacheRetention {
		t.Error("together should not support long cache retention")
	}
}

func TestRegistry_RegisterAndResolve(t *testing.T) {
	r := NewRegistry(nil)
	RegisterBuiltins(r)

	// List providers
	ids := r.ListProviders()
	if len(ids) < 4 {
		t.Fatalf("expected at least 4 providers, got %d: %v", len(ids), ids)
	}

	// Get specific provider
	prov := r.GetProvider("deepseek")
	if prov == nil || prov.BaseURL != "https://api.deepseek.com" {
		t.Errorf("deepseek provider: %+v", prov)
	}

	// Resolve model by ref
	m := r.ResolveModel("deepseek/deepseek-chat")
	if m == nil || m.Provider != "deepseek" || m.ID != "deepseek-chat" {
		t.Errorf("model: %+v", m)
	}

	// Resolve by model name only
	m2 := r.ResolveModel("deepseek-reasoner")
	if m2 == nil {
		t.Error("should find deepseek-reasoner")
	}

	// Unknown model
	if m3 := r.ResolveModel("nonexistent"); m3 != nil {
		t.Error("should return nil for unknown model")
	}
}

func TestRegistry_AuthEnvVars(t *testing.T) {
	r := NewRegistry(nil)
	RegisterBuiltins(r)

	prov := r.GetProvider("deepseek")
	if len(prov.AuthEnvVars) != 1 || prov.AuthEnvVars[0] != "DEEPSEEK_API_KEY" {
		t.Errorf("deepseek auth env vars: %v", prov.AuthEnvVars)
	}

	prov2 := r.GetProvider("google")
	if len(prov2.AuthEnvVars) != 2 {
		t.Errorf("google auth env vars: %v", prov2.AuthEnvVars)
	}
}
