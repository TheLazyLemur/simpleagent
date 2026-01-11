package tools

import (
	"encoding/json"
	"fmt"

	"simpleagent/claude"
)

// Subagent config - set from main before tool is called
var (
	subagentClient       *claude.Client
	subagentModel        string
	subagentSystemPrompt string
)

// SetSubagentConfig stores config for task tool to use
func SetSubagentConfig(client *claude.Client, model, systemPrompt string) {
	subagentClient = client
	subagentModel = model
	subagentSystemPrompt = systemPrompt
}

// ResetSubagentConfig clears config (for testing)
func ResetSubagentConfig() {
	subagentClient = nil
	subagentModel = ""
	subagentSystemPrompt = ""
}

func init() {
	register(claude.Tool{
		Name:        "Task",
		Description: "Spawn a subagent to research a question. Blocks until complete.",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"prompt":      {Type: "string", Description: "Research question for the subagent"},
				"description": {Type: "string", Description: "Short label for the task (shown in UI)"},
			},
			Required: []string{"prompt", "description"},
		},
	}, task)
}

type taskResult struct {
	description string
	output      string
}

func (r taskResult) String() string { return r.output }
func (r taskResult) Render() {
	fmt.Printf("\n%s %s\n", infoBadge.Render("task"), dimStyle.Render(r.description))
}

func task(input json.RawMessage) Result {
	var args struct {
		Prompt      string `json:"prompt"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("Task", Error(err.Error()))
	}

	if args.Prompt == "" {
		return newResult("Task", Error("prompt required"))
	}
	if args.Description == "" {
		return newResult("Task", Error("description required"))
	}

	if subagentClient == nil {
		return newResult("Task", Error("subagent not configured"))
	}

	summary, err := RunSubagent(subagentClient, subagentModel, subagentSystemPrompt, args.Prompt)
	if err != nil {
		return newResult("Task", Error(err.Error()))
	}

	return taskResult{description: args.Description, output: summary}
}
