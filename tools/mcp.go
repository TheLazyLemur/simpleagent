package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"simpleagent/claude"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpServer struct {
	name   string
	client *client.Client
	tools  []mcp.Tool
}

// MCPClients manages connections to MCP servers
type MCPClients struct {
	servers []*mcpServer
}

// MCPServerConfig defines an MCP server to connect to
type MCPServerConfig struct {
	Name string   `json:"name"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

// NewMCPClients connects to all configured MCP servers
func NewMCPClients(ctx context.Context, configs []MCPServerConfig) *MCPClients {
	mc := &MCPClients{}

	for _, cfg := range configs {
		srv, err := connectMCPServer(ctx, cfg.Name, cfg.Cmd, cfg.Args)
		if err != nil {
			fmt.Printf("[mcp:%s] failed: %v\n", cfg.Name, err)
			continue
		}
		fmt.Printf("[mcp:%s] connected, %d tools\n", cfg.Name, len(srv.tools))
		mc.servers = append(mc.servers, srv)
	}

	return mc
}

func connectMCPServer(ctx context.Context, name, cmd string, args []string) (*mcpServer, error) {
	c, err := client.NewStdioMCPClient(cmd, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "simpleagent",
				Version: "0.1.0",
			},
		},
	})
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return &mcpServer{name: name, client: c, tools: result.Tools}, nil
}

// Tools returns all MCP tools in Claude format
func (mc *MCPClients) Tools() []claude.Tool {
	var tools []claude.Tool
	for _, srv := range mc.servers {
		for _, t := range srv.tools {
			tools = append(tools, claude.Tool{
				Name:        srv.name + "__" + t.Name,
				Description: t.Description,
				InputSchema: convertMCPSchema(t.InputSchema),
			})
		}
	}
	return tools
}

// Execute finds and calls the tool on the right server
func (mc *MCPClients) Execute(ctx context.Context, name string, input json.RawMessage) (string, bool) {
	for _, srv := range mc.servers {
		for _, t := range srv.tools {
			// Handle prefixed name (format: "server__tool")
			rawName := strings.TrimPrefix(name, srv.name+"__")
			if t.Name == rawName {
				return executeMCPTool(ctx, srv, rawName, input), true
			}
		}
	}
	return "", false
}

func executeMCPTool(ctx context.Context, srv *mcpServer, name string, input json.RawMessage) string {
	var args map[string]any
	if len(input) > 0 {
		json.Unmarshal(input, &args)
	}

	result, err := srv.client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	var output string
	for _, c := range result.Content {
		if text, ok := c.(mcp.TextContent); ok {
			output += text.Text
		}
	}
	return output
}

// Close shuts down all MCP connections
func (mc *MCPClients) Close() {
	for _, srv := range mc.servers {
		srv.client.Close()
	}
}

func convertMCPSchema(schema mcp.ToolInputSchema) claude.InputSchema {
	props := make(map[string]claude.Property)
	for name, p := range schema.Properties {
		props[name] = convertMCPProperty(p)
	}
	return claude.InputSchema{
		Type:       "object",
		Properties: props,
		Required:   schema.Required,
	}
}

func convertMCPProperty(p any) claude.Property {
	m, ok := p.(map[string]any)
	if !ok {
		return claude.Property{Type: "string"}
	}

	prop := claude.Property{}
	if t, ok := m["type"].(string); ok {
		prop.Type = t
	}
	if d, ok := m["description"].(string); ok {
		prop.Description = d
	}
	return prop
}
