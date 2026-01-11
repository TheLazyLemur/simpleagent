package claude

import (
	"encoding/json"
	"net/http"
)

// Message types
type Message struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	Role             string         `json:"role"`
	Content          []ContentBlock `json:"content"`
	Model            string         `json:"model"`
	StopReason       string         `json:"stop_reason"`
	Usage            *Usage         `json:"usage,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"` // GLM-4.7
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

type MessageParam struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// Tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Items       *Property           `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
}

// ThinkingConfig controls thinking/reasoning mode
type ThinkingConfig struct {
	Type string `json:"type"` // "enabled" or "disabled"
}

// MessageCreateParams for API request
type MessageCreateParams struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []MessageParam  `json:"messages"`
	System    string          `json:"system,omitempty"`
	Tools     []Tool          `json:"tools,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
	Thinking  *ThinkingConfig `json:"thinking,omitempty"`
}

// MessageResponse wraps Message with raw response access
type MessageResponse struct {
	*Message
	Response *http.Response
}
