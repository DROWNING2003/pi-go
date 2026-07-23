package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTool(t *testing.T) {
	dir := t.TempDir()
	w := NewWriteTool(dir)

	result, err := w.Execute(context.Background(), []byte(`{"path":"test.txt","content":"hello"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %+v", result.Content)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "test.txt"))
	if string(data) != "hello" {
		t.Errorf("file content: %q", data)
	}
}

func TestWriteTool_OutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	w := NewWriteTool(dir)

	result, _ := w.Execute(context.Background(), []byte(`{"path":"/etc/passwd","content":"x"}`))
	if !result.IsError {
		t.Error("should reject path outside workspace")
	}
}

func TestEditTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello world"), 0644)

	e := NewEditTool(dir)
	result, err := e.Execute(context.Background(), []byte(`{"path":"f.txt","oldText":"hello","newText":"hi"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %+v", result.Content)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	if string(data) != "hi world" {
		t.Errorf("file content: %q", data)
	}
}

func TestEditTool_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello hello"), 0644)

	e := NewEditTool(dir)
	result, _ := e.Execute(context.Background(), []byte(`{"path":"f.txt","oldText":"hello","newText":"hi"}`))
	if !result.IsError {
		t.Error("should reject non-unique match")
	}
}

func TestBashTool(t *testing.T) {
	dir := t.TempDir()
	b := NewBashTool(dir)

	result, err := b.Execute(context.Background(), []byte(`{"command":"echo hello"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %+v", result.Content)
	}
	if result.Content[0].Text != "hello\n" {
		t.Errorf("output: %q", result.Content[0].Text)
	}
}

func TestBashTool_ExitCode(t *testing.T) {
	b := NewBashTool("")

	result, err := b.Execute(context.Background(), []byte(`{"command":"exit 1"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.IsError {
		t.Error("should be error for non-zero exit code")
	}
}
