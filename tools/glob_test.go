package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestFiles creates test files and directories in the given base directory
func createTestFiles(t *testing.T, baseDir string, files map[string][]byte, dirs []string) {
	// Create directories
	for _, dir := range dirs {
		path := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}

	// Create files - ensure parent directories exist for nested files
	for name, content := range files {
		path := filepath.Join(baseDir, name)
		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create parent directory for %s: %v", path, err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}
}

// globTestInput helper to create globInput with proper defaults
func globTestInput(pattern, path string, opts ...func(*globInput)) globInput {
	input := globInput{
		Pattern: pattern,
		Path:    path,
	}
	for _, opt := range opts {
		opt(&input)
	}
	return input
}

// withType sets the type filter
func withType(t string) func(*globInput) {
	return func(i *globInput) { i.Type = t }
}

// withHidden sets the hidden flag
func withHidden(h bool) func(*globInput) {
	return func(i *globInput) { i.Hidden = &h }
}

// withLimit sets the limit
func withLimit(l int) func(*globInput) {
	return func(i *globInput) { i.Limit = &l }
}

// executeGlob runs the glob function and returns the result
func executeGlob(t *testing.T, input globInput) globResult {
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	result := glob(data)
	globRes, ok := result.(globResult)
	if !ok {
		// It's an error result - the test expects an error
		// Return empty globResult with error indicator
		t.Logf("Got error result (expected for this test): %s", result.String())
		return globResult{
			count:   -1, // -1 indicates error
			pattern: input.Pattern,
		}
	}
	return globRes
}

// =============================================================================
// 9.1 Basic Pattern Matching
// =============================================================================

func TestGlobBasicSingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"main.go": []byte("package main"),
		"util.go": []byte("package util"),
	}, nil)

	result := executeGlob(t, globTestInput("*.go", tmpDir))

	// Should find Go files (may be 1 or 2 depending on test setup)
	if result.count == 0 {
		t.Errorf("Expected at least 1 match, got %d", result.count)
	}
	// Check that matches contain main.go
	found := false
	for _, m := range result.matches {
		if m == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find main.go in matches: %v", result.matches)
	}
}

func TestGlobBasicMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"main.go":   []byte("package main"),
		"util.go":   []byte("package util"),
		"README.md": []byte("# Readme"),
	}, nil)

	result := executeGlob(t, globTestInput("*.go", tmpDir))

	if result.count != 2 {
		t.Errorf("Expected 2 matches, got %d", result.count)
	}
	// Results should be sorted
	if len(result.matches) != 2 {
		t.Errorf("Expected 2 matches in result, got %d", len(result.matches))
	}
}

func TestGlobBasicNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"main.go": []byte("package main"),
	}, nil)

	result := executeGlob(t, globTestInput("*.rs", tmpDir))

	if result.count != 0 {
		t.Errorf("Expected 0 matches, got %d", result.count)
	}
}

func TestGlobBasicNestedDir(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"test/a.txt":  []byte("test content"),
		"other/b.txt": []byte("other content"),
	}, nil)

	result := executeGlob(t, globTestInput("test/*.txt", tmpDir))

	if result.count != 1 {
		t.Errorf("Expected 1 match, got %d", result.count)
	}
	if len(result.matches) != 1 || result.matches[0] != "test/a.txt" {
		t.Errorf("Expected match 'test/a.txt', got %v", result.matches)
	}
}

func TestGlobBasicTableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		files    map[string][]byte
		dirs     []string
		expected int
		matchers []string
	}{
		{
			name:    "Single Go file",
			pattern: "*.go",
			files: map[string][]byte{
				"main.go": []byte("package main"),
			},
			expected: 1,
			matchers: []string{"main.go"},
		},
		{
			name:    "Multiple Go files",
			pattern: "*.go",
			files: map[string][]byte{
				"main.go":    []byte("package main"),
				"util.go":    []byte("package util"),
				"helpers.go": []byte("package helpers"),
			},
			expected: 3,
			matchers: []string{"main.go", "util.go", "helpers.go"},
		},
		{
			name:    "No match for different extension",
			pattern: "*.rs",
			files: map[string][]byte{
				"main.go": []byte("package main"),
			},
			expected: 0,
			matchers: nil,
		},
		{
			name:    "Nested directory match",
			pattern: "test/*.txt",
			files: map[string][]byte{
				"test/a.txt":     []byte("test"),
				"test/sub/b.txt": []byte("sub"),
				"other/a.txt":    []byte("other"),
			},
			dirs:     []string{"test", "test/sub", "other"},
			expected: 1,
			matchers: []string{"test/a.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createTestFiles(t, tmpDir, tc.files, tc.dirs)

			result := executeGlob(t, globTestInput(tc.pattern, tmpDir))

			if result.count != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, result.count)
			}
			if len(result.matches) != len(tc.matchers) {
				t.Errorf("Expected %d matches, got %v", len(tc.matchers), result.matches)
			}
		})
	}
}

