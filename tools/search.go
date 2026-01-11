package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"simpleagent/claude"
)

func init() {
	register(claude.Tool{
		Name:        "Grep",
		Description: "Search for text patterns in files",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"pattern":   {Type: "string", Description: "Text pattern to search for"},
				"path":      {Type: "string", Description: "Directory to search in (default: current directory)"},
				"type":      {Type: "string", Description: "Filter by file type extension (e.g., 'go', 'py', 'js')"},
				"recursive": {Type: "boolean", Description: "Search subdirectories (default: true)"},
				"context":   {Type: "integer", Description: "Number of context lines before/after match (default: 2)"},
				"limit":     {Type: "integer", Description: "Maximum number of matches to return (default: 50)"},
			},
			Required: []string{"pattern"},
		},
	}, grep)
}

type grepResult struct {
	output string
}

func (r grepResult) String() string { return r.output }
func (r grepResult) Render()        { fmt.Print(r.output) }

func grep(input json.RawMessage) Result {
	var args struct {
		Pattern   string `json:"pattern"`
		Path      string `json:"path"`
		Type      string `json:"type"`
		Recursive *bool  `json:"recursive"`
		Context   *int   `json:"context"`
		Limit     *int   `json:"limit"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Grep", Error(err.Error()))
	}

	// Set defaults
	searchPath := args.Path
	if searchPath == "" {
		searchPath = "."
	}
	recursive := true
	if args.Recursive != nil {
		recursive = *args.Recursive
	}
	context := 2
	if args.Context != nil {
		context = *args.Context
	}
	limit := 50
	if args.Limit != nil {
		limit = *args.Limit
	}

	var matches []string
	totalMatches := 0
	var errorMsgs []string

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories if not recursive
		if info.IsDir() {
			if !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary files based on extension hints
		skipExts := []string{".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".zip", ".tar", ".gz", ".exe", ".so", ".a", ".o"}
		for _, ext := range skipExts {
			if strings.HasSuffix(path, ext) {
				return nil
			}
		}

		// Filter by type if specified
		if args.Type != "" {
			if !strings.HasSuffix(path, "."+args.Type) {
				return nil
			}
		}

		// Skip hidden files and directories
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			return nil
		}

		// Search in file
		fileMatches := searchFile(path, args.Pattern, context)
		for _, m := range fileMatches {
			if m == "" {
				// Separator line, always include if under limit
				if len(matches) < limit {
					matches = append(matches, m)
				}
				continue
			}
			totalMatches++
			if totalMatches <= limit {
				matches = append(matches, m)
			}
		}

		return nil
	})

	if err != nil {
		errorMsgs = append(errorMsgs, fmt.Sprintf("walk error: %v", err))
	}

	// Build output
	var sb strings.Builder
	if len(errorMsgs) > 0 {
		sb.WriteString(strings.Join(errorMsgs, "\n"))
		sb.WriteString("\n")
	}
	if len(matches) == 0 {
		sb.WriteString("No matches found")
	} else {
		sb.WriteString(strings.Join(matches, "\n"))
		if totalMatches > limit {
			sb.WriteString(fmt.Sprintf("\n\n... %d more matches truncated (showing %d of %d)", totalMatches-limit, limit, totalMatches))
		}
	}

	return grepResult{output: sb.String()}
}

func searchFile(path, pattern string, context int) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var lines []string

	// Read all lines first for context
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		if strings.Contains(line, pattern) {
			// Calculate context range
			start := max(0, len(lines)-1-context)
			end := len(lines)

			for i := start; i < end; i++ {
				lineContent := lines[i]
				displayLineNum := lineNum - (end - i - 1)
				if i == end-1 {
					// This is the matching line, highlight it
					highlighted := highlightMatch(lineContent, pattern)
					matches = append(matches, fmt.Sprintf("%s:%d:%s", path, displayLineNum, highlighted))
				} else {
					matches = append(matches, fmt.Sprintf("%s:%d: %s", path, displayLineNum, lineContent))
				}
			}
			// Add separator between matches
			matches = append(matches, "")
		}
	}

	return matches
}

func highlightMatch(line, pattern string) string {
	// Simple case-insensitive highlight
	idx := strings.Index(strings.ToLower(line), strings.ToLower(pattern))
	if idx == -1 {
		return line
	}

	return line[:idx] + "[" + line[idx:idx+len(pattern)] + "]" + line[idx+len(pattern):]
}
