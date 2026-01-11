package tools

import (
	"encoding/json"
	"testing"
)

func TestQuestions_ParsesInput(t *testing.T) {
	// given
	// ... a valid questions input
	input, _ := json.Marshal(map[string]any{
		"questions": []map[string]any{
			{
				"question":    "Which approach?",
				"header":      "Approach",
				"multiSelect": false,
				"options": []map[string]any{
					{"label": "Option A", "description": "First option"},
					{"label": "Option B", "description": "Second option"},
				},
			},
		},
	})

	// when
	// ... we parse the input
	var args struct {
		Questions []Question `json:"questions"`
	}
	err := json.Unmarshal(input, &args)

	// then
	// ... should parse correctly
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(args.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(args.Questions))
	}
	q := args.Questions[0]
	if q.Question != "Which approach?" {
		t.Errorf("expected 'Which approach?', got %q", q.Question)
	}
	if q.Header != "Approach" {
		t.Errorf("expected 'Approach', got %q", q.Header)
	}
	if q.MultiSelect != false {
		t.Errorf("expected multiSelect=false")
	}
	if len(q.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(q.Options))
	}
	if q.Options[0].Label != "Option A" {
		t.Errorf("expected 'Option A', got %q", q.Options[0].Label)
	}
}
