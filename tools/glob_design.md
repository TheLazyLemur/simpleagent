# Glob Tool Design Documentation

## Overview

This document provides comprehensive design documentation for implementing a `glob` tool in the simpleagent codebase. The tool will allow pattern-based file finding, complementing the existing `grep` and `ls` tools.

---

## 1. Pattern Syntax and Matching Behavior

### 1.1 Standard Wildcards

Go's `filepath.Match` implements Unix glob-style patterns:

| Pattern | Meaning |
|---------|---------|
| `*` | Matches any sequence of non-separator characters |
| `?` | Matches any single non-separator character |
| `[abc]` | Matches any single character in the set |
| `[a-z]` | Matches any single character in the range |
| `[!abc]` or `[^abc]` | Matches any single character NOT in the set |

### 1.2 Path Separator Behavior

**Important**: Patterns are matched against the entire path, including separators.

- `*.go` matches `file.go` in current directory
- `**/*.go` or `**/*.txt` matches files recursively (if directory wildcard supported)
- `dir/*.go` matches files in `dir/` subdirectory

### 1.3 Character Classes Details

```go
// Examples:
[abc]      // 'a', 'b', or 'c'
[a-z]      // any lowercase letter
[0-9]      // any digit
[a-zA-Z]   // any letter
[!0-9]     // any non-digit
[]]        // literal ] must be first or after !
[-a]       // literal - must be first or last
```

### 1.4 Brace Expansion (Advanced)

**Note**: Go's standard library does NOT support brace expansion (`{a,b,c}`). This is a bash feature, not standard glob.

If brace expansion is needed, implement manually or use a third-party library like `github.com/gobwas/glob`.

---

## 2. Go Standard Library Functions

### 2.1 `filepath.Match(pattern, name string) (bool, error)`

Direct pattern matching against a single name:

```go
matched, err := filepath.Match("*.go", "main.go")  // true
matched, err := filepath.Match("*.go", "main.rs")  // false
matched, err := filepath.Match("[ABC]*", "abc")    // false (case-sensitive!)
```

**Caveats**:
- Case-sensitive matching (on case-sensitive filesystems)
- Does NOT handle path separators specially - `*` does not cross `/`
- No support for `**` (recursive matching)

### 2.2 `filepath(pattern string) ([]string, error)`

Returns names of files matching the pattern in a single directory:

```go
files, err := filepath.Glob("*.go")      // files in current dir
files, err := filepath.Glob("test/*")    // files in test/
files, err := filepath.Abs("../*.md")    // absolute paths work
```

**Implementation**: Uses `filepath.Match` internally, iterates single directory.

### 2.3 `filepath.WalkDir(root string, fn WalkDirFunc) error`

Recursively walks directory tree (preferred over `filepath.Walk` for performance):

```go
filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
    // Process each path
    return nil
})
```

---

## 3. Tool Registry Integration

### 3.1 Existing Pattern Analysis

Looking at `tools/tools.go` and existing tools:

```go
// Registry pattern from tools.go
var registry = make(map[string]func(json.RawMessage) Result)
var allTools []claude.Tool
var readOnlyTools = map[string]bool{...}

func register(t claude.Tool, fn func(json.RawMessage) Result) {
    allTools = append(allTools, t)
    registry[t.Name] = fn
}
```

### 3.2 Result Interface

All tools return `Result` interface:

```go
type Result interface {
    String() string
    Render()
}
```

Existing implementations:
- `toolResult` - standard implementation (output + render)
- `grepResult` - custom for grep-specific formatting

### 3.3 read_file-only Tools Registration

The glob tool should be read-only (safe for plan mode):

```go
var readOnlyTools = map[string]bool{
    "read_file":         true,
    "ls":                true,
    "grep":              true,
    "git":               true,
    "glob":              true,  // Add this
    // ...
}
```

---

## 4. Proposed API Design

### 4.1 Tool Definition

```go
register(claude.Tool{
    Name:        "glob",
    Description: "Find files matching glob patterns",
    InputSchema: claude.InputSchema{
        Type: "object",
        Properties: map[string]claude.Property{
            "pattern": {
                Type:        "string",
                Description: "Glob pattern (e.g., '**/*.go', '*.txt')",
            },
            "path": {
                Type:        "string",
                Description: "Base path to search from (default: current directory)",
            },
            "type": {
                Type:        "string",
                Description: "Filter by file type: 'file', 'dir', or 'symlink'",
            },
            "hidden": {
                Type:        "boolean",
                Description: "Include hidden files and directories (default: false)",
            },
            "limit": {
                Type:        "integer",
                Description: "Maximum number of results to return (default: 100)",
            },
        },
        Required: []string{"pattern"},
    },
}, glob)
```

