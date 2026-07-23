package session

import (
	"testing"
)

func TestRegisterCleanup(t *testing.T) {
	called := false
	unregister := RegisterCleanup(func(sessionID string) {
		called = true
	})
	defer unregister()

	if err := Cleanup("test-session"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if !called {
		t.Error("cleanup not called")
	}
}

func TestCleanup_Empty(t *testing.T) {
	if err := Cleanup("test"); err != nil {
		t.Errorf("empty cleanup: %v", err)
	}
}

func TestUnregister(t *testing.T) {
	count := 0
	unregister := RegisterCleanup(func(sessionID string) { count++ })
	unregister()

	if err := Cleanup("test"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if count != 0 {
		t.Error("unregistered cleanup should not be called")
	}
}
