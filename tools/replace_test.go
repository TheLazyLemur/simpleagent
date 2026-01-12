package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceText_BasicUniqueMatch(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("hello world\nfoo bar\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":     path,
		"old_text": "foo",
		"new_text": "baz",
	})
	result := replaceText(input)

	// then
	if strings.Contains(result.String(), "error") {
		t.Errorf("expected success, got: %s", result.String())
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "baz bar") {
		t.Errorf("expected 'baz bar', got: %s", string(content))
	}
}

func TestReplaceText_MultipleMatchesError(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("foo\nbar\nfoo\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":     path,
		"old_text": "foo",
		"new_text": "baz",
	})
	result := replaceText(input)

	// then
	output := result.String()
	if !strings.Contains(output, "found 2 times") {
		t.Errorf("expected multiple match error, got: %s", output)
	}
	if !strings.Contains(output, "disambiguate") {
		t.Errorf("expected disambiguate hint, got: %s", output)
	}
}

func TestReplaceText_ScopedReplacementWithLineRange(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("foo\nbar\nfoo\nbaz\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when - replace foo only in lines 3-4
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_text":   "foo",
		"new_text":   "replaced",
		"start_line": 3,
		"end_line":   4,
	})
	result := replaceText(input)

	// then
	if strings.Contains(result.String(), "error") {
		t.Errorf("expected success, got: %s", result.String())
	}
	content, _ := os.ReadFile(path)
	lines := strings.Split(string(content), "\n")
	if lines[0] != "foo" {
		t.Errorf("line 1 should remain 'foo', got: %s", lines[0])
	}
	if lines[2] != "replaced" {
		t.Errorf("line 3 should be 'replaced', got: %s", lines[2])
	}
}

func TestReplaceText_StartLineOutOfBounds(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_text":   "line1",
		"new_text":   "x",
		"start_line": 0, // invalid: 1-indexed
	})
	result := replaceText(input)

	// then
	if !strings.Contains(result.String(), "out of range") {
		t.Errorf("expected out of range error, got: %s", result.String())
	}
}

func TestReplaceText_EndLineOutOfBounds(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_text":   "line1",
		"new_text":   "x",
		"end_line":   100, // beyond file
	})
	result := replaceText(input)

	// then
	if !strings.Contains(result.String(), "out of range") {
		t.Errorf("expected out of range error, got: %s", result.String())
	}
}

func TestReplaceText_StartLineGreaterThanEndLine(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_text":   "line",
		"new_text":   "x",
		"start_line": 3,
		"end_line":   1,
	})
	result := replaceText(input)

	// then
	if !strings.Contains(result.String(), "must be less than") {
		t.Errorf("expected start < end error, got: %s", result.String())
	}
}

func TestReplaceText_OldTextNotFound(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("hello world\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":     path,
		"old_text": "nonexistent",
		"new_text": "x",
	})
	result := replaceText(input)

	// then
	if !strings.Contains(result.String(), "not found") {
		t.Errorf("expected not found error, got: %s", result.String())
	}
}

func TestReplaceText_OldTextNotFoundInScope(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("foo\nbar\nbaz\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when - foo is on line 1, but we scope to lines 2-3
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_text":   "foo",
		"new_text":   "x",
		"start_line": 2,
		"end_line":   3,
	})
	result := replaceText(input)

	// then
	output := result.String()
	if !strings.Contains(output, "not found") {
		t.Errorf("expected not found error, got: %s", output)
	}
	if !strings.Contains(output, "lines 2-3") {
		t.Errorf("expected scope mentioned in error, got: %s", output)
	}
}

func TestReplaceText_FileNotFound(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	// when
	input, _ := json.Marshal(map[string]any{
		"path":     "/nonexistent/path/file.txt",
		"old_text": "x",
		"new_text": "y",
	})
	result := replaceText(input)

	// then
	if !strings.Contains(result.String(), "reading") {
		t.Errorf("expected read error, got: %s", result.String())
	}
}

func TestReplaceText_MultilineReplacement(t *testing.T) {
	// given
	SetPermissionsMode("accept_all")
	defer SetPermissionsMode("prompt")

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("start\nold line 1\nold line 2\nend\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	input, _ := json.Marshal(map[string]any{
		"path":     path,
		"old_text": "old line 1\nold line 2",
		"new_text": "new single line",
	})
	result := replaceText(input)

	// then
	if strings.Contains(result.String(), "error") {
		t.Errorf("expected success, got: %s", result.String())
	}
	content, _ := os.ReadFile(path)
	expected := "start\nnew single line\nend\n"
	if string(content) != expected {
		t.Errorf("expected %q, got: %q", expected, string(content))
	}
}
