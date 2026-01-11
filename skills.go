package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a loaded skill with full content
type Skill struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	Path         string   // SKILL.md path
	Dir          string   // skill directory
	Content      string   // body after frontmatter
}

// SkillMeta is minimal info for system prompt injection
type SkillMeta struct {
	Name        string
	Description string
}

// Global skill registry (populated by LoadSkillsMeta)
var skillPaths = make(map[string]string) // name -> SKILL.md path

// LoadSkillsMeta discovers skills and returns metadata for system prompt
func LoadSkillsMeta() []SkillMeta {
	var skills []SkillMeta
	skillPaths = make(map[string]string)

	// Search directories (project takes precedence)
	dirs := []string{".claude/skills"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".claude/skills"))
	}

	for _, baseDir := range dirs {
		dir := findDir(baseDir)
		if dir == "" {
			// For home dir, check directly
			if filepath.IsAbs(baseDir) {
				if info, err := os.Stat(baseDir); err == nil && info.IsDir() {
					dir = baseDir
				}
			}
		}
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			data, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}

			fm, _ := ParseFrontmatter(string(data))
			if fm == "" {
				continue
			}

			var skill Skill
			if err := yaml.Unmarshal([]byte(fm), &skill); err != nil {
				continue
			}

			if skill.Name == "" || skill.Description == "" {
				continue
			}

			// Skip if already registered (project takes precedence)
			if _, exists := skillPaths[skill.Name]; exists {
				continue
			}

			skillPaths[skill.Name] = skillPath
			skills = append(skills, SkillMeta{
				Name:        skill.Name,
				Description: skill.Description,
			})
		}
	}

	return skills
}

// FormatSkillsSection formats skills for system prompt injection
func FormatSkillsSection(skills []SkillMeta) string {
	if len(skills) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "<available-skills>")
	lines = append(lines, "Use invoke_skill when a task matches a skill description.")
	lines = append(lines, "")
	for _, s := range skills {
		lines = append(lines, "- "+s.Name+": "+s.Description)
	}
	lines = append(lines, "</available-skills>")
	return strings.Join(lines, "\n")
}

// LoadSkill loads full skill content by name
func LoadSkill(name string) (*Skill, error) {
	path, ok := skillPaths[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fm, body := ParseFrontmatter(string(data))

	var skill Skill
	if err := yaml.Unmarshal([]byte(fm), &skill); err != nil {
		return nil, err
	}

	skill.Path = path
	skill.Dir = filepath.Dir(path)
	skill.Content = strings.TrimSpace(body)

	return &skill, nil
}
