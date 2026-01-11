package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"simpleagent/claude"
	"simpleagent/tools"
)

// readMultiLine handles \ continuation and pasted multi-line text
func readMultiLine(r *bufio.Reader) string {
	var lines []string
	for {
		line, _ := r.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		// Check for \ continuation
		if strings.HasSuffix(line, "\\") {
			lines = append(lines, strings.TrimSuffix(line, "\\"))
			fmt.Print("  ")
			continue
		}
		lines = append(lines, line)

		// Check if more data buffered (pasted text)
		time.Sleep(5 * time.Millisecond) // brief pause to let paste buffer fill
		if r.Buffered() == 0 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// runBashQuick runs a command and returns output
func runBashQuick(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(out) + "\n" + tools.Error(err.Error())
	}
	return string(out)
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func makeSkillLoader() func(string) (*tools.SkillInfo, error) {
	return func(name string) (*tools.SkillInfo, error) {
		skill, err := LoadSkill(name)
		if err != nil {
			return nil, err
		}
		return &tools.SkillInfo{
			Name:         skill.Name,
			AllowedTools: skill.AllowedTools,
			Dir:          skill.Dir,
			Content:      skill.Content,
		}, nil
	}
}

var (
	baseURL      = getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.minimax.io/anthropic")
	model        = getEnvOrDefault("ANTHROPIC_MODEL", "MiniMax-M2.1")
	sessionTodos []tools.Todo
)

var mdRenderer *glamour.TermRenderer

func init() {
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		mdRenderer = nil
	}
}

func main() {
	resumeFlag := flag.String("resume", "", "Resume a session by ID")
	listFlag := flag.Bool("sessions", false, "List all sessions")
	deleteFlag := flag.String("delete", "", "Delete a session by ID")
	flag.Parse()

	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		fmt.Println(tools.Error(fmt.Sprintf("creating session dir: %v", err)))
		os.Exit(1)
	}

	switch {
	case *listFlag:
		listSessions()
		return
	case *deleteFlag != "":
		deleteSession(*deleteFlag)
		return
	}

	client := claude.NewClient(claude.WithBaseURL(baseURL))
	reader := bufio.NewReader(os.Stdin)
	var messages []claude.MessageParam
	var sessionID string
	turnsSinceTodoWrite := 0

	var planMode bool
	permissionsMode := "prompt" // default to prompt mode

	if *resumeFlag != "" {
		sess, err := loadSession(*resumeFlag)
		if err != nil {
			fmt.Println(tools.Error(fmt.Sprintf("loading session: %v", err)))
			os.Exit(1)
		}
		messages = sess.Messages
		sessionTodos = sess.Todos
		planMode = sess.PlanMode
		sessionID = sess.Meta.ID
		if sess.PermissionsMode != "" {
			permissionsMode = sess.PermissionsMode
		}
		fmt.Println(tools.Status("resumed") + " " + tools.Dim(sessionID))
		if planMode {
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

	// Load config for MCP servers
	config, _, _ := LoadConfig()
	var mcpClients *tools.MCPClients
	if len(config.MCPServers) > 0 {
		ctx := context.Background()
		mcpClients = tools.NewMCPClients(ctx, config.MCPServers)
		defer mcpClients.Close()
		fmt.Println(tools.Status("mcp") + " " + tools.Dim(fmt.Sprintf("%d server(s)", len(config.MCPServers))))
	}

	systemPrompt, loadedFiles, err := BuildSystemPrompt()
	if err != nil {
		fmt.Println(tools.Error(fmt.Sprintf("loading config: %v", err)))
	}
	if len(loadedFiles) > 0 {
		fmt.Println(tools.Status("loaded") + " " + tools.Dim(fmt.Sprintf("%d memory file(s)", len(loadedFiles))))
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

	for {
		fmt.Print(tools.Prompt())
		input := readMultiLine(reader)
		if input == "" {
			continue
		}

		// Handle commands
		if input == "/plan" {
			planMode = !planMode
			if planMode {
				fmt.Println(tools.Plan("plan mode") + " " + tools.Dim("read-only"))
			} else {
				fmt.Println(tools.Status("plan mode off") + " " + tools.Dim("full access"))
			}
			continue
		}

		// !! - run bash and add to context
		if after, ok := strings.CutPrefix(input, "!!"); ok {
			cmd := after
			out := runBashQuick(cmd)
			fmt.Print(out)
			messages = append(messages, claude.MessageParam{
				Role:    "user",
				Content: fmt.Sprintf("$ %s\n%s", cmd, out),
			})
			continue
		}

		// ! - run bash, output only (not added to context)
		if after, ok := strings.CutPrefix(input, "!"); ok {
			cmd := after
			fmt.Print(runBashQuick(cmd))
			continue
		}

		fmt.Printf("%s %s\n", tools.User(), input)
		messages = append(messages, claude.MessageParam{Role: "user", Content: input})

		for {
			toolSet := tools.All()
			if planMode {
				toolSet = tools.ReadOnly()
			}

			stream := client.Messages.Stream(claude.MessageCreateParams{
				Model:     model,
				MaxTokens: 4096,
				System:    systemPrompt,
				Messages:  messages,
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
				break
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

			messages = append(messages, claude.MessageParam{Role: "assistant", Content: msg.Content})

			var toolResults []claude.ToolResultBlock
			for _, block := range msg.Content {
				if block.Type == "tool_use" {
					result := tools.Execute(block.Name, block.Input)
					result.Render()
					if block.Name == "TodoWrite" {
						turnsSinceTodoWrite = 0
					}
					// Check if user accepted the plan
					if block.Name == "ExitPlanMode" && strings.Contains(result.String(), `"decision":"Accept"`) {
						planMode = false
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
				break
			}

			state := &AgentState{
				PlanMode:            planMode,
				TurnsSinceTodoWrite: turnsSinceTodoWrite,
			}
			if reminders := GetReminders(state); reminders != "" {
				last := &toolResults[len(toolResults)-1]
				last.Content += "\n" + reminders
			}

			messages = append(messages, claude.MessageParam{Role: "user", Content: toolResults})
		}

		if sessionID != "" {
			saveSession(sessionID, messages, sessionTodos, planMode, tools.GetPermissionsMode())
		}
		turnsSinceTodoWrite++
	}
}
