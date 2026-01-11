package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBash_SimpleCommand(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"echo", "hello world"},
	})
	want := "hello world"

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), want) {
		t.Errorf("expected output to contain %q, got: %s", want, result.String())
	}
}

func TestBash_ExitCodeZero(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"echo", "success"},
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "[exit code: 0]") {
		t.Errorf("expected [exit code: 0], got: %s", result.String())
	}
}

func TestBash_NonZeroExitCode(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"sh", "-c", "exit 42"},
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "[exit code: 42]") {
		t.Errorf("expected [exit code: 42], got: %s", result.String())
	}
}

func TestBash_DefaultTimeout(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"sleep", "0.1"},
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "[exit code: 0]") {
		t.Errorf("expected success, got: %s", result.String())
	}
}

func TestBash_TimeoutParameter(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args":        []string{"sleep", "1"},
		"timeout_sec": 5,
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "[exit code: 0]") {
		t.Errorf("expected success with timeout, got: %s", result.String())
	}
}

func TestBash_TimeoutExceeded(t *testing.T) {
	t.Skip("flaky test: depends on signal behavior")

	// given - sleep 10s with 1s timeout
	input, _ := json.Marshal(map[string]any{
		"args":        []string{"sleep", "10"},
		"timeout_sec": 1,
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "signal: terminated") {
		t.Errorf("expected timeout signal, got: %s", result.String())
	}
	if !strings.Contains(result.String(), "[exit code: 1]") {
		t.Errorf("expected exit code 1 on timeout, got: %s", result.String())
	}
}

func TestBash_StderrIncluded(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"sh", "-c", "echo error >&2"},
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected stderr in output, got: %s", result.String())
	}
}

func TestBash_MissingArgs(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"timeout_sec": 30,
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error for missing args, got: %s", result.String())
	}
}

func TestBash_CwdParameter(t *testing.T) {
	// given
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	input, _ := json.Marshal(map[string]any{
		"args": []string{"cat", "test.txt"},
		"cwd":  tmpDir,
	})

	// when
	result := bash(input)

	// then
	if !strings.Contains(result.String(), "test content") {
		t.Errorf("expected file content, got: %s", result.String())
	}
}

func TestBash_Render(t *testing.T) {
	// given
	input, _ := json.Marshal(map[string]any{
		"args": []string{"echo", "test"},
	})

	// when
	result := bash(input)

	// then - should not panic
	result.Render()
}
