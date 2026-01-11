package tools

import (
	"testing"
)

func TestSubagentTools_ContainsExpectedTools(t *testing.T) {
	// given
	expected := map[string]bool{
		"read_file":    true,
		"ls":           true,
		"grep":         true,
		"write_file":   true,
		"replace_text": true,
		"done":         true,
	}

	// when
	actual := make(map[string]bool)
	for _, tool := range SubagentTools {
		actual[tool.Name] = true
	}

	// then
	if len(actual) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(actual))
	}
	for name := range expected {
		if !actual[name] {
			t.Errorf("expected tool %s not found in SubagentTools", name)
		}
	}
}

func TestSubagentTools_DoesNotContainDangerousTools(t *testing.T) {
	// given
	dangerous := []string{"bash", "rm", "mkdir", "git"}

	// when/then
	for _, tool := range SubagentTools {
		for _, d := range dangerous {
			if tool.Name == d {
				t.Errorf("SubagentTools should not contain %s", d)
			}
		}
	}
}

func TestSubagentTools_HasDoneTool(t *testing.T) {
	// given/when
	hasDone := false
	for _, tool := range SubagentTools {
		if tool.Name == "done" {
			hasDone = true
			break
		}
	}

	// then
	if !hasDone {
		t.Error("SubagentTools must include done tool")
	}
}
