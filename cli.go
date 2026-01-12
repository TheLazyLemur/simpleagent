package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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
	reader := bufio.NewReader(os.Stdin)
	var sessionID string
	var sess *SessionFile

	sessionModel := ""
	sessionProvider := ""

	// Load or create session
	if *resumeFlag != "" {
		var err error
		sess, err = loadSession(*resumeFlag)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}
		sessionTodos = sess.Todos
		sessionID = sess.Meta.ID
		if sess.Meta.Model != "" {
			sessionModel = sess.Meta.Model
		}
		if sess.Meta.Provider != "" {
			sessionProvider = sess.Meta.Provider
		}
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
	configErr := err != nil
	if err != nil {
		fmt.Println(tools.Error(fmt.Sprintf("loading config: %v", err)))
	}
	if config == nil {
		config = &Config{}
	}
	if config.Provider == "" || config.Model == "" {
		if err := applyProviderDefaults(config); err != nil {
			return err
		}
	}
	if sessionProvider == "" {
		sessionProvider = config.Provider
	}
	if sessionModel == "" {
		sessionModel = config.Model
	}
	providerDiff := sessionProvider != "" && sessionProvider != config.Provider
	modelDiff := sessionModel != "" && sessionModel != config.Model
	if !configErr && sess != nil && (providerDiff || modelDiff) {
		var mismatchLabel string
		switch {
		case providerDiff && modelDiff:
			mismatchLabel = "provider/model"
		case providerDiff:
			mismatchLabel = "provider"
		case modelDiff:
			mismatchLabel = "model"
		}
		fmt.Printf("%s session %s differs: session %s/%s, config %s/%s. switch to config? [y/N]: ", tools.Prompt(), mismatchLabel, sessionProvider, sessionModel, config.Provider, config.Model)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(tools.Error(fmt.Sprintf("reading prompt: %v", err)))
			response = ""
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			sessionProvider = config.Provider
			sessionModel = config.Model
		}
	}
	providerDefaults, ok := ProviderDefaultsByName[sessionProvider]
	if !ok {
		return fmt.Errorf("invalid provider %q", sessionProvider)
	}
	if sessionModel == "" {
		sessionModel = providerDefaults.DefaultModel
	}
	modelAllowed := false
	for _, model := range providerDefaults.Models {
		if model == sessionModel {
			modelAllowed = true
			break
		}
	}
	if !modelAllowed {
		return fmt.Errorf("invalid model %q for provider %q", sessionModel, sessionProvider)
	}
	client := claude.NewClient(claude.WithBaseURL(providerDefaults.BaseURL))

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
			Model:        sessionModel,
			SystemPrompt: systemPrompt,
		},
	})

	// Create agent
	agent, err := NewAgent(sessionID, sess, client, reader, systemPrompt, sessionProvider, sessionModel, mcpClients, &sessionTodos)
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
