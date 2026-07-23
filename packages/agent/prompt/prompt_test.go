package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	result := Build(Options{
		Base:         "You are helpful.",
		ContextFiles: []string{"# Project rules\nBe concise."},
		Skills: []Skill{
			{Name: "test-skill", Description: "A test skill", FilePath: "/tmp/test-skill/SKILL.md"},
		},
		ToolNames: []string{"read", "write"},
	})

	if !strings.Contains(result, "You are helpful.") {
		t.Error("missing base")
	}
	if !strings.Contains(result, "<project_context>") {
		t.Error("missing project context")
	}
	if !strings.Contains(result, "<available_skills>") {
		t.Error("missing skills")
	}
	if !strings.Contains(result, "test-skill") {
		t.Error("missing skill name")
	}
	if !strings.Contains(result, "- read") {
		t.Error("missing tool")
	}
}

func TestBuild_EmptyOptions(t *testing.T) {
	result := Build(Options{})
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestLoadSkills(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`# Test Skill
This skill does testing.`), 0644)

	skills := LoadSkills(dir)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "Test Skill" {
		t.Errorf("name: %q", skills[0].Name)
	}
	if skills[0].Description != "This skill does testing." {
		t.Errorf("description: %q", skills[0].Description)
	}
}

func TestLoadSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	skills := LoadSkills(dir)
	if len(skills) != 0 {
		t.Error("expected no skills")
	}
}
