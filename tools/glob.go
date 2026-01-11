package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"simpleagent/claude"
)

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`   // optional, default "."
	Type    string `json:"type"`   // optional: "file", "dir", "symlink"
	Hidden  *bool  `json:"hidden"` // optional, default false
	Limit   *int   `json:"limit"`  // optional, default 100
}

type globResult struct {
	name      string
	pattern   string
	matches   []string
	count     int
	truncated bool
	limit     int
}

func (r globResult) String() string {
	if r.count == 0 {
		return fmt.Sprintf("No matches found for pattern '%s'", r.pattern)
	}
	var sb strings.Builder
	if r.truncated {
		sb.WriteString(fmt.Sprintf("Found %d match%s (limit %d):\n", r.count, plural(r.count), r.limit))
	} else {
		sb.WriteString(fmt.Sprintf("Found %d match%s:\n", r.count, plural(r.count)))
	}
	for _, m := range r.matches {
		sb.WriteString(m)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (r globResult) Render() {
	fmt.Printf("\n%s\n", r)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}

func init() {
	register(claude.Tool{
		Name:        "glob",
		Description: "Find files matching glob patterns",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern (e.g., '*.go', '**/*.txt')",
				},
				"path": {
					Type:        "string",
					Description: "Base path to search from (default: current directory)",
				},
				"type": {
					Type:        "string",
					Description: "Filter by type: 'file', 'dir', 'symlink' (default: all)",
				},
				"hidden": {
					Type:        "boolean",
					Description: "Include hidden files and directories (default: false)",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of results (default: 100)",
				},
			},
			Required: []string{"pattern"},
		},
	}, glob)
}

func glob(input json.RawMessage) Result {
	var args globInput
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("glob", Error(err.Error()))
	}

	if args.Path == "" {
		args.Path = "."
	}

	if args.Pattern == "" {
		return newResult("glob", Error("pattern is required"))
	}

	limit := 100
	if args.Limit != nil && *args.Limit > 0 {
		limit = *args.Limit
	}

	includeHidden := args.Hidden != nil && *args.Hidden

	var matches []string
	truncated := false

	err := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible
		}

		relPath, err := filepath.Rel(args.Path, path)
		if err != nil {
			relPath = path
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Skip hidden unless requested
		if !includeHidden {
			baseName := filepath.Base(relPath)
			if strings.HasPrefix(baseName, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Type filtering
		if args.Type != "" {
			switch args.Type {
			case "file":
				if d.IsDir() {
					return nil
				}
			case "dir":
				if !d.IsDir() {
					return nil
				}
			case "symlink":
				if d.Type()&os.ModeSymlink == 0 {
					return nil
				}
			}
		}

		// Match using doublestar (supports ** and [!...])
		matched, err := doublestar.Match(args.Pattern, relPath)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}

		if matched {
			matches = append(matches, relPath)
			if len(matches) >= limit {
				truncated = true
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil {
		return newResult("glob", Error(err.Error()))
	}

	sort.Strings(matches)

	return globResult{
		name:      "glob",
		pattern:   args.Pattern,
		matches:   matches,
		count:     len(matches),
		truncated: truncated,
		limit:     limit,
	}
}
