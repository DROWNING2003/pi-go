// Package harness provides agent harness types matching TS harness/types.ts
package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// Result represents a fallible operation result (TS Result<T, E>).
type Result[T any] struct {
	Ok    bool
	Value T
	Error error
}

func Ok[T any](value T) Result[T]    { return Result[T]{Ok: true, Value: value} }
func Err[T any](err error) Result[T] { return Result[T]{Ok: false, Error: err} }

// ToError normalizes an unknown value to an error.
func ToError(v interface{}) error {
	if v == nil {
		return nil
	}
	if e, ok := v.(error); ok {
		return e
	}
	if s, ok := v.(string); ok {
		return fmt.Errorf("%s", s)
	}
	return fmt.Errorf("%v", v)
}

// Skill is a loaded skill definition (TS Skill interface).
type Skill struct {
	Name                   string `json:"name"`
	Description            string `json:"description"`
	Content                string `json:"content"`
	FilePath               string `json:"filePath"`
	DisableModelInvocation bool   `json:"disableModelInvocation,omitempty"`
}

// PromptTemplate is a reusable prompt template (TS PromptTemplate interface).
type PromptTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ArgHint     string `json:"argHint,omitempty"`
	Content     string `json:"content"`
}

// SessionContext holds the derived session state (TS SessionContext).
type SessionContext struct {
	SystemPrompt  string
	Messages      []json.RawMessage
	Model         *provider.ProviderModel
	ThinkingLevel model.ThinkingLevel
	ActiveTools   []string
}

// ExecutionEnv abstracts file system operations for testing (TS ExecutionEnv).
type ExecutionEnv struct {
	CWD string
}

func NewExecutionEnv(cwd string) *ExecutionEnv {
	return &ExecutionEnv{CWD: cwd}
}

func (e *ExecutionEnv) ReadFile(path string) Result[string] {
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.CWD, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Err[string](err)
	}
	return Ok(string(data))
}

func (e *ExecutionEnv) WriteFile(path, content string) Result[struct{}] {
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.CWD, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return Err[struct{}](err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return Err[struct{}](err)
	}
	return Ok(struct{}{})
}

func (e *ExecutionEnv) ListDir(path string) Result[[]os.DirEntry] {
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.CWD, path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return Err[[]os.DirEntry](err)
	}
	return Ok(entries)
}

func (e *ExecutionEnv) FileInfo(path string) Result[os.FileInfo] {
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.CWD, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Err[os.FileInfo](err)
	}
	return Ok(info)
}

// FormatSkillsForSystemPrompt formats skills as XML for the system prompt (TS formatSkillsForSystemPrompt).
func FormatSkillsForSystemPrompt(skills []Skill) string {
	visible := make([]Skill, 0)
	for _, s := range skills {
		if !s.DisableModelInvocation {
			visible = append(visible, s)
		}
	}
	if len(visible) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "The following skills provide specialized instructions for specific tasks.")
	lines = append(lines, "Read the full skill file when the task matches its description.")
	lines = append(lines, "When a skill file references a relative path, resolve it against the skill directory and use that absolute path in tool commands.")
	lines = append(lines, "")
	lines = append(lines, "<available_skills>")
	for _, s := range visible {
		lines = append(lines, "  <skill>")
		lines = append(lines, fmt.Sprintf("    <name>%s</name>", escapeXML(s.Name)))
		lines = append(lines, fmt.Sprintf("    <description>%s</description>", escapeXML(s.Description)))
		lines = append(lines, fmt.Sprintf("    <location>%s</location>", escapeXML(s.FilePath)))
		lines = append(lines, "  </skill>")
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n")
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
