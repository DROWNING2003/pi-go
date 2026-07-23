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
	workspace string // optional workspace root for relative paths
}

// NewReadTool creates a read tool with an optional workspace directory.
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

	// Safety: resolve symlinks and check boundaries
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return &Result{
			Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("cannot resolve path %s: %v", path, err))},
			IsError: true,
		}, nil
	}

	if t.workspace != "" {
		workspaceAbs, _ := filepath.Abs(t.workspace)
		if !strings.HasPrefix(resolved, workspaceAbs+string(filepath.Separator)) && resolved != workspaceAbs {
			return &Result{
				Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("path outside workspace: %s", path))},
				IsError: true,
			}, nil
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolved)
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
