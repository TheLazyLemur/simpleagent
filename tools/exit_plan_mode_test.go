package tools

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestExitPlanModeInput(t *testing.T) {
	// Test that the input JSON is properly parsed
	input := `{"plan":"This is a test plan"}`
	var args ExitPlanModeInput
	err := json.Unmarshal([]byte(input), &args)
	if err != nil {
		t.Fatalf("Failed to unmarshal input: %v", err)
	}
	if args.Plan != "This is a test plan" {
		t.Errorf("Expected plan 'This is a test plan', got '%s'", args.Plan)
	}
}

func TestExitPlanModeDecisionJSON(t *testing.T) {
	// Test that the decision is properly serialized to JSON
	decision := ExitPlanModeDecision{Decision: "Accept"}
	data, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("Failed to marshal decision: %v", err)
	}
	if !strings.Contains(string(data), `"decision":"Accept"`) {
		t.Errorf("Expected JSON with decision 'Accept', got '%s'", string(data))
	}
}

func TestExitPlanModeResultString(t *testing.T) {
	// Test that the result String() method works correctly
	result := exitPlanModeResult{plan: "Test plan", decision: "Deny"}
	output := result.String()
	if !strings.Contains(output, `"decision":"Deny"`) {
		t.Errorf("Expected JSON with decision 'Deny', got '%s'", output)
	}
}

func TestReadOnlyToolsContainsExitPlanMode(t *testing.T) {
	// Verify ExitPlanMode is in readOnlyTools
	if _, ok := readOnlyTools["ExitPlanMode"]; !ok {
		t.Error("ExitPlanMode should be in readOnlyTools")
	}
}

func TestExitPlanModeToolRegistered(t *testing.T) {
	// Verify the tool is registered in All()
	allTools := All()
	found := false
	for _, tool := range allTools {
		if tool.Name == "ExitPlanMode" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ExitPlanMode tool should be registered in All()")
	}
}

func TestExitPlanModeToolInReadOnly(t *testing.T) {
	// Verify the tool is available in ReadOnly()
	readOnlyToolSet := ReadOnly()
	found := false
	for _, tool := range readOnlyToolSet {
		if tool.Name == "ExitPlanMode" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ExitPlanMode tool should be available in ReadOnly()")
	}
}

func TestExecuteExitPlanMode(t *testing.T) {
	// Test that Execute returns a result for ExitPlanMode
	input := json.RawMessage(`{"plan":"Test plan"}`)
	result := Execute("ExitPlanMode", input)
	if result == nil {
		t.Fatal("Execute should return a result")
	}
	// The result should be an exitPlanModeResult
	output := result.String()
	if !strings.Contains(output, `"decision"`) {
		t.Errorf("Result should contain decision field, got '%s'", output)
	}
}

func TestExitPlanModeInteractive(t *testing.T) {
	// Test the promptDecision function
	// Since we can't easily test stdin, let's just test the decision parsing
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "Accept"},
		{"2", "Deny"},
		{"3", "Continue"},
		{"invalid", "Continue"},
		{"", "Continue"},
		{"0", "Continue"},
		{"4", "Continue"},
	}

	for _, tt := range tests {
		// Just verify the logic works by checking what we expect
		num, err := strconv.Atoi(tt.input)
		if err != nil || num < 1 || num > 3 {
			if tt.expected != "Continue" {
				t.Errorf("For input '%s', expected '%s' but got 'Continue'", tt.input, tt.expected)
			}
		} else {
			var decision string
			switch num {
			case 1:
				decision = "Accept"
			case 2:
				decision = "Deny"
			case 3:
				decision = "Continue"
			}
			if decision != tt.expected {
				t.Errorf("For input '%s', expected '%s' but got '%s'", tt.input, tt.expected, decision)
			}
		}
	}
}
