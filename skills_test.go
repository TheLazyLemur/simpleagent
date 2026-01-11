package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillsMeta(t *testing.T) {
	// Change to project root for skill discovery
	wd, _ := os.Getwd()
	defer os.Chdir(wd)

	// Should find the test skill
	skills := LoadSkillsMeta()

	found := false
	for _, s := range skills {
		if s.Name == "test" {
			found = true
			if s.Description != "Test skill for verification" {
				t.Errorf("unexpected description: %s", s.Description)
			}
		}
	}

	if !found {
		t.Error("test skill not found")
	}
}

func TestLoadSkill(t *testing.T) {
	// First load meta to populate skillPaths
	LoadSkillsMeta()

	skill, err := LoadSkill("test")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	if skill.Name != "test" {
		t.Errorf("unexpected name: %s", skill.Name)
	}

	if len(skill.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(skill.AllowedTools))
	}

	if skill.Content == "" {
		t.Error("content is empty")
	}

	_ = filepath.Join(".claude", "skills", "test") // suppress unused
	if !filepath.IsAbs(skill.Dir) {
		t.Logf("skill dir: %s", skill.Dir)
	}
	if filepath.Base(filepath.Dir(skill.Dir)) != "skills" {
		t.Errorf("unexpected dir: %s", skill.Dir)
	}
}

func TestFormatSkillsSection(t *testing.T) {
	skills := []SkillMeta{
		{Name: "foo", Description: "Foo skill"},
		{Name: "bar", Description: "Bar skill"},
	}

	section := FormatSkillsSection(skills)

	if section == "" {
		t.Error("section is empty")
	}

	if !contains(section, "<available-skills>") {
		t.Error("missing opening tag")
	}

	if !contains(section, "- foo: Foo skill") {
		t.Error("missing foo skill")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
