// Package prompt provides prompt template loading and system prompt building.
package prompt

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template is a loaded prompt template (.md file with optional YAML frontmatter).
type Template struct {
	Name        string
	Description string
	ArgHint     string
	Content     string
	FilePath    string
}

// LoadTemplates scans directories for .md files and parses them as templates.
func LoadTemplates(paths ...string) ([]Template, []string) {
	var templates []Template
	var warnings []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				warnings = append(warnings, "read dir "+path+": "+err.Error())
				continue
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				t, warn := loadTemplateFile(filepath.Join(path, e.Name()))
				if t != nil {
					templates = append(templates, *t)
				}
				if warn != "" {
					warnings = append(warnings, warn)
				}
			}
		} else if strings.HasSuffix(path, ".md") {
			t, warn := loadTemplateFile(path)
			if t != nil {
				templates = append(templates, *t)
			}
			if warn != "" {
				warnings = append(warnings, warn)
			}
		}
	}
	return templates, warnings
}

func loadTemplateFile(path string) (*Template, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "read " + path + ": " + err.Error()
	}
	return ParseTemplate(path, string(data))
}

// ParseTemplate parses a markdown template with optional YAML frontmatter.
// Frontmatter format:
//
//	---
//	description: "..."
//	argument-hint: "..."
//	---
//	Template body...
func ParseTemplate(path, content string) (*Template, string) {
	t := &Template{
		FilePath: filepath.Base(path),
		Content:  content,
	}

	// Parse YAML frontmatter (--- delimiters)
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---\n")
		if end > 0 {
			fm := content[4 : end+4]
			t.Content = content[end+9:]
			parseFrontmatter(t, fm)
		}
	}

	// Extract name from first heading or filename
	lines := strings.Split(t.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			t.Name = strings.TrimPrefix(line, "# ")
			break
		}
	}
	if t.Name == "" {
		t.Name = strings.TrimSuffix(path, ".md")
	}

	return t, ""
}

func parseFrontmatter(t *Template, fm string) {
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip quotes
		val = strings.Trim(val, "\"'")

		switch key {
		case "description":
			t.Description = val
		case "argument-hint":
			t.ArgHint = val
		}
	}
}

// FormatAsPrompt formats templates for inclusion in the system prompt.
func FormatTemplates(templates []Template) string {
	if len(templates) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "Available prompt templates:")
	for _, t := range templates {
		lines = append(lines, "- "+t.Name)
		if t.Description != "" {
			lines = append(lines, "  "+t.Description)
		}
	}
	return strings.Join(lines, "\n")
}
