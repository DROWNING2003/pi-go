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

// EditTool implements targeted text replacements in files.
type EditTool struct {
	workspace string
}

func NewEditTool(workspace string) *EditTool {
	return &EditTool{workspace: workspace}
}

func (t *EditTool) Name() string { return "edit" }
func (t *EditTool) Description() string {
	return "Replace a specific text in a file. The old text must match exactly once."
}
func (t *EditTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to edit"},
			"oldText": {"type": "string", "description": "Exact text to replace"},
			"newText": {"type": "string", "description": "Replacement text"}
		},
		"required": ["path", "oldText", "newText"]
	}`)
}

func (t *EditTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Path    string `json:"path"`
		OldText string `json:"oldText"`
		NewText string `json:"newText"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("invalid arguments: " + err.Error())}, IsError: true}, nil
	}

	path := params.Path
	if t.workspace != "" && !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	if !isSafePath(t.workspace, path) {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("path outside workspace: " + path)}, IsError: true}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("read error: " + err.Error())}, IsError: true}, nil
	}

	content := string(data)
	count := strings.Count(content, params.OldText)
	if count == 0 {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("oldText not found in file")}, IsError: true}, nil
	}
	if count > 1 {
		return &Result{Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("oldText matches %d times, must be unique", count))}, IsError: true}, nil
	}

	newContent := strings.Replace(content, params.OldText, params.NewText, 1)
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("write error: " + err.Error())}, IsError: true}, nil
	}

	return &Result{Content: []model.ContentBlock{model.NewTextContent("file edited successfully")}}, nil
}