### 4.2 Input Parameters

```go
type globInput struct {
    Pattern string `json:"pattern"`
    Path    string `json:"path"`      // optional, default "."
    Type    string `json:"type"`      // optional: "file", "dir", "symlink"
    Hidden  *bool  `json:"hidden"`    // optional, default false
    Limit   *int   `json:"limit"`     // optional, default 100
}
```

### 4.3 Return Type

```go
type globResult struct {
    matches   []string
    truncated bool
    total     int
    limit     int
}

func (r globResult) String() string {
    // Returns formatted output
}

func (r globResult) Render() {
    // Prints to stdout
}
```

---

## 5. Edge Cases

### 5.1 Hidden Files and Directories

| Pattern | Hidden Included? | Behavior |
|---------|------------------|----------|
| `*.go` | No | Matches non-hidden only |
| `.*` | Yes | Matches hidden files in root only |
| `**/.*` | Yes | Recursive hidden files |
| `**/*.go` | No | Excludes hidden paths |

**Decision**: By default, skip hidden files/directories. Use `hidden: true` to include.

### 5.2 Symlinks

| Behavior | Description |
|----------|-------------|
| Follow symlinks | Yes (default) - matches target paths |
| Broken symlinks | Skip with warning or error? |
| Symlink loops | Handle via visited set or OS error |

**Recommendation**: Follow symlinks by default. Skip broken symlinks silently (continue walking).

### 5.3 Directories vs Files

| Type Filter | Behavior |
|-------------|----------|
| `"file"` (default) | Return files only |
| `"dir"` | Return directories only |
| `"symlink"` | Return symlinks only |
| `""` (none) | Return all (files, dirs, symlinks) |

### 5.4 Empty Results

- Return `"No matches found for pattern '*.go'"` (informational, not error)
- Count as success, not failure
- Useful for the AI to know pattern didn't match anything

### 5.5 Permission Errors

- Skip files/directories with permission errors (continue walking)
- Optionally track and report at end: "Warning: some paths skipped due to permissions"

### 5.6 Special Characters in Patterns

| Pattern | Behavior |
|---------|----------|
| `[` without `]` | Parse error |
| `[!]` | Negated empty set = matches nothing |
| `*` at root | May match absolute paths unexpectedly |

**Validation**: Validate pattern format before walking. Return error for invalid patterns.

---

## 6. Performance Considerations

### 6.1 Walk vs Glob Trade-offs

| Approach | Pros | Cons |
|----------|------|------|
| `filepath.Walk` | Simple, recursive, full control | Visits every file even with filters |
| `filepath.WalkDir` | Faster (uses `os.DirEntry` cache) | Same as Walk |
| `exec.Command("find")` | Fastest, most powerful | Platform-specific, security concerns |
| `golang.org/x/tools/fs` | Recursive glob support | External dependency |

### 6.2 Optimization Strategies

1. **Early termination**: Stop walking when `limit` is reached
2. **Pattern analysis**: If simple pattern, use `filepath` methods first
3. **Parallel walking**: Not recommended (output ordering matters)
4. **Caching**: Not applicable (live file system)

### 6.3 Recursive Pattern Handling

Go standard library doesn't support `**`. Options:

1. **Custom implementation**: Replace `**` with `/**/` and walk
2. **Use `github.com/gobwas/glob`**: Supports recursive patterns
3. **Manual walk + match**: Walk all, match with custom function

**Recommendation**: Implement custom `**` handling:
- Replace `**` in pattern with special marker
- Walk directory tree
- Match remaining pattern segments

Example:
```
Pattern: **/*.go
1. Replace ** with §§ (marker)
2. Walk recursively, tracking path depth
3. Match against modified pattern
```

---

## 7. Implementation Details

### 7.1 Core Algorithm

