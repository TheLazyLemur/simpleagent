package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTaskTool_RequiresConfig(t *testing.T) {
	// given - no config set
	ResetSubagentConfig()
	input := json.RawMessage(`{"prompt": "test", "description": "test"}`)

	// when
	result := Execute("task", input)

	// then - should error about missing config
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error, got: %s", result.String())
	}
}

func TestTaskTool_InvalidJSON(t *testing.T) {
	// given
	input := json.RawMessage(`{invalid}`)

	// when
	result := Execute("task", input)

	// then
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error in output, got: %s", result.String())
	}
}

func TestTaskTool_MissingPrompt(t *testing.T) {
	// given
	input := json.RawMessage(`{"description": "test"}`)

	// when
	result := Execute("task", input)

	// then
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error for missing prompt, got: %s", result.String())
	}
}

func TestTaskTool_MissingDescription(t *testing.T) {
	// given
	input := json.RawMessage(`{"prompt": "test"}`)

	// when
	result := Execute("task", input)

	// then
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error for missing description, got: %s", result.String())
	}
}

func TestTaskTool_InAllTools(t *testing.T) {
	// given/when
	allTools := All()

	// then
	found := false
	for _, tool := range allTools {
		if tool.Name == "task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("task tool should be in All()")
	}
}

func TestTaskTool_HasRequiredSchema(t *testing.T) {
	// given/when
	allTools := All()

	// then
	var found bool
	for _, tool := range allTools {
		if tool.Name == "task" {
			found = true
			if _, ok := tool.InputSchema.Properties["prompt"]; !ok {
				t.Error("task tool missing prompt property")
			}
			if _, ok := tool.InputSchema.Properties["description"]; !ok {
				t.Error("task tool missing description property")
			}
			break
		}
	}
	if !found {
		t.Fatal("task tool not found")
	}
}

func TestSetSubagentConfig(t *testing.T) {
	// given
	ResetSubagentConfig()

	// when
	SetSubagentConfig(nil, "test-model", "test-prompt")

	// then - should not panic, config stored
	// Actual functionality tested via integration tests
}
