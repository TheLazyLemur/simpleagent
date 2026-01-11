package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDoneTool_ReturnsSignal(t *testing.T) {
	// given
	input := json.RawMessage(`{"summary": "found 3 bugs in auth module"}`)

	// when
	result := Execute("done", input)

	// then
	if !strings.HasPrefix(result.String(), DoneSignalPrefix) {
		t.Errorf("expected prefix %s, got: %s", DoneSignalPrefix, result.String())
	}
	expected := DoneSignalPrefix + "found 3 bugs in auth module"
	if result.String() != expected {
		t.Errorf("expected %q, got: %q", expected, result.String())
	}
}

func TestDoneTool_TruncatesLongSummary(t *testing.T) {
	// given
	longSummary := strings.Repeat("x", 600)
	input, _ := json.Marshal(map[string]string{"summary": longSummary})

	// when
	result := Execute("done", json.RawMessage(input))

	// then
	summary := strings.TrimPrefix(result.String(), DoneSignalPrefix)
	if len(summary) != 500 {
		t.Errorf("expected summary length 500, got: %d", len(summary))
	}
}

func TestDoneTool_NotInAllTools(t *testing.T) {
	// given/when
	allTools := All()

	// then - done should not be in main tool list (subagent only)
	for _, tool := range allTools {
		if tool.Name == "done" {
			t.Error("done tool should not be in All() - it's subagent-only")
		}
	}
}

func TestDoneTool_ExecutorRegistered(t *testing.T) {
	// given/when
	result := Execute("done", json.RawMessage(`{"summary": "test"}`))

	// then - should execute, not return "unknown tool"
	if result.String() == "unknown tool" {
		t.Error("done executor should be registered")
	}
}

func TestDoneTool_EmptySummary(t *testing.T) {
	// given
	input := json.RawMessage(`{"summary": ""}`)

	// when
	result := Execute("done", input)

	// then - should return signal with empty summary
	expected := DoneSignalPrefix
	if result.String() != expected {
		t.Errorf("expected %q, got: %q", expected, result.String())
	}
}

func TestDoneTool_InvalidJSON(t *testing.T) {
	// given
	input := json.RawMessage(`{invalid}`)

	// when
	result := Execute("done", input)

	// then - should contain error message
	if !strings.Contains(result.String(), "error") {
		t.Errorf("expected error in output, got: %s", result.String())
	}
}
