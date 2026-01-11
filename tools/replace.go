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
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("ReplaceText", Error(fmt.Sprintf("invalid input: %v", err)))
	}

	// Read file first to get actual content for comparison
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return newResult("ReplaceText", Error(fmt.Sprintf("reading: %v", err)))
	}
	content := string(data)
	updated := strings.Replace(content, args.OldText, args.NewText, 1)
	if content == updated {
		return newResult("ReplaceText", Error("old_text not found"))
	}

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