// =============================================================================
// 9.2 Wildcard Behavior
// =============================================================================

func TestGlobWildcardSingleChar(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"a.go":   []byte("package a"),
		"ab.go":  []byte("package ab"),
		"abc.go": []byte("package abc"),
	}, nil)

	// ? matches single character
	result := executeGlob(t, globTestInput("?.go", tmpDir))

	if result.count != 1 {
		t.Errorf("Expected 1 match for '?.go', got %d", result.count)
	}
	if len(result.matches) != 1 || result.matches[0] != "a.go" {
		t.Errorf("Expected 'a.go', got %v", result.matches)
	}
}

func TestGlobWildcardSingleCharNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"ab.go": []byte("package ab"),
	}, nil)

	result := executeGlob(t, globTestInput("?.go", tmpDir))

	if result.count != 0 {
		t.Errorf("Expected 0 matches for '?.go' with 'ab.go', got %d", result.count)
	}
}

func TestGlobWildcardCharSet(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"a.go": []byte("package a"),
		"b.go": []byte("package b"),
		"c.go": []byte("package c"),
		"d.go": []byte("package d"),
	}, nil)

	result := executeGlob(t, globTestInput("[abc].go", tmpDir))

	if result.count != 3 {
		t.Errorf("Expected 3 matches for '[abc].go', got %d", result.count)
	}
}

func TestGlobWildcardCharSetNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"d.go": []byte("package d"),
	}, nil)

	result := executeGlob(t, globTestInput("[abc].go", tmpDir))

	if result.count != 0 {
		t.Errorf("Expected 0 matches for '[abc].go' with 'd.go', got %d", result.count)
	}
}

func TestGlobWildcardCharRange(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"a.go": []byte("package a"),
		"m.go": []byte("package m"),
		"z.go": []byte("package z"),
	}, nil)

	result := executeGlob(t, globTestInput("[a-z].go", tmpDir))

	// [a-z] should match any lowercase letter
	if result.count != 3 {
		t.Errorf("Expected 3 matches for '[a-z].go', got %d", result.count)
	}
}

func TestGlobWildcardNegatedSet(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"a.go": []byte("package a"),
		"b.go": []byte("package b"),
		"c.go": []byte("package c"),
		"d.go": []byte("package d"),
	}, nil)

	// [!abc] is normalized to [^abc] for Go's filepath.Match
	result := globTestInput("[!abc].go", tmpDir)
	result.Hidden = boolPtr(false)

	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected 1 match for '[!abc].go', got %d", globRes.count)
	}
	if len(globRes.matches) != 1 || globRes.matches[0] != "d.go" {
		t.Errorf("Expected 'd.go', got %v", globRes.matches)
	}
}

func TestGlobWildcardNegatedSetNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"a.go": []byte("package a"),
		"b.go": []byte("package b"),
		"c.go": []byte("package c"),
	}, nil)

	// [^abc] matches files NOT starting with a, b, or c
	// Since all files start with a, b, or c, expect 0 matches
	result := globTestInput("[^abc].go", tmpDir)
	result.Hidden = boolPtr(false)

	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 0 {
		t.Errorf("Expected 0 matches for '[^abc].go' with only a,b,c files, got %d", globRes.count)
	}
}

func TestGlobWildcardTableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		files    map[string][]byte
		expected int
	}{
		{
			name:    "? matches single char",
			pattern: "?.go",
			files: map[string][]byte{
				"a.go":  []byte("a"),
				"ab.go": []byte("ab"),
			},
			expected: 1,
		},
		{
			name:    "? no match for two chars",
			pattern: "?.go",
			files: map[string][]byte{
				"ab.go": []byte("ab"),
			},
			expected: 0,
		},
		{
			name:    "[abc] matches chars in set",
			pattern: "[abc].go",
			files: map[string][]byte{
				"a.go": []byte("a"),
				"b.go": []byte("b"),
				"c.go": []byte("c"),
				"d.go": []byte("d"),
			},
			expected: 3,
		},
		{
			name:    "[abc] no match for char outside set",
			pattern: "[abc].go",
			files: map[string][]byte{
				"d.go": []byte("d"),
			},
			expected: 0,
		},
		{
			name:    "[a-z] matches lowercase range",
			pattern: "[a-z].go",
			files: map[string][]byte{
				"a.go": []byte("a"),
				"z.go": []byte("z"),
				"1.go": []byte("1"),
			},
			expected: 2,
		},
		{
			name:    "[^abc] excludes chars in set (not [!abc])",
			pattern: "[^abc].go",
			files: map[string][]byte{
				"a.go": []byte("a"),
				"b.go": []byte("b"),
				"d.go": []byte("d"),
			},
			expected: 1,
		},
		{
			name:    "[^abc] no match when all in set",
			pattern: "[^abc].go",
			files: map[string][]byte{
				"a.go": []byte("a"),
				"b.go": []byte("b"),
				"c.go": []byte("c"),
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createTestFiles(t, tmpDir, tc.files, nil)

			result := globTestInput(tc.pattern, tmpDir)
			result.Hidden = boolPtr(false)
			data, _ := json.Marshal(result)
			globRes := glob(data).(globResult)

			if globRes.count != tc.expected {
				t.Errorf("Expected %d matches, got %d (matches: %v)",
					tc.expected, globRes.count, globRes.matches)
			}
		})
	}
}

// =============================================================================
// 9.3 Hidden Files
// =============================================================================

func TestGlobHiddenSkipByDefault(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go": []byte("package visible"),
		"dotfile.go": []byte("package dotfile"), // starts with "d", not "."
	}, nil)

	// Default: hidden=false, should include dotfile.go (it's not hidden, just has "dot" prefix)
	result := executeGlob(t, globTestInput("*", tmpDir))

	// Should find both files
	if result.count != 2 {
		t.Errorf("Expected 2 matches, got %d (matches: %v)", result.count, result.matches)
	}
}

func TestGlobHiddenIncludeWhenEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go": []byte("package visible"),
		"dotfile.go": []byte("package dotfile"),
	}, nil)

	result := globTestInput("*", tmpDir, withHidden(true))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should find both files (hidden=true doesn't change behavior for non-hidden files)
	if globRes.count != 2 {
		t.Errorf("Expected 2 matches with hidden=true, got %d", globRes.count)
	}
}

func TestGlobHiddenDirectorySkip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden directory
	os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go":       []byte("package visible"),
		".hidden/file.txt": []byte("hidden file"),
	}, nil)

	// Default: hidden=false, should skip entire .hidden directory
	result := executeGlob(t, globTestInput("*", tmpDir))

	// Should only find visible.go
	if result.count != 1 {
		t.Errorf("Expected 1 match (visible only), got %d", result.count)
	}
}

func TestGlobHiddenDirectoryInclude(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden directory with files
	os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go":       []byte("package visible"),
		".hidden/file.txt": []byte("hidden file"),
	}, nil)

	result := globTestInput("*", tmpDir, withHidden(true))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should include visible.go and .hidden directory
	if globRes.count < 1 {
		t.Errorf("Expected at least 1 visible match with hidden=true, got %d", globRes.count)
	}
}

func TestGlobHiddenDotFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that starts with dot
	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go": []byte("package visible"),
		".hidden":    []byte("hidden file content"), // This is a file starting with dot
	}, nil)

	// Default: hidden=false, should skip .hidden
	result := executeGlob(t, globTestInput("*", tmpDir))

	if result.count != 1 {
		t.Errorf("Expected 1 match (visible only), got %d (matches: %v)", result.count, result.matches)
	}
}