```go
func glob(input json.RawMessage) Result {
    // 1. Parse input
    var args globInput
    json.Unmarshal(input, &args)

    // 2. Set defaults
    if args.Path == "" {
        args.Path = "."
    }
    hidden := args.Hidden != nil && *args.Hidden
    limit := 50
    if args.Limit != nil {
        limit = *args.Limit
    }

    // 3. Validate pattern
    if err := validatePattern(args.Pattern); err != nil {
        return newResult("glob", Error(err.Error()))
    }

    // 4. Walk and collect matches
    var matches []string
    var errorMsgs []string

    err := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            if os.IsPermission(err) {
                errorMsgs = append(errorMsgs, fmt.Sprintf("skipping %s: permission denied", path))
                return nil
            }
            return nil // Skip other errors
        }

        // Skip hidden unless requested
        if !hidden && strings.HasPrefix(filepath.Base(path), ".") {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        // Type filter
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
            }
        }

        // Match pattern (handle ** recursively)
        if matchesPattern(args.Pattern, path) {
            matches = append(matches, path)
            if len(matches) >= limit {
                return filepath.SkipAll // Stop walking
            }
        }

        return nil
    })

    // 5. Build output
    return globResult{
        matches:   matches,
        truncated: len(matches) >= limit,
        total:     len(matches),
        limit:     limit,
    }
}
```

### 7.2 Recursive Pattern Handling

```go
func matchesPattern(pattern, path string) bool {
    // Handle ** (recursive)
    if strings.Contains(pattern, "**") {
        return matchesRecursive(pattern, path)
    }
    // Standard match
    matched, _ := filepath.Match(pattern, filepath.Base(path))
    return matched
}

func matchesRecursive(pattern, path string) bool {
    // Split pattern by ** and match segments
    // This is complex - see implementation notes
}
```

### 7.3 Output Formatting

```go
func (r globResult) String() string {
    var sb strings.Builder

    // Header
    sb.WriteString(fmt.Sprintf("Found %d match%s", r.total, plural(r.total)))

    // Truncation notice
    if r.truncated {
        sb.WriteString(fmt.Sprintf(" (showing %d of 100+)", r.limit))
    }
    sb.WriteString(":\n\n")

    // Matches (one per line)
    for _, m := range r.matches {
        sb.WriteString(m)
        sb.WriteString("\n")
    }

    // Warnings
    if len(r.warnings) > 0 {
        sb.WriteString("\nWarnings:\n")
        for _, w := range r.warnings {
            sb.WriteString("  - ")
            sb.WriteString(w)
            sb.WriteString("\n")
        }
    }

    return sb.String()
}
```

---

## 8. Error Handling

### 8.1 Error Types and Responses

| Error Condition | Response |
|-----------------|----------|
| Invalid pattern | `Error("invalid pattern: ...")` |
| Path doesn't exist | `Error("path not found: ...")` |
| Path not accessible | `Error("cannot access path: ...")` |
| Permission denied | Include in warnings, continue |
| Empty results | Normal output (not error) |

### 8.2 Never Panic

All errors return via `newResult("glob", Error(...))`:

```go
if err != nil {
    return newResult("glob", Error(err.Error()))
}
```

---

## 9. Test Cases

### 9.1 Basic Pattern Matching

| Test | Pattern | Input | Expected |
|------|---------|-------|----------|
| Single file | `*.go` | `main.go` | ✓ Match |
| Multiple files | `*.go` | `main.go, util.go` | ✓ Both |
| No match | `*.rs` | `main.go` | ✗ Empty |
| Nested dir | `test/*.txt` | `test/a.txt` | ✓ Match |

### 9.2 Wildcard Behavior

| Test | Pattern | Input | Result |
|------|---------|-------|--------|
| Single char | `?.go` | `a.go` | ✓ Match |
| Single char | `?.go` | `ab.go` | ✗ No match |
| Char set | `[abc].go` | `a.go` | ✓ Match |
| Char set | `[abc].go` | `d.go` | ✗ No match |
| Char range | `[a-z].go` | `z.go` | ✓ Match |
| Negated set | `[!abc].go` | `d.go` | ✓ Match |
| Negated set | `[!abc].go` | `a.go` | ✗ No match |

### 9.3 Hidden Files

| Test | Hidden param | Pattern | Input | Result |
|------|--------------|---------|-------|--------|
| Skip hidden | default | `*` | `.hidden` | ✗ Skipped |
| Skip hidden | default | `*` | `visible.go` | ✓ Matched |
| Include hidden | `true` | `*` | `.hidden` | ✓ Matched |
| Include hidden | `true` | `.*` | `.hidden` | ✓ Matched |

### 9.4 Type Filtering

| Test | Type | Input | Result |
|------|------|-------|--------|
| Files only | `"file"` | `dir/` | ✗ Skipped |
| Files only | `"file"` | `file.go` | ✓ Matched |
| Dirs only | `"dir"` | `dir/` | ✓ Matched |
| Dirs only | `"dir"` | `file.go` | ✗ Skipped |

### 9.5 Recursive Patterns

