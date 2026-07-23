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
	Name                   string
	Description            string
	FilePath               string
	Content                string
	DisableModelInvocation bool
}

// Build constructs a system prompt for the agent.
func Build(opts Options) string {
	var parts []string
	if opts.Base != "" {
		parts = append(parts, opts.Base)
	}
	if len(opts.ContextFiles) > 0 {
		parts = append(parts, "", "<project_context>")
		for _, f := range opts.ContextFiles {
			parts = append(parts, f)
		}
		parts = append(parts, "</project_context>")
	}
	if len(opts.Skills) > 0 {
		parts = append(parts, "", formatSkills(opts.Skills))
	}
	if len(opts.ToolNames) > 0 {
		sort.Strings(opts.ToolNames)
		parts = append(parts, "", "You have access to the following tools:")
		for _, name := range opts.ToolNames {
			parts = append(parts, fmt.Sprintf("- %s", name))
		}
	}
	return strings.Join(parts, "\n")
}

type Options struct {
	Base         string
	ContextFiles []string
	Skills       []Skill
	ToolNames    []string
}

func DefaultBase() string {
	return `You are an expert coding assistant. You help users by reading files, executing commands, editing code, and writing new files.

Guidelines:
- Use bash for file operations like ls, rg, find
- Use read to examine files
- Use edit for precise changes
- Use write for new files or complete rewrites
- Be concise in your responses`
}

// LoadSkills recursively scans a directory for SKILL.md files, respecting .gitignore.
func LoadSkills(skillsDir string) []Skill {
	return loadSkillsRecursive(skillsDir, skillsDir, loadIgnoreRules(skillsDir))
}

func loadSkillsRecursive(rootDir, currentDir string, ig *ignoreMatcher) []Skill {
	var skills []Skill
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.Name() == "SKILL.md" && !entry.IsDir() {
			skillPath := filepath.Join(currentDir, entry.Name())
			relPath, _ := filepath.Rel(rootDir, skillPath)
			if ig != nil && ig.matches(relPath) {
				continue
			}
			data, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}
			skill := parseSkillFile(filepath.Base(currentDir), string(data), skillPath)
			if skill != nil {
				skills = append(skills, *skill)
			}
			return skills // Only one SKILL.md per directory
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" {
			continue
		}
		subDir := filepath.Join(currentDir, entry.Name())
		relPath, _ := filepath.Rel(rootDir, subDir)
		if ig != nil && ig.matches(relPath+"/") {
			continue
		}
		skills = append(skills, loadSkillsRecursive(rootDir, subDir, ig)...)
	}
	return skills
}

type ignoreMatcher struct {
	patterns []string
}

func loadIgnoreRules(dir string) *ignoreMatcher {
	var patterns []string
	for _, name := range []string{".gitignore", ".ignore"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			patterns = append(patterns, line)
		}
	}
	if len(patterns) == 0 {
		return nil
	}
	return &ignoreMatcher{patterns: patterns}
}

func (ig *ignoreMatcher) matches(path string) bool {
	for _, p := range ig.patterns {
		p = strings.TrimPrefix(p, "/")
		negate := false
		if strings.HasPrefix(p, "!") {
			negate = true
			p = p[1:]
		}
		matched, _ := filepath.Match(p, path)
		if !matched {
			matched, _ = filepath.Match(p, filepath.Base(path))
		}
		if matched {
			return !negate
		}
	}
	return false
}

func parseSkillFile(name, content, path string) *Skill {
	skill := &Skill{Name: name, FilePath: path, Content: content}
	// Parse YAML frontmatter
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---\n")
		if end > 0 {
			fm := content[4 : end+4]
			skill.Content = strings.TrimSpace(content[end+9:])
			for _, line := range strings.Split(fm, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "name:") {
					skill.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				} else if strings.HasPrefix(line, "description:") {
					skill.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				} else if strings.HasPrefix(line, "disable-model-invocation:") {
					skill.DisableModelInvocation = strings.TrimSpace(strings.TrimPrefix(line, "disable-model-invocation:")) == "true"
				}
			}
		}
	}
	// Fallback: extract name from first heading
	if skill.Name == name {
		for _, line := range strings.Split(skill.Content, "\n") {
			if strings.HasPrefix(line, "# ") {
				skill.Name = strings.TrimPrefix(line, "# ")
				break
			}
		}
	}
	if skill.Description == "" {
		for _, line := range strings.Split(skill.Content, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				skill.Description = line
				break
			}
		}
	}
	return skill
}

func formatSkills(skills []Skill) string {
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
	lines = append(lines, "Use the read tool to load a skill file when the task matches its description.")
	lines = append(lines, "", "<available_skills>")
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

func ToolNames(reg *tool.Registry) []string {
	var names []string
	for _, name := range []string{"read", "write", "edit", "bash", "web_fetch"} {
		if reg.Get(name) != nil {
			names = append(names, name)
		}
	}
	return names
}
