package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"simpleagent/claude"
)

var globalMCPClients *MCPClients

// SetMCPClients wires MCP integration
func SetMCPClients(c *MCPClients) {
	globalMCPClients = c
}

// Config holds all tool package configuration
type Config struct {
	MCPClients      *MCPClients
	PermissionsMode string
	RuleMatcher     func(string) (string, []string)
	SkillLoader     func(string) (*SkillInfo, error)
	Subagent        *SubagentConfig
}

// SubagentConfig holds subagent/task tool configuration
type SubagentConfig struct {
	Client       *claude.Client
	Model        string
	SystemPrompt string
}

// Init configures the tools package (full replacement, caller provides complete config)
func Init(cfg Config) {
	globalMCPClients = cfg.MCPClients
	permissionsMode = cfg.PermissionsMode
	if permissionsMode == "" {
		permissionsMode = "prompt"
	}
	RuleMatcher = cfg.RuleMatcher
	SkillLoader = cfg.SkillLoader
	if cfg.Subagent != nil {
		subagentClient = cfg.Subagent.Client
		subagentModel = cfg.Subagent.Model
		subagentSystemPrompt = cfg.Subagent.SystemPrompt
	}
}

// Result is returned by Execute, with tool-specific rendering
type Result interface {
	String() string
	Render()
}

type toolResult struct {
	name   string
	output string
}

func (r toolResult) String() string { return r.output }
func (r toolResult) Render()        { fmt.Printf("\n%s\n", Tool(r.name)) }

// newResult creates a standard tool result
func newResult(name, output string) Result {
	return toolResult{name: name, output: output}
}

// rawResult prints output directly without tool badge (for bash, grep, etc.)
type rawResult struct{ output string }

func (r rawResult) String() string { return r.output }
func (r rawResult) Render()        { fmt.Print(r.output) }

var registry = make(map[string]func(json.RawMessage) Result)
var allTools []claude.Tool
var readOnlyTools = map[string]bool{
	"ReadFile":        true,
	"Ls":              true,
	"Grep":            true,
	"Git":             true, // filtered to read-only ops at execution
	"Glob":            true,
	"AskUserQuestion": true,
	"ExitPlanMode":    true,
}

// Skill tool restrictions
var allowedTools map[string]bool // nil = all allowed

// SetAllowedTools restricts tools during skill execution
func SetAllowedTools(tools []string) {
	if len(tools) == 0 {
		allowedTools = nil
		return
	}
	allowedTools = make(map[string]bool)
	for _, t := range tools {
		allowedTools[t] = true
	}
	allowedTools["InvokeSkill"] = true // always allow
}

// ClearAllowedTools removes skill tool restrictions
func ClearAllowedTools() {
	allowedTools = nil
}

// All returns all registered tool definitions (local + MCP)
func All() []claude.Tool {
	if globalMCPClients == nil {
		return allTools
	}
	return append(allTools, globalMCPClients.Tools()...)
}

// ReadOnly returns tools allowed in plan mode
func ReadOnly() []claude.Tool {
	var tools []claude.Tool
	for _, t := range allTools {
		if readOnlyTools[t.Name] {
			tools = append(tools, t)
		}
	}
	return tools
}

// Execute runs a tool by name (local first, then MCP fallback)
func Execute(name string, input json.RawMessage) Result {
	// Check skill tool restrictions (applies to both local and MCP)
	if allowedTools != nil && !allowedTools[name] {
		return toolResult{name: name, output: "error: tool '" + name + "' not allowed in current skill"}
	}
	if fn, ok := registry[name]; ok {
		return fn(input)
	}
	// MCP fallback
	if globalMCPClients != nil {
		if result, found := globalMCPClients.Execute(context.Background(), name, input); found {
			return newResult(name, result)
		}
	}
	return toolResult{name: name, output: "unknown tool"}
}

func register(t claude.Tool, fn func(json.RawMessage) Result) {
	allTools = append(allTools, t)
	registry[t.Name] = fn
}
