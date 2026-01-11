package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"simpleagent/tools"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

const configDirName = ".simpleagent"
const configFileName = "config.json"
const rulesDirName = ".rules"

const systemInstructions = `<system-instructions>
You are a coding assistant with filesystem and git access.

Tools: read_file, write_file, replace_text, ls, mkdir, rm, git, grep, todo_write, ask_user_question

Guidelines:
- Read files before editing
- Use replace_text for edits, write_file for new files
- Confirm destructive ops (rm recursive, force push)
- When ambiguous, ask - don't assume
- Keep responses concise

Todo tracking (use todo_write for multi-step tasks):
- Each todo needs content (imperative), active_form (present continuous), status
- Status: pending, in_progress, completed
- Keep exactly ONE task in_progress at a time
- Mark complete immediately after finishing
</system-instructions>`

type Config struct {
	MemoryFiles []string               `json:"memory_files"`
	MCPServers  []tools.MCPServerConfig `json:"mcp_servers"`
}

// Rule represents a rule file with YAML frontmatter
type Rule struct {
	Pattern string `yaml:"paths,omitempty"` // glob pattern for conditional loading
}

// ruleFile represents a parsed rule file
type ruleFile struct {
	Rule
	SourceFile string // original filename
	Content    string // rule content (after frontmatter)
}

// DefaultMemoryFiles returns the default memory files when no config exists
func DefaultMemoryFiles() []string {
	return []string{"CLAUDE.md"}
}

// wrapXML wraps content in an XML tag with source attribute
func wrapXML(tag, source, content string) string {
	return "<" + tag + " source=\"" + source + "\">\n" + content + "\n</" + tag + ">"
}

// findInAncestors walks up from cwd looking for target, returns full path or ""
func findInAncestors(target string, mustBeDir bool) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, target)
		info, err := os.Stat(path)
		if err == nil && (!mustBeDir || info.IsDir()) {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// FindConfig searches for config file in current dir and parent dirs
func FindConfig() (string, error) {
	path := findInAncestors(filepath.Join(configDirName, configFileName), false)
	if path == "" {
		return "", os.ErrNotExist
	}
	return path, nil
}

// LoadConfig loads the configuration from the config file
// Returns config and the directory containing the config (for path resolution)
func LoadConfig() (*Config, string, error) {
	configPath, err := FindConfig()
	if err != nil {
		// No config file found, return defaults (resolve from cwd)
		return &Config{MemoryFiles: DefaultMemoryFiles()}, "", nil
	}
	configDir := filepath.Dir(filepath.Dir(configPath)) // go up from .simpleagent/config.json

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, "", err
	}

	// Filter out empty entries
	var validFiles []string
	for _, f := range cfg.MemoryFiles {
		if f != "" {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		validFiles = DefaultMemoryFiles()
	}

	cfg.MemoryFiles = validFiles
	return &cfg, configDir, nil
}

// GetMemoryFilesContent reads all configured memory files and concatenates them
// configDir is the directory containing the config file (for resolving relative paths)
func GetMemoryFilesContent(config *Config, configDir string) (string, []string) {
	var contents []string
	var loaded []string

	for _, memFile := range config.MemoryFiles {
		// Resolve relative paths from config directory
		path := memFile
		if configDir != "" && !filepath.IsAbs(memFile) {
			path = filepath.Join(configDir, memFile)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content != "" {
			contents = append(contents, wrapXML("memory", memFile, content))
			loaded = append(loaded, memFile)
		}
	}

	return strings.Join(contents, "\n\n"), loaded
}

// BuildSystemPrompt loads config, memory files, and returns the full system prompt
func BuildSystemPrompt() (string, []string, error) {
	config, configDir, err := LoadConfig()
	if err != nil {
		return systemInstructions, nil, err
	}

	memoryContent, loadedFiles := GetMemoryFilesContent(config, configDir)

	// Load rules from .rules directory
	_, rules := LoadRules()
	alwaysRules := formatAlwaysRules(rules)

	var loaded []string
	if len(loadedFiles) > 0 {
		loaded = append(loaded, loadedFiles...)
	}
	// Only report always-loaded rules (no pattern)
	for _, r := range rules {
		if r.Pattern == "" {
			loaded = append(loaded, r.SourceFile)
		}
	}

	prompt := systemInstructions
	if memoryContent != "" {
		prompt += "\n\n" + memoryContent
	}
	if alwaysRules != "" {
		prompt += "\n\n" + alwaysRules
	}

	// Load and inject skills
	skillsMeta := LoadSkillsMeta()
	skillsSection := FormatSkillsSection(skillsMeta)
	if skillsSection != "" {
		prompt += "\n\n" + skillsSection
	}

	// Store rules globally for conditional matching
	globalRules = rules

	return prompt, loaded, nil
}

// ParseFrontmatter extracts YAML frontmatter and body from file content
func ParseFrontmatter(content string) (frontmatter, body string) {
	lines := strings.Split(content, "\n")
	var fmLines []string
	bodyStart := 0

	// Check for opening ---
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		bodyStart = 1
		// Collect frontmatter until closing ---
		for i := bodyStart; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				bodyStart = i + 1
				break
			}
			fmLines = append(fmLines, lines[i])
		}
	}

	frontmatter = strings.Join(fmLines, "\n")
	body = strings.Join(lines[bodyStart:], "\n")
	return
}