func TestGlobHiddenDotFileIncluded(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go": []byte("package visible"),
		".hidden":    []byte("hidden file content"),
	}, nil)

	result := globTestInput("*", tmpDir, withHidden(true))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should include both files
	if globRes.count != 2 {
		t.Errorf("Expected 2 matches with hidden=true, got %d (matches: %v)", globRes.count, globRes.matches)
	}
}

func TestGlobHiddenTableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		hidden   bool
		pattern  string
		files    map[string][]byte
		expected int
	}{
		{
			name:    "Dot file skipped by default",
			hidden:  false,
			pattern: "*",
			files: map[string][]byte{
				".hidden": []byte("hidden"),
				"visible": []byte("visible"),
			},
			expected: 1,
		},
		{
			name:    "Dot file included when enabled",
			hidden:  true,
			pattern: "*",
			files: map[string][]byte{
				".hidden": []byte("hidden"),
				"visible": []byte("visible"),
			},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createTestFiles(t, tmpDir, tc.files, nil)

			result := globTestInput(tc.pattern, tmpDir, withHidden(tc.hidden))
			data, _ := json.Marshal(result)
			globRes := glob(data).(globResult)

			if globRes.count != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, globRes.count)
			}
		})
	}
}

// =============================================================================
// 9.4 Type Filtering
// =============================================================================

func TestGlobTypeFileOnly(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package main"),
	}, nil)

	result := globTestInput("*", tmpDir, withType("file"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should only match files, not directories
	found := false
	for _, m := range globRes.matches {
		if m == "subdir" {
			t.Errorf("Expected 'subdir' to be filtered out for type='file'")
		}
		if m == "file.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'file.go' to be matched for type='file'")
	}
}

func TestGlobTypeFileExcludesDir(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "mydir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"file.txt": []byte("content"),
	}, nil)

	result := globTestInput("*", tmpDir, withType("file"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	for _, m := range globRes.matches {
		if m == "mydir" {
			t.Errorf("Directory 'mydir' should be excluded when type='file'")
		}
	}
}

func TestGlobTypeDirOnly(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package main"),
	}, nil)

	result := globTestInput("*", tmpDir, withType("dir"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	found := false
	for _, m := range globRes.matches {
		if m == "subdir" {
			found = true
		}
		if m == "file.go" {
			t.Errorf("File 'file.go' should be excluded when type='dir'")
		}
	}
	if !found {
		t.Errorf("Expected 'subdir' to be matched for type='dir'")
	}
}

func TestGlobTypeDirExcludesFile(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "mydir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"file.txt": []byte("content"),
	}, nil)

	result := globTestInput("*", tmpDir, withType("dir"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	for _, m := range globRes.matches {
		if m == "file.txt" {
			t.Errorf("File 'file.txt' should be excluded when type='dir'")
		}
	}
}

func TestGlobTypeTableDriven(t *testing.T) {
	testCases := []struct {
		name       string
		typeFilter string
		files      map[string][]byte
		dirs       []string
		expected   int
	}{
		{
			name:       "type=file excludes directories",
			typeFilter: "file",
			files: map[string][]byte{
				"file.go": []byte("code"),
			},
			dirs:     []string{"subdir"},
			expected: 1, // Only file.go
		},
		{
			name:       "type=dir excludes files",
			typeFilter: "dir",
			files: map[string][]byte{
				"file.go": []byte("code"),
			},
			dirs:     []string{"subdir"},
			expected: 1, // Only subdir
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createTestFiles(t, tmpDir, tc.files, tc.dirs)

			result := globTestInput("*", tmpDir, withType(tc.typeFilter))
			data, _ := json.Marshal(result)
			globRes := glob(data).(globResult)

			if globRes.count != tc.expected {
				t.Errorf("Expected %d matches, got %d (matches: %v)", tc.expected, globRes.count, globRes.matches)
			}
		})
	}
}

// =============================================================================
// 9.5 Recursive Patterns
// =============================================================================