| Test | Pattern | Input | Result |
|------|---------|-------|--------|
| Recursive `**` | `**/*.go` | `a.go` | ✓ Match |
| Recursive `**` | `**/*.go` | `sub/b.go` | ✓ Match |
| Recursive `**` | `**/*.go` | `sub/sub/c.go` | ✓ Match |
| Deep recursive | `**/*.txt` | `a/b/c/d.txt` | ✓ Match |

### 9.6 Path Handling

| Test | Pattern | Path | Result |
|------|---------|------|--------|
| Relative path | `*.go` | `./` | Works |
| Subdirectory | `*.go` | `./subdir` | Only subdir |
| Absolute path | `/tmp/*.go` | - | Uses abs path |
| Parent dir | `../*.md` | - | Works |

### 9.7 Edge Cases

| Test | Pattern | Expected |
|------|---------|----------|
| Empty directory | `*` on empty dir | "No matches" |
| Permission denied | pattern + protected dir | Warning, continue |
| Broken symlink | pattern + broken link | Skip or error? |
| Symlink loop | pattern + loop | OS error, handle gracefully |
| Pattern with special chars | `[abc].go` | Works |
| Invalid pattern | `[unclosed` | Error returned |
| Deep nesting | `**/*.go` with many dirs | Truncation at limit |

### 9.8 Performance Tests

| Test | Condition | Expected |
|------|-----------|----------|
| Early termination | limit=10, 100 matches | Stops at 10 |
| Large tree | 10,000 files | Completes in reasonable time |
| Many matches | 500 matches | All returned (if under limit) |

---

## 10. Comparison with Similar Tools

### 10.1 ripgrep (`rg`)

| Feature | ripgrep | Proposed glob |
|---------|---------|---------------|
| Pattern type | Regex | Glob (wildcard) |
| Recursive | Default | With `**` |
| File type filter | `-t` | `type` param |
| Hidden files | `-u` | `hidden` param |
| Symlinks | `-L` | Follow by default |

### 10.2 `fd` (find alternative)

| Feature | fd | Proposed glob |
|---------|-----|---------------|
| Pattern type | Glob (default) or regex | Glob only |
| Case-insensitive | `-i` | Not initially |
| Hidden | `-H` | `hidden` param |
| Type filter | `-t f/d/l` | `type` param |
| Max depth | `-d N` | Not initially |
| Symlink handling | `-L` | Follow by default |

### 10.3 Unix `find`

| Feature | find | Proposed glob |
|---------|------|---------------|
| Pattern | `-name "*.go"` | Native glob |
| Recursive | Default | With `**` |
| Type | `-type f/d` | `type` param |
| Hidden | `-name ".*"` | `hidden` param |
| Depth | `-maxdepth` | Not initially |

---

## 11. Implementation Checklist

- [ ] Create `tools/glob.go`
- [ ] Implement `glob` function with `filepath.WalkDir`
- [ ] Handle `**` recursive patterns
- [ ] Add `type` filter parameter
- [ ] Add `hidden` parameter
- [ ] Add `limit` parameter with default 100
- [ ] Implement result type with truncation notice
- [ ] Register tool in `init()`
- [ ] Add to `readOnlyTools` map
- [ ] Create `tools/glob_test.go`
- [ ] Test all pattern types
- [ ] Test edge cases (hidden, symlinks, permissions)
- [ ] Test empty results
- [ ] Test performance with large directories
- [ ] Update `CLAUDE.md` if needed

---

## 12. Example Usage Scenarios

### 12.1 Find all Go files

```json
{
  "pattern": "*.go"
}
```

### 12.2 Find all test files recursively

```json
{
  "pattern": "**/*_test.go"
}
```

### 12.3 Find all markdown in docs directory

```json
{
  "pattern": "docs/**/*.md"
}
```

### 12.4 Find all directories named "test"

```json
{
  "pattern": "**/test",
  "type": "dir"
}
```

### 12.5 Find all hidden config files

```json
{
  "pattern": ".*",
  "hidden": true
}
```

### 12.6 Find first 10 JSON files

```json
{
  "pattern": "**/*.json",
  "limit": 10
}
```

---

## 13. Summary

This design provides:

1. **Pattern Support**: Standard glob wildcards (`*`, `?`, `[...]`) plus recursive `**`
2. **Filters**: Type filtering (`file`, `dir`, `symlink`) and hidden file inclusion
3. **Safety**: read_file-only operation, permission error handling
4. **Usability**: Default values, truncation notices, clear output format
5. **Compatibility**: Follows existing tool patterns in the codebase

The implementation should be straightforward using Go's standard library, with custom handling for recursive patterns that Go's `filepath.Match` doesn't support natively.
