package tools

import (
	"encoding/json"
	"fmt"

	"simpleagent/claude"
)

// DoneSignalPrefix is used by RunSubagent to detect completion
const DoneSignalPrefix = "DONE_SIGNAL:"

// DoneTool is available only for subagents to signal completion
var DoneTool = claude.Tool{
	Name:        "Done",
	Description: "Signal task completion and return findings to parent agent. Call when your task is complete.",
	InputSchema: claude.InputSchema{
		Type: "object",
		Properties: map[string]claude.Property{
			"summary": {Type: "string", Description: "Summary of findings (max 500 chars)"},
		},
		Required: []string{"summary"},
	},
}

func init() {
	// Register executor only, not the tool definition (subagent-only)
	registry["Done"] = executeDone
}

func executeDone(input json.RawMessage) Result {
	var args struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Done", Error(err.Error()))
	}

	if len(args.Summary) > 500 {
		args.Summary = args.Summary[:500]
	}

	return newResult("Done", fmt.Sprintf("%s%s", DoneSignalPrefix, args.Summary))
}
