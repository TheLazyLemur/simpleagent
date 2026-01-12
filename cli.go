package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"simpleagent/claude"
	"simpleagent/tools"
)

// RunCLI handles CLI flag dispatch (list/delete sessions or continue to agent)
// Returns (shouldContinue, error)
func RunCLI(listFlag *bool, deleteFlag, resumeFlag *string) (bool, error) {
	switch {
	case *listFlag:
		listSessions()
		return false, nil
	case *deleteFlag != "":
		deleteSession(*deleteFlag)
		return false, nil
	}
	return true, nil
}

// AgentSession orchestrates agent session (config → MCP → tools → agent loop)
func AgentSession(resumeFlag *string) error {
	client := claude.NewClient(claude.WithBaseURL(baseURL))
	reader := bufio.NewReader(os.Stdin)
	var sessionID string
	var sess *SessionFile

	// Load or create session
	if *resumeFlag != "" {
		var err error
		sess, err = loadSession(*resumeFlag)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}
		sessionTodos = sess.Todos
		sessionID = sess.Meta.ID
		permissionsMode := "prompt"
		if sess.PermissionsMode != "" {
			permissionsMode = sess.PermissionsMode
		}
		fmt.Println(tools.Status("resumed") + " " + tools.Dim(sessionID))
		if sess.PlanMode {
			fmt.Println(tools.Plan("plan mode"))
		}
		if permissionsMode == "accept_all" {
			fmt.Println(tools.Warning("accept-all permissions"))
		}
	} else {
		sessionID = newSessionID()
		fmt.Println(tools.Status("new session") + " " + tools.Dim(sessionID))
	}

	fmt.Println(tools.Dim("ctrl+c to quit"))
	fmt.Println(tools.Separator())

	// Build system prompt and config
	systemPrompt, loadedFiles, config, err := BuildSystemPrompt()
	if err != nil {
		fmt.Println(tools.Error(fmt.Sprintf("loading config: %v", err)))
	}

	// Setup MCP clients
	var mcpClients *tools.MCPClients
	if len(config.MCPServers) > 0 {
		ctx := context.Background()
		mcpClients = tools.NewMCPClients(ctx, config.MCPServers)
		defer mcpClients.Close()
		fmt.Println(tools.Status("mcp") + " " + tools.Dim(fmt.Sprintf("%d server(s)", len(config.MCPServers))))
	}
	if len(loadedFiles) > 0 {
		fmt.Println(tools.Status("loaded") + " " + tools.Dim(fmt.Sprintf("%d memory file(s)", len(loadedFiles))))
	}

	// Initialize tools
	permissionsMode := "prompt"
	if sess != nil && sess.PermissionsMode != "" {
		permissionsMode = sess.PermissionsMode
	}
	tools.Init(tools.Config{
		MCPClients:      mcpClients,
		PermissionsMode: permissionsMode,
		RuleMatcher:     GetMatchingRules,
		SkillLoader:     makeSkillLoader(),
		Todos:           &sessionTodos,
		Subagent: &tools.SubagentConfig{
			Client:       client,
			Model:        model,
			SystemPrompt: systemPrompt,
		},
	})

	// Create agent
	agent, err := NewAgent(sessionID, sess, client, reader, systemPrompt, model, mcpClients, &sessionTodos)
	if err != nil {
		return fmt.Errorf("creating agent: %w", err)
	}

	// Main loop
	for {
		fmt.Print(tools.Prompt())
		input := readMultiLine(reader)
		if input == "" {
			continue
		}

		// Handle input and check if we should infer
		shouldInfer, err := agent.HandleInput(input)
		if err != nil {
			return err
		}

		// Print status for special commands
		if input == "/plan" {
			if agent.planMode {
				fmt.Println(tools.Plan("plan mode") + " " + tools.Dim("read-only"))
			} else {
				fmt.Println(tools.Status("plan mode off") + " " + tools.Dim("full access"))
			}
		} else if input[0] == '!' {
			// Bash commands already executed in HandleInput, output already printed
		}

		if !shouldInfer {
			continue
		}

		// Normal user message - print it
		fmt.Printf("%s %s\n", tools.User(), input)

		// Run inference turn
		if err := agent.RunInferenceTurn(); err != nil {
			fmt.Printf("\n%s\n", tools.Error(err.Error()))
		}

		// Save session
		if err := agent.Save(); err != nil {
			fmt.Println(tools.Error(fmt.Sprintf("save failed: %v", err)))
		}
	}
}
