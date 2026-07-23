package util

import (
	"strings"
	"testing"
)

func TestUUIDv7(t *testing.T) {
	id := UUIDv7()
	if len(id) != 36 {
		t.Errorf("length: %d", len(id))
	}
	if !strings.Contains(id, "-") {
		t.Error("should contain hyphens")
	}
	// Should be unique
	id2 := UUIDv7()
	if id == id2 {
		t.Error("should be unique")
	}
}

func TestEstimateTokens(t *testing.T) {
	if n := EstimateTokens("hello"); n != 2 {
		t.Errorf("5 chars: %d", n)
	}
	if n := EstimateTokens(""); n != 0 {
		t.Errorf("empty: %d", n)
	}
}

func TestShortHash(t *testing.T) {
	h1 := ShortHash("hello")
	h2 := ShortHash("hello")
	if h1 != h2 {
		t.Error("should be deterministic")
	}
	if len(h1) != 8 {
		t.Errorf("length: %d", len(h1))
	}
}