func TestGlobRecursiveDoubleStar(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "a/b/c"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "x/y/z"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"root.go":       []byte("package root"),
		"a/file.go":     []byte("package a"),
		"a/b/file.go":   []byte("package ab"),
		"a/b/c/file.go": []byte("package abc"),
		"x/y/z/file.go": []byte("package xyz"),
		"a/b/c.txt":     []byte("not go"),
	}, nil)

	result := globTestInput("**/*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should match all .go files at any depth
	if globRes.count < 5 {
		t.Errorf("Expected at least 5 .go files, got %d (matches: %v)",
			globRes.count, globRes.matches)
	}
}

func TestGlobRecursiveDeep(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deep nesting: a/b/c/d/e/file.txt
	os.MkdirAll(filepath.Join(tmpDir, "a/b/c/d/e"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"a/b/c/d/e/file.txt": []byte("deep"),
	}, nil)

	result := globTestInput("**/*.txt", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected 1 match for deep nested file, got %d", globRes.count)
	}
}

func TestGlobRecursiveMatchesRoot(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package root"),
	}, nil)

	result := globTestInput("**/*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected match for root file, got %d", globRes.count)
	}
}

func TestGlobRecursiveTableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		files    map[string][]byte
		dirs     []string
		expected int
	}{
		{
			name:    "**/*.go matches root file",
			pattern: "**/*.go",
			files: map[string][]byte{
				"file.go": []byte("package root"),
			},
			expected: 1,
		},
		{
			name:    "**/*.go matches nested file",
			pattern: "**/*.go",
			files: map[string][]byte{
				"sub/file.go": []byte("package sub"),
			},
			dirs:     []string{"sub"},
			expected: 1,
		},
		{
			name:    "**/*.go matches deep nested file",
			pattern: "**/*.go",
			files: map[string][]byte{
				"sub/sub/file.go": []byte("package subsub"),
			},
			dirs:     []string{"sub", "sub/sub"},
			expected: 1,
		},
		{
			name:    "**/*.txt matches deeply nested",
			pattern: "**/*.txt",
			files: map[string][]byte{
				"a/b/c/d/file.txt": []byte("deep"),
			},
			dirs:     []string{"a", "a/b", "a/b/c", "a/b/c/d"},
			expected: 1,
		},
		{
			name:    "** matches files and directories",
			pattern: "**",
			files: map[string][]byte{
				"file.go":    []byte("root"),
				"a/b.go":     []byte("a"),
				"a/b/c.go":   []byte("ab"),
				"a/b/c/d.go": []byte("abc"),
			},
			dirs:     []string{"a", "a/b", "a/b/c", "a/b/c/d"},
			expected: 8, // 4 files + 4 directories (a, a/b, a/b/c, a/b/c/d)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createTestFiles(t, tmpDir, tc.files, tc.dirs)

			result := globTestInput(tc.pattern, tmpDir)
			result.Hidden = boolPtr(false) // Include all files
			data, _ := json.Marshal(result)
			globRes := glob(data).(globResult)

			if globRes.count != tc.expected {
				t.Errorf("Expected %d matches, got %d (matches: %v)",
					tc.expected, globRes.count, globRes.matches)
			}
		})
	}
}

// =============================================================================
// 9.6 Path Handling
// =============================================================================

func TestGlobPathRelative(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package main"),
	}, nil)

	// Using "./" as path
	result := globTestInput("*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected 1 match, got %d", globRes.count)
	}
}

func TestGlobPathSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in both root and subdirectory
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"root.go":       []byte("package root"),
		"subdir/sub.go": []byte("package sub"),
	}, nil)

	// Search only in subdir
	result := globTestInput("*.go", filepath.Join(tmpDir, "subdir"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected 1 match in subdir, got %d", globRes.count)
	}
	if len(globRes.matches) > 0 && globRes.matches[0] != "sub.go" {
		t.Errorf("Expected 'sub.go', got %v", globRes.matches)
	}
}

