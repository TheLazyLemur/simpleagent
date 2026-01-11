package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"simpleagent/claude"
)

// RuleMatcher is a function that returns matching rules for a file path
// Returns (rule content, matched rule sources)
var RuleMatcher func(filePath string) (string, []string)

func init() {
	register(claude.Tool{
		Name:        "ReadFile",
		Description: "Read contents of a file",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path":       {Type: "string", Description: "File path to read"},
				"start_line": {Type: "integer", Description: "Start line number (1-based, optional)"},
				"end_line":   {Type: "integer", Description: "End line number (1-based, optional)"},
			},
			Required: []string{"path"},
		},
	}, readFile)

	register(claude.Tool{
		Name:        "WriteFile",
		Description: "Create a new file or overwrite an existing file",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path":    {Type: "string", Description: "File path to write to"},
				"content": {Type: "string", Description: "Content to write to the file"},
			},
			Required: []string{"path", "content"},
		},
	}, writeFile)

	register(claude.Tool{
		Name:        "Ls",
		Description: "Run ls on a dir, defaults to ./",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path": {Type: "string", Description: "Directory path to read"},
			},
			Required: []string{"path"},
		},
	}, ls)

	register(claude.Tool{
		Name:        "Mkdir",
		Description: "Create a directory (and parent directories if needed)",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path": {Type: "string", Description: "Directory path to create"},
			},
			Required: []string{"path"},
		},
	}, mkdir)

	register(claude.Tool{
		Name:        "Rm",
		Description: "Remove a file or directory",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"path":      {Type: "string", Description: "File or directory path to remove"},
				"recursive": {Type: "boolean", Description: "Remove directories and their contents recursively"},
			},
			Required: []string{"path"},
		},
	}, rm)
}

func readFile(input json.RawMessage) Result {
	var args struct {
		Path      string `json:"path"`
		StartLine *int   `json:"start_line"`
		EndLine   *int   `json:"end_line"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("ReadFile", Error(fmt.Sprintf("invalid input: %v", err)))
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return newResult("ReadFile", Error(err.Error()))
	}
	content := string(data)
	if args.StartLine != nil || args.EndLine != nil {
		lines := strings.Split(content, "\n")
		start := 0
		end := len(lines)
		if args.StartLine != nil {
			start = max(*args.StartLine-1, 0)
		}
		if args.EndLine != nil {
			end = min(*args.EndLine, len(lines))
		}
		if start > end {
			return newResult("ReadFile", Error("start_line cannot be greater than end_line"))
		}
		if start >= len(lines) {
			return newResult("ReadFile", Error("start_line is out of range"))
		}
		content = strings.Join(lines[start:end], "\n")
	}

	// Append matching rules if available
	if RuleMatcher != nil {
		if rules, sources := RuleMatcher(args.Path); rules != "" {
			fmt.Println(Status(fmt.Sprintf("loaded rule(s): %v", sources)))
			content += "\n\n" + rules
		}
	}

	return newResult("ReadFile", content)
}

func writeFile(input json.RawMessage) Result {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("WriteFile", Error(fmt.Sprintf("invalid input: %v", err)))
	}

	// Generate content preview for permission prompt
	preview := formatContentPreview(args.Content)

	// Request permission before writing
	allowed, reason, setAcceptAll := RequestPermissionWithDiff("WriteFile", args.Path, fmt.Sprintf("Write %d bytes", len(args.Content)), preview)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("WriteFile", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return newResult("WriteFile", Error(err.Error()))
	}
	return newResult("WriteFile", fmt.Sprintf("wrote to %s", args.Path))
}

// formatContentPreview creates a preview of content to be written
func formatContentPreview(content string) string {
	return formatLines("+++", "New file content", content, 10)
}

func ls(input json.RawMessage) Result {
	var args struct{ Path string }
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Ls", Error(fmt.Sprintf("invalid input: %v", err)))
	}
	cmd := exec.Command("ls", args.Path)
	out, err := cmd.Output()
	if err != nil {
		return newResult("Ls", Error(err.Error()))
	}
	return newResult("Ls", string(out))
}

func mkdir(input json.RawMessage) Result {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Mkdir", Error(fmt.Sprintf("invalid input: %v", err)))
	}
	if err := os.MkdirAll(args.Path, 0755); err != nil {
		return newResult("Mkdir", Error(err.Error()))
	}
	return newResult("Mkdir", fmt.Sprintf("created directory %s", args.Path))
}

func rm(input json.RawMessage) Result {
	var args struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Rm", Error(fmt.Sprintf("invalid input: %v", err)))
	}

	// Request permission with danger warning
	details := "Remove file"
	if args.Recursive {
		details = "Recursive remove (dangerous)"
	}
	allowed, reason, setAcceptAll := RequestPermission("Rm", args.Path, details)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("Rm", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	var err error
	if args.Recursive {
		err = os.RemoveAll(args.Path)
	} else {
		err = os.Remove(args.Path)
	}
	if err != nil {
		return newResult("Rm", Error(err.Error()))
	}
	return newResult("Rm", fmt.Sprintf("removed %s", args.Path))
}
