package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// ReadTool implements file reading with path safety checks.
type ReadTool struct {
	workspace string
}

func NewReadTool(workspace string) *ReadTool {
	return &ReadTool{workspace: workspace}
}

func (t *ReadTool) Name() string { return "read" }
func (t *ReadTool) Description() string {
	return "Read the contents of a file at the given path"
}
func (t *ReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to read"}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{
			Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("invalid arguments: %v", err))},
			IsError: true,
		}, nil
	}

	path := params.Path
	if t.workspace != "" && !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	if !isSafePath(t.workspace, path) {
		return &Result{
			Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("path outside workspace: %s", path))},
			IsError: true,
		}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Result{
				Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("file not found: %s", path))},
				IsError: true,
			}, nil
		}
		return &Result{
			Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("read error: %v", err))},
			IsError: true,
		}, nil
	}

	return &Result{
		Content: []model.ContentBlock{model.NewTextContent(string(data))},
	}, nil
}

// resolvePath resolves symlinks in a path, handling non-existent files.
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}
	// File doesn't exist yet - resolve parent and rejoin
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err == nil {
		return filepath.Join(resolvedParent, base)
	}
	return path
}

func isSafePath(workspace, path string) bool {
	if workspace == "" {
		return true
	}
	resolvedPath := resolvePath(path)
	resolvedWorkspace := resolvePath(workspace)
	sep := string(filepath.Separator)
	return strings.HasPrefix(resolvedPath, resolvedWorkspace+sep) || resolvedPath == resolvedWorkspace
}
