package jsonl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.Write(map[string]string{"key": "val"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if buf.String() != "{\"key\":\"val\"}\n" {
		t.Errorf("output: %q", buf.String())
	}
}

func TestReader(t *testing.T) {
	input := "{\"a\":1}\n{\"b\":2}\n"
	r := NewReader(strings.NewReader(input))

	var m map[string]int
	if err := r.Decode(&m); err != nil {
		t.Fatalf("decode 1: %v", err)
	}
	if m["a"] != 1 {
		t.Errorf("a: %d", m["a"])
	}
	if err := r.Decode(&m); err != nil {
		t.Fatalf("decode 2: %v", err)
	}
	if m["b"] != 2 {
		t.Errorf("b: %d", m["b"])
	}
	if err := r.Decode(&m); err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestReader_CRLF(t *testing.T) {
	input := "{\"a\":1}\r\n"
	r := NewReader(strings.NewReader(input))
	var m map[string]int
	if err := r.Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["a"] != 1 {
		t.Error("value mismatch")
	}
}