// LoadRules loads all rule files from .rules and .claude/rules directories
func LoadRules() (string, []ruleFile) {
	var rules []ruleFile
	var primaryDir string

	// Check both .rules and .claude/rules
	for _, dirName := range []string{rulesDirName, ".claude/rules"} {
		rulesDir := findDir(dirName)
		if rulesDir == "" {
			continue
		}
		if primaryDir == "" {
			primaryDir = rulesDir
		}

		entries, err := os.ReadDir(rulesDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Skip non-markdown/text files
			if !strings.HasSuffix(entry.Name(), ".md") && !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(rulesDir, entry.Name()))
			if err != nil {
				continue
			}

			fm, body := ParseFrontmatter(string(data))
			if body == "" {
				continue
			}

			var rule Rule
			if fm != "" {
				yaml.Unmarshal([]byte(fm), &rule)
			}

			rules = append(rules, ruleFile{
				Rule:       rule,
				SourceFile: dirName + "/" + entry.Name(),
				Content:    strings.TrimSpace(body),
			})
		}
	}

	return primaryDir, rules
}

// findDir searches for a directory in current dir and parent dirs
func findDir(dirName string) string {
	return findInAncestors(dirName, true)
}

// formatAlwaysRules formats rules without patterns into a single string
func formatAlwaysRules(rules []ruleFile) string {
	var always []string
	for _, r := range rules {
		if r.Pattern == "" {
			always = append(always, wrapXML("rule", r.SourceFile, r.Content))
		}
	}
	return strings.Join(always, "\n\n")
}

// GetMatchingRules returns rules that match the given file path
// Returns (rule content, matched rule sources)
func GetMatchingRules(filePath string) (string, []string) {
	if len(globalRules) == 0 {
		return "", nil
	}

	// Normalize path (./main.go → main.go) then convert to relative
	relPath := filepath.Clean(filePath)
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, relPath); err == nil {
			relPath = rel
		}
	}

	var matched []string
	var sources []string
	for _, r := range globalRules {
		if r.Pattern == "" {
			continue // skip always-loaded rules
		}
		if match, _ := doublestar.Match(r.Pattern, relPath); match {
			matched = append(matched, wrapXML("rule", r.SourceFile, r.Content))
			sources = append(sources, r.SourceFile)
		}
	}
	return strings.Join(matched, "\n\n"), sources
}

// Global state for rules (set during BuildSystemPrompt)
var globalRules []ruleFile
