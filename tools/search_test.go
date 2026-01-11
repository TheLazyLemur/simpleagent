package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrep_LimitTruncatesResults(t *testing.T) {
	// given
	// ... a directory with many matching lines
	tmpDir := t.TempDir()
	content := strings.Repeat("match\n", 100) // 100 matching lines
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	// ... we grep with limit=10
	input, _ := json.Marshal(map[string]any{
		"pattern": "match",
		"path":    tmpDir,
		"limit":   10,
		"context": 0,
	})
	result := grep(input)

	// then
	// ... should return only 10 matches with truncation message
	output := result.String()
	matchCount := strings.Count(output, "test.txt:")
	if matchCount != 10 {
		t.Errorf("expected 10 matches, got %d", matchCount)
	}
	if !strings.Contains(output, "truncated") {
		t.Errorf("expected truncation message, got: %s", output)
	}
}

func TestGrep_DefaultLimitApplied(t *testing.T) {
	// given
	// ... a directory with many matching lines (more than default limit)
	tmpDir := t.TempDir()
	content := strings.Repeat("match\n", 200) // 200 matching lines
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	// ... we grep without specifying limit
	input, _ := json.Marshal(map[string]any{
		"pattern": "match",
		"path":    tmpDir,
		"context": 0,
	})
	result := grep(input)

	// then
	// ... should apply default limit (50) and show truncation
	output := result.String()
	matchCount := strings.Count(output, "test.txt:")
	if matchCount != 50 {
		t.Errorf("expected 50 matches (default limit), got %d", matchCount)
	}
	if !strings.Contains(output, "truncated") {
		t.Errorf("expected truncation message")
	}
}

func TestGrep_NoTruncationUnderLimit(t *testing.T) {
	// given
	// ... a directory with few matching lines
	tmpDir := t.TempDir()
	content := "match\nmatch\nmatch\n" // 3 matching lines
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	// ... we grep with limit=10
	input, _ := json.Marshal(map[string]any{
		"pattern": "match",
		"path":    tmpDir,
		"limit":   10,
		"context": 0,
	})
	result := grep(input)

	// then
	// ... should return all 3 matches without truncation message
	output := result.String()
	matchCount := strings.Count(output, "test.txt:")
	if matchCount != 3 {
		t.Errorf("expected 3 matches, got %d", matchCount)
	}
	if strings.Contains(output, "truncated") {
		t.Errorf("should not have truncation message")
	}
}
