// Package prompt builds system prompts from project context, tools, and skills.
package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/agent/tool"
)

// Skill represents a loaded skill definition.
type Skill struct {
	Name        string
	Description string
	FilePath    string
}

// Build constructs a system prompt for the agent.
func Build(opts Options) string {
	var parts []string

	// Base instruction
	if opts.Base != "" {
		parts = append(parts, opts.Base)
	}

	// Project context (AGENTS.md, CLAUDE.md)
	if len(opts.ContextFiles) > 0 {
		parts = append(parts, "")
		parts = append(parts, "<project_context>")
		for _, f := range opts.ContextFiles {
			parts = append(parts, f)
		}
		parts = append(parts, "</project_context>")
	}

	// Available skills
	if len(opts.Skills) > 0 {
		parts = append(parts, "")
		parts = append(parts, formatSkills(opts.Skills))
	}

	// Available tools
	if len(opts.ToolNames) > 0 {
		sort.Strings(opts.ToolNames)
		parts = append(parts, "")
		parts = append(parts, "You have access to the following tools:")
		for _, name := range opts.ToolNames {
			parts = append(parts, fmt.Sprintf("- %s", name))
		}
	}

	return strings.Join(parts, "\n")
}

// Options configures the system prompt builder.
type Options struct {
	Base         string
	ContextFiles []string
	Skills       []Skill
	ToolNames    []string
}

// DefaultBase returns the default base system prompt.
func DefaultBase() string {
	return `You are an expert coding assistant. You help users by reading files, executing commands, editing code, and writing new files.

Guidelines:
- Use bash for file operations like ls, rg, find
- Use read to examine files
- Use edit for precise changes
- Use write for new files or complete rewrites
- Be concise in your responses`
}

// LoadSkills scans a directory for SKILL.md files and returns parsed skills.
func LoadSkills(skillsDir string) []Skill {
	var skills []Skill
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}
		skill := parseSkillFile(entry.Name(), string(data), skillPath)
		if skill != nil {
			skills = append(skills, *skill)
		}
	}
	return skills
}

func parseSkillFile(name, content, path string) *Skill {
	// Parse name from first heading or use directory name
	skill := &Skill{
		Name:     name,
		FilePath: path,
	}

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		skill.Name = strings.TrimPrefix(lines[0], "# ")
	}

	// Find description: first non-empty, non-heading paragraph
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		skill.Description = line
		break
	}

	return skill
}

func formatSkills(skills []Skill) string {
	var lines []string
	lines = append(lines, "The following skills provide specialized instructions for specific tasks.")
	lines = append(lines, "Use the read tool to load a skill file when the task matches its description.")
	lines = append(lines, "")
	lines = append(lines, "<available_skills>")
	for _, s := range skills {
		lines = append(lines, fmt.Sprintf("  <skill>"))
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
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// ToolNames returns the names of all tools in a registry.
func ToolNames(reg *tool.Registry) []string {
	// We can't iterate the registry's internal map directly,
	// so we check the four built-in tools.
	names := []string{}
	for _, name := range []string{"read", "write", "edit", "bash"} {
		if reg.Get(name) != nil {
			names = append(names, name)
		}
	}
	return names
}
