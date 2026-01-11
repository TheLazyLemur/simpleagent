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
		Name:        "read_file",
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
		Name:        "write_file",
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
		Name:        "ls",
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
		Name:        "mkdir",
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
		Name:        "rm",
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
	json.Unmarshal(input, &args)
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return newResult("read_file", Error(err.Error()))
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
			return newResult("read_file", Error("start_line cannot be greater than end_line"))
		}
		if start >= len(lines) {
			return newResult("read_file", Error("start_line is out of range"))
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

	return newResult("read_file", content)
}

func writeFile(input json.RawMessage) Result {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	json.Unmarshal(input, &args)

	// Generate content preview for permission prompt
	preview := formatContentPreview(args.Content)

	// Request permission before writing
	allowed, reason, setAcceptAll := RequestPermissionWithDiff("write_file", args.Path, fmt.Sprintf("Write %d bytes", len(args.Content)), preview)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("write_file", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return newResult("write_file", Error(err.Error()))
	}
	return newResult("write_file", fmt.Sprintf("wrote to %s", args.Path))
}

// formatContentPreview creates a preview of content to be written
func formatContentPreview(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	truncated := truncateLines(lines, 10)

	var b strings.Builder
	b.WriteString("+++ New file content (")
	b.WriteString(fmt.Sprintf("%d lines", len(lines)))
	b.WriteString(")\n")
	for _, line := range truncated {
		b.WriteString("+ ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(lines) > 10 {
		b.WriteString(fmt.Sprintf("+ ... and %d more lines\n", len(lines)-10))
	}

	return b.String()
}

func ls(input json.RawMessage) Result {
	var args struct{ Path string }
	json.Unmarshal(input, &args)
	cmd := exec.Command("ls", args.Path)
	out, err := cmd.Output()
	if err != nil {
		return newResult("ls", Error(err.Error()))
	}
	return newResult("ls", string(out))
}

func mkdir(input json.RawMessage) Result {
	var args struct {
		Path string `json:"path"`
	}
	json.Unmarshal(input, &args)
	if err := os.MkdirAll(args.Path, 0755); err != nil {
		return newResult("mkdir", Error(err.Error()))
	}
	return newResult("mkdir", fmt.Sprintf("created directory %s", args.Path))
}

func rm(input json.RawMessage) Result {
	var args struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	json.Unmarshal(input, &args)

	// Request permission with danger warning
	details := "Remove file"
	if args.Recursive {
		details = "Recursive remove (dangerous)"
	}
	allowed, reason, setAcceptAll := RequestPermission("rm", args.Path, details)
	if setAcceptAll {
		SetPermissionsMode("accept_all")
		fmt.Println("\n" + Status("accept-all mode enabled for this session"))
	}
	if !allowed {
		return newResult("rm", Error(fmt.Sprintf("permission denied: %s", reason)))
	}

	var err error
	if args.Recursive {
		err = os.RemoveAll(args.Path)
	} else {
		err = os.Remove(args.Path)
	}
	if err != nil {
		return newResult("rm", Error(err.Error()))
	}
	return newResult("rm", fmt.Sprintf("removed %s", args.Path))
}
