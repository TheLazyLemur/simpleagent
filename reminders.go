package main

import (
	"strings"

	"simpleagent/tools"
)

// ReminderFunc checks state and returns a reminder string (empty if not applicable)
type ReminderFunc func(state *AgentState) string

var reminders []ReminderFunc

func init() {
	reminders = []ReminderFunc{
		todoReminder,
		planModeReminder,
	}
}

// AgentState holds state needed for reminder checks
type AgentState struct {
	PlanMode            bool
	TurnsSinceTodoWrite int
}

// GetReminders returns all applicable reminders for current state
func GetReminders(state *AgentState) string {
	var parts []string
	for _, fn := range reminders {
		if r := fn(state); r != "" {
			parts = append(parts, r)
		}
	}
	return strings.Join(parts, "\n")
}

func todoReminder(state *AgentState) string {
	if state.TurnsSinceTodoWrite >= 3 && tools.HasPending() {
		return `<system-reminder>
The todo_write tool hasn't been used recently. If working on multi-step tasks, consider updating the todo list to track progress.
</system-reminder>`
	}
	return ""
}

func planModeReminder(state *AgentState) string {
	if state.PlanMode {
		return `<system-reminder>
You are in PLAN MODE. You can only read and explore - no file modifications allowed.
Use read_file, ls, grep, and git (read-only ops) to analyze the codebase.
When your plan is complete, tell the user to exit plan mode with /plan to execute.
</system-reminder>`
	}
	return ""
}
