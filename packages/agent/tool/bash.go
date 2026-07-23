package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// BashTool implements shell command execution with timeout and cancel support.
type BashTool struct {
	workspace      string
	defaultTimeout time.Duration
}

func NewBashTool(workspace string) *BashTool {
	return &BashTool{workspace: workspace, defaultTimeout: 30 * time.Second}
}

func (t *BashTool) Name() string { return "bash" }
func (t *BashTool) Description() string {
	return "Execute a shell command and return stdout/stderr/exit code"
}
func (t *BashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "Shell command to execute"}
		},
		"required": ["command"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("invalid arguments: " + err.Error())}, IsError: true}, nil
	}

	if params.Command == "" {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("empty command")}, IsError: true}, nil
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, t.defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", params.Command)
	if t.workspace != "" {
		cmd.Dir = t.workspace
	}
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "[stderr]\n" + stderr.String()
	}

	if execCtx.Err() == context.DeadlineExceeded {
		return &Result{
			Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("command timed out after %v\n%s", t.defaultTimeout, truncate(output, 4000)))},
			IsError: true,
		}, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &Result{Content: []model.ContentBlock{model.NewTextContent("command error: " + err.Error())}, IsError: true}, nil
		}
	}

	text := truncate(output, 4000)
	if exitCode != 0 {
		text += fmt.Sprintf("\n[exit code: %d]", exitCode)
	}

	return &Result{
		Content: []model.ContentBlock{model.NewTextContent(text)},
		IsError: exitCode != 0,
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
