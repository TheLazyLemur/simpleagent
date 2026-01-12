package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"simpleagent/claude"
)

func init() {
	register(claude.Tool{
		Name:        "ReplaceText",
		Description: "Replace text with new text. Fails if old_text matches multiple locations - use start_line/end_line to disambiguate.",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path":       {Type: "string", Description: "File to perform replace on"},
				"old_text":   {Type: "string", Description: "Old text to replace"},
				"new_text":   {Type: "string", Description: "New text to be swapped in"},
				"start_line": {Type: "integer", Description: "Optional: start line (1-indexed) to scope replacement"},
				"end_line":   {Type: "integer", Description: "Optional: end line (inclusive) to scope replacement"},
			},
			Required: []string{"path", "old_text", "new_text"},
		},
	}, replaceText)
}

func replaceText(input json.RawMessage) Result {
	var args struct {
		Path      string `json:"path"`
		OldText   string `json:"old_text"`
		NewText   string `json:"new_text"`
		StartLine *int   `json:"start_line"`
		EndLine   *int   `json:"end_line"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("ReplaceText", Error(fmt.Sprintf("invalid input: %v", err)))
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		return newResult("ReplaceText", Error(fmt.Sprintf("reading: %v", err)))
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	// Determine search scope
	startIdx, endIdx := 0, len(lines)
	if args.StartLine != nil {
		if *args.StartLine < 1 || *args.StartLine > len(lines) {
			return newResult("ReplaceText", Error(fmt.Sprintf("start_line %d out of range (1-%d)", *args.StartLine, len(lines))))
		}
		startIdx = *args.StartLine - 1
	}
	if args.EndLine != nil {
		if *args.EndLine < 1 || *args.EndLine > len(lines) {
			return newResult("ReplaceText", Error(fmt.Sprintf("end_line %d out of range (1-%d)", *args.EndLine, len(lines))))
		}
		endIdx = *args.EndLine
	}
	if startIdx >= endIdx {
		return newResult("ReplaceText", Error("start_line must be less than end_line"))
	}

	// Build scoped content
	scopedLines := lines[startIdx:endIdx]
	scopedContent := strings.Join(scopedLines, "\n")

	// Check uniqueness within scope
	count := strings.Count(scopedContent, args.OldText)
	if count == 0 {
		if args.StartLine != nil || args.EndLine != nil {
			return newResult("ReplaceText", Error(fmt.Sprintf("old_text not found in lines %d-%d", startIdx+1, endIdx)))
		}
		return newResult("ReplaceText", Error("old_text not found"))
	}
	if count > 1 {
		// Find line numbers of each occurrence for helpful error
		matchLines := findMatchLines(scopedLines, args.OldText, startIdx+1)
		return newResult("ReplaceText", Error(fmt.Sprintf("old_text found %d times at lines %v - use start_line/end_line to disambiguate", count, matchLines)))
	}

	// Replace within scope
	updatedScoped := strings.Replace(scopedContent, args.OldText, args.NewText, 1)
	var updatedLines []string
	updatedLines = append(updatedLines, lines[:startIdx]...)
	updatedLines = append(updatedLines, strings.Split(updatedScoped, "\n")...)
	updatedLines = append(updatedLines, lines[endIdx:]...)
	updated := strings.Join(updatedLines, "\n")

	// Generate diff from old/new text (not file content) for permission prompt
	diff := formatDiff(args.OldText, args.NewText)

	// Request permission before replacing
	allowed, reason, setAcceptAll := RequestPermissionWithDiff("ReplaceText", args.Path, "Replace text in file", diff)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("ReplaceText", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	if err := os.WriteFile(args.Path, []byte(updated), 0644); err != nil {
		return newResult("ReplaceText", Error(fmt.Sprintf("writing: %v", err)))
	}
	return newResult("ReplaceText", "replaced")
}

// formatDiff creates a unified diff-like view of the change
func formatDiff(oldText, newText string) string {
	return formatLines("---", "Removed", oldText, 5) + "\n" + formatLines("+++", "Added", newText, 5)
}

// findMatchLines returns line numbers (1-indexed) where pattern starts
func findMatchLines(lines []string, pattern string, baseLineNum int) []int {
	var result []int
	joined := strings.Join(lines, "\n")
	idx := 0
	for {
		pos := strings.Index(joined[idx:], pattern)
		if pos == -1 {
			break
		}
		absPos := idx + pos
		lineNum := baseLineNum + strings.Count(joined[:absPos], "\n")
		result = append(result, lineNum)
		idx = absPos + 1
	}
	return result
}
