package main

import (
	"bufio"
	"fmt"
	"strings"

	"simpleagent/claude"
	"simpleagent/tools"
)

// Agent manages a single agent session
type Agent struct {
	// Dependencies
	client     *claude.Client
	mcpClients *tools.MCPClients
	reader     *bufio.Reader

	// Session state
	sessionID           string
	messages            []claude.MessageParam
	todos               *[]tools.Todo
	planMode            bool
	permissionsMode     string
	turnsSinceTodoWrite int

	// Config
	systemPrompt string
	model        string
}

// NewAgent creates an agent from session (resume or new)
func NewAgent(
	sessionID string,
	sess *SessionFile,
	client *claude.Client,
	reader *bufio.Reader,
	systemPrompt string,
	model string,
	mcpClients *tools.MCPClients,
	todos *[]tools.Todo,
) (*Agent, error) {
	agent := &Agent{
		client:          client,
		mcpClients:      mcpClients,
		reader:          reader,
		sessionID:       sessionID,
		todos:           todos,
		systemPrompt:    systemPrompt,
		model:           model,
		permissionsMode: "prompt",
	}

	// Resume from existing session
	if sess != nil {
		agent.messages = sess.Messages
		*agent.todos = sess.Todos
		agent.planMode = sess.PlanMode
		if sess.PermissionsMode != "" {
			agent.permissionsMode = sess.PermissionsMode
		}
	}

	return agent, nil
}

// HandleInput processes user input, returns (shouldInfer bool, error)
// Returns false for commands that don't need inference (/plan, !, !!)
func (a *Agent) HandleInput(input string) (bool, error) {
	// /plan - toggle plan mode
	if input == "/plan" {
		a.planMode = !a.planMode
		// Print omitted in method - caller handles
		return false, nil
	}

	// !! - run bash and add to context
	if after, ok := strings.CutPrefix(input, "!!"); ok {
		cmd := after
		out := runBashQuick(cmd)
		// Print omitted - caller handles
		a.messages = append(a.messages, claude.MessageParam{
			Role:    "user",
			Content: fmt.Sprintf("$ %s\n%s", cmd, out),
		})
		return false, nil
	}

	// ! - run bash, output only (not added to context)
	if after, ok := strings.CutPrefix(input, "!"); ok {
		// Print omitted - caller handles
		_ = runBashQuick(after)
		return false, nil
	}

	// Normal input - append to messages
	a.messages = append(a.messages, claude.MessageParam{Role: "user", Content: input})
	return true, nil
}

// Save persists current session state
func (a *Agent) Save() error {
	if a.sessionID == "" {
		return nil
	}
	if err := saveSession(a.sessionID, a.messages, *a.todos, a.planMode, a.permissionsMode); err != nil {
		return err
	}
	a.turnsSinceTodoWrite++
	return nil
}

// RunInferenceTurn executes one agentic loop iteration
func (a *Agent) RunInferenceTurn() error {
	for {
		toolSet := tools.All()
		if a.planMode {
			toolSet = tools.ReadOnly()
		}

		stream := a.client.Messages.Stream(claude.MessageCreateParams{
			Model:     a.model,
			MaxTokens: 4096,
			System:    a.systemPrompt,
			Messages:  a.messages,
			Tools:     toolSet,
			Thinking:  &claude.ThinkingConfig{Type: "enabled"},
		})

		var textBuffer strings.Builder
		stream.OnText(func(s string) {
			textBuffer.WriteString(s)
		})
		stream.OnThinking(func(s string) {
			fmt.Print(tools.Thinking(s))
		})

		msg, err := stream.FinalMessage()
		if err != nil {
			fmt.Printf("\n%s\n", tools.Error(err.Error()))
			return err
		}
		fmt.Println("")

		// Render collected text with glamour
		if text := textBuffer.String(); text != "" {
			fmt.Printf("%s\n", tools.Agent())
			if mdRenderer != nil {
				rendered, err := mdRenderer.Render(text)
				if err == nil {
					fmt.Print(strings.TrimSpace(rendered))
				} else {
					fmt.Print(text)
				}
			} else {
				fmt.Print(text)
			}
		}

		a.messages = append(a.messages, claude.MessageParam{Role: "assistant", Content: msg.Content})

		var toolResults []claude.ToolResultBlock
		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				result := tools.Execute(block.Name, block.Input)
				result.Render()
				if block.Name == "TodoWrite" {
					a.turnsSinceTodoWrite = 0
				}
				// Check if user accepted the plan
				if block.Name == "ExitPlanMode" && strings.Contains(result.String(), `"decision":"Accept"`) {
					a.planMode = false
					fmt.Println("\n" + tools.Status("plan mode off") + " " + tools.Dim("full access"))
				}
				toolResults = append(toolResults, claude.ToolResultBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   result.String(),
				})
			}
		}

		if len(toolResults) == 0 {
			fmt.Println()
			return nil
		}

		hasPending := false
		for _, t := range *a.todos {
			if t.Status == "pending" || t.Status == "in_progress" {
				hasPending = true
				break
			}
		}
		state := &AgentState{
			PlanMode:            a.planMode,
			TurnsSinceTodoWrite: a.turnsSinceTodoWrite,
			HasPendingTodos:     hasPending,
		}
		if reminders := GetReminders(state); reminders != "" {
			last := &toolResults[len(toolResults)-1]
			last.Content += "\n" + reminders
		}

		a.messages = append(a.messages, claude.MessageParam{Role: "user", Content: toolResults})
	}
}
