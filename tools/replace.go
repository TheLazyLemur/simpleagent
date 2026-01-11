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
		Name:        "replace_text",
		Description: "Replace text with new text",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path":     {Type: "string", Description: "File to perform replace on"},
				"old_text": {Type: "string", Description: "Old text to replace"},
				"new_text": {Type: "string", Description: "New text to be swapped in"},
			},
			Required: []string{"path", "old_text", "new_text"},
		},
	}, replaceText)
}

func replaceText(input json.RawMessage) Result {
	var args struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	json.Unmarshal(input, &args)

	// Read file first to get actual content for comparison
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return newResult("replace_text", Error(fmt.Sprintf("reading: %v", err)))
	}
	content := string(data)
	updated := strings.Replace(content, args.OldText, args.NewText, 1)
	if content == updated {
		return newResult("replace_text", Error("old_text not found"))
	}

	// Generate diff from old/new text (not file content) for permission prompt
	diff := formatDiff(args.OldText, args.NewText)

	// Request permission before replacing
	allowed, reason, setAcceptAll := RequestPermissionWithDiff("replace_text", args.Path, "Replace text in file", diff)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("replace_text", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	if err := os.WriteFile(args.Path, []byte(updated), 0644); err != nil {
		return newResult("replace_text", Error(fmt.Sprintf("writing: %v", err)))
	}
	return newResult("replace_text", "replaced")
}

// formatDiff creates a unified diff-like view of the change
func formatDiff(oldText, newText string) string {
	// Escape and truncate for display
	oldLines := strings.Split(strings.ReplaceAll(oldText, "\r\n", "\n"), "\n")
	newLines := strings.Split(strings.ReplaceAll(newText, "\r\n", "\n"), "\n")

	oldTruncated := truncateLines(oldLines, 5)
	newTruncated := truncateLines(newLines, 5)

	var b strings.Builder
	b.WriteString("--- Removed (")
	b.WriteString(fmt.Sprintf("%d lines", len(oldLines)))
	b.WriteString(")\n")
	for _, line := range oldTruncated {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(oldLines) > 5 {
		b.WriteString(fmt.Sprintf("- ... and %d more lines\n", len(oldLines)-5))
	}

	b.WriteString("\n+++ Added (")
	b.WriteString(fmt.Sprintf("%d lines", len(newLines)))
	b.WriteString(")\n")
	for _, line := range newTruncated {
		b.WriteString("+ ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(newLines) > 5 {
		b.WriteString(fmt.Sprintf("+ ... and %d more lines\n", len(newLines)-5))
	}

	return b.String()
}
