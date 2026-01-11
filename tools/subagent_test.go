package tools

import (
	"testing"
)

func TestSubagentTools_ContainsExpectedTools(t *testing.T) {
	// given
	expected := map[string]bool{
		"ReadFile":    true,
		"Ls":          true,
		"Grep":        true,
		"WriteFile":   true,
		"ReplaceText": true,
		"Done":        true,
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
	dangerous := []string{"Bash", "Rm", "Mkdir", "Git"}

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
		if tool.Name == "Done" {
			hasDone = true
			break
		}
	}

	// then
	if !hasDone {
		t.Error("SubagentTools must include done tool")
	}
}