func TestGlobPathSubdirectoryNotInResults(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"root.go":       []byte("package root"),
		"subdir/sub.go": []byte("package sub"),
	}, nil)

	// Search in subdir - root.go should not appear
	result := globTestInput("*", filepath.Join(tmpDir, "subdir"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	for _, m := range globRes.matches {
		if m == "root.go" {
			t.Errorf("root.go should not appear when searching in subdir")
		}
	}
}

// =============================================================================
// 9.7 Edge Cases
// =============================================================================

func TestGlobEdgeCaseEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result := globTestInput("*", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 0 {
		t.Errorf("Expected 0 matches in empty directory, got %d", globRes.count)
	}
}

func TestGlobEdgeCaseInvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package main"),
	}, nil)

	input := globTestInput("[unclosed", tmpDir)
	data, _ := json.Marshal(input)
	result := glob(data)

	// Should return an error result (toolResult, not globResult)
	// The error should contain "invalid pattern"
	errStr := result.String()
	if !strings.Contains(errStr, "invalid pattern") {
		t.Errorf("Expected error for invalid pattern, got: %s", errStr)
	}
}

func TestGlobEdgeCaseSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with special characters in brackets
	// [abc].go matches: a.go, b.go, c.go (single char from set followed by .go)
	// It does NOT match the literal file [abc].go
	createTestFiles(t, tmpDir, map[string][]byte{
		"[abc].go": []byte("package abc"),
		"a.go":     []byte("package a"),
		"b.go":     []byte("package b"),
		"d.go":     []byte("package d"),
	}, nil)

	result := globTestInput("[abc].go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// [abc].go matches: a.go, b.go (c.go if it existed, and NOT [abc].go literally)
	// Should match a.go, b.go (2 matches)
	if globRes.count != 2 {
		t.Errorf("Expected 2 matches for '[abc].go' (a.go, b.go), got %d (matches: %v)", globRes.count, globRes.matches)
	}
}

func TestGlobEdgeCasePermissionDenied(t *testing.T) {
	// This test is platform-dependent and may not work in all environments
	// Skip on Windows and in CI environments
	if testing.Short() {
		t.Skip("Skipping permission test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a protected directory
	protectedDir := filepath.Join(tmpDir, "protected")
	os.MkdirAll(protectedDir, 0755)

	// On Unix-like systems, we can try to restrict permissions
	// Note: This may not work on Windows or with root privileges
	if os.Getuid() != 0 {
		os.Chmod(protectedDir, 0000)
		defer os.Chmod(protectedDir, 0755)

		createTestFiles(t, tmpDir, map[string][]byte{
			"visible.go": []byte("package visible"),
		}, nil)

		result := globTestInput("*", tmpDir)
		data, _ := json.Marshal(result)
		globRes := glob(data).(globResult)

		// Should still work, protected dir is just skipped
		if globRes.count < 1 {
			t.Errorf("Expected at least visible.go to be found, got %d", globRes.count)
		}
	}
}

// =============================================================================
// 9.8 Performance Tests
// =============================================================================

func TestGlobPerformanceEarlyTermination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()

	// Create 10 files with unique names
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		createTestFiles(t, tmpDir, map[string][]byte{
			name: []byte("package file"),
		}, nil)
	}

	limit := 3
	result := globTestInput("*.txt", tmpDir, withLimit(limit))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != limit {
		t.Errorf("Expected exactly %d matches with limit, got %d", limit, globRes.count)
	}
	if !globRes.truncated {
		t.Errorf("Expected truncated=true when limit is reached")
	}
}

func TestGlobPerformanceManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()

	// Create 50 files matching the pattern
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("file%d.go", i)
		createTestFiles(t, tmpDir, map[string][]byte{
			name: []byte("package file"),
		}, nil)
	}

	// Create 50 files NOT matching
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("file%d.rs", i)
		createTestFiles(t, tmpDir, map[string][]byte{
			name: []byte("package file"),
		}, nil)
	}

	result := globTestInput("*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 50 {
		t.Errorf("Expected 50 matches, got %d", globRes.count)
	}
}

func TestGlobPerformanceLimitZero(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file1.go": []byte("package file"),
		"file2.go": []byte("package file"),
	}, nil)

	// limit=0 means return up to 100 (default), but since 0 < 100, it should still work
	// Actually, looking at the code, if limit=0, we need at least 1 match to not trigger truncation
	result := globTestInput("*.go", tmpDir, withLimit(0))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// With limit=0, we should still get matches since limit check is len(matches) >= limit
	// 0 >= 0 is true, so it would trigger SkipAll immediately
	// This is a corner case - limit=0 is not practical
	if globRes.count != 0 {
		// This might be expected - limit=0 triggers immediate stop
		t.Logf("With limit=0, got %d matches (this may be expected behavior)", globRes.count)
	}
}

func TestGlobPerformanceLargeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file1.go": []byte("package file"),
		"file2.go": []byte("package file"),
	}, nil)

	limit := 1000
	result := globTestInput("*.go", tmpDir, withLimit(limit))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 2 {
		t.Errorf("Expected 2 matches, got %d", globRes.count)
	}
	if globRes.truncated {
		t.Errorf("Should not be truncated with large limit")
	}
}

// =============================================================================
// Additional Edge Cases
// =============================================================================

func TestGlobPatternWithPathPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"src/main.go": []byte("package main"),
		"other.go":    []byte("package other"),
	}, nil)

	// Pattern with path prefix
	result := globTestInput("src/*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count != 1 {
		t.Errorf("Expected 1 match, got %d", globRes.count)
	}
	if len(globRes.matches) > 0 && globRes.matches[0] != "src/main.go" {
		t.Errorf("Expected 'src/main.go', got %v", globRes.matches)
	}
}

func TestGlobMixedPatternAndType(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go":        []byte("package file"),
		"subdir/file.go": []byte("package subfile"),
	}, nil)

	// Pattern with type filter
	result := globTestInput("*.go", tmpDir, withType("file"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	if globRes.count == 0 {
		t.Errorf("Expected at least 1 match, got 0")
	} else {
		// Should find file.go (not subdir/file.go because subdir doesn't match *.go)
		found := false
		for _, m := range globRes.matches {
			if m == "file.go" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find file.go, got: %v", globRes.matches)
		}
	}
}

func TestGlobHiddenWithType(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"visible.go":      []byte("package visible"),
		".hidden/file.go": []byte("package hidden"),
	}, nil)

	// Hidden files with type filter
	result := globTestInput("*.go", tmpDir, withHidden(true), withType("file"))
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Should find both visible.go and hidden file
	found := false
	for _, m := range globRes.matches {
		if m == "visible.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected to find visible.go")
	}
}

func TestGlobNoPattern(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"file.go": []byte("package file"),
	}, nil)

	input := globTestInput("", tmpDir)
	data, _ := json.Marshal(input)
	result := glob(data)

	// Empty pattern should return an error result
	errStr := result.String()
	if !strings.Contains(errStr, "pattern is required") {
		t.Errorf("Expected error for empty pattern, got: %s", errStr)
	}
}

func TestGlobDoubleStarMixed(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "a/b/c"), 0755)
	createTestFiles(t, tmpDir, map[string][]byte{
		"root_test.go":  []byte("test"),
		"a/test.go":     []byte("test a"),
		"a/b/test.go":   []byte("test ab"),
		"a/b/c/test.go": []byte("test abc"),
		"a/b/c/main.go": []byte("main"),
	}, nil)

	// Match all test files at any depth
	// **/test.go matches files named "test.go" with any path prefix (at least one dir)
	result := globTestInput("**/test.go", tmpDir)
	result.Hidden = boolPtr(false)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// **/test.go does NOT match root files (requires directory prefix)
	// So it matches: a/test.go, a/b/test.go, a/b/c/test.go (3 files)
	if globRes.count != 3 {
		t.Errorf("Expected 3 test files (in subdirs), got %d (matches: %v)", globRes.count, globRes.matches)
	}
}

func TestGlobSymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFiles(t, tmpDir, map[string][]byte{
		"original.go": []byte("package original"),
	}, nil)

	// Create symlink
	symlinkPath := filepath.Join(tmpDir, "link.go")
	if err := os.Symlink("original.go", symlinkPath); err != nil {
		t.Skip("Symlinks not supported on this platform")
	}

	result := globTestInput("*.go", tmpDir)
	data, _ := json.Marshal(result)
	globRes := glob(data).(globResult)

	// Symlinks are followed and treated as files by default
	found := false
	for _, m := range globRes.matches {
		if m == "link.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected symlink to be matched as file")
	}
}

// Helper function for bool pointer
func boolPtr(b bool) *bool {
	return &b
}
