package tools

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"simpleagent/claude"
)

// SubagentTools is the limited tool set available to subagents
var SubagentTools []claude.Tool

func init() {
	// Build subagent tools from registered tools + DoneTool
	subagentToolNames := []string{"ReadFile", "Ls", "Grep", "WriteFile", "ReplaceText"}
	for _, t := range allTools {
		if slices.Contains(subagentToolNames, t.Name) {
			SubagentTools = append(SubagentTools, t)
		}
	}
	SubagentTools = append(SubagentTools, DoneTool)
}

// RunSubagent executes a subagent with isolated message history
// Returns summary from done tool or final text response
func RunSubagent(client *claude.Client, model, systemPrompt, prompt string) (string, error) {
	// Note: timeout checked between turns; API calls use client.Timeout (10 min)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	messages := []claude.MessageParam{{Role: "user", Content: prompt}}
	const maxTurns = 100

	for turn := 0; turn < maxTurns; turn++ {
		select {
		case <-ctx.Done():
			return "", errors.New("subagent timeout")
		default:
		}

		msg, err := client.Messages.Create(claude.MessageCreateParams{
			Model:     model,
			MaxTokens: 16384,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     SubagentTools,
		})
		if err != nil {
			return "", fmt.Errorf("api call: %w", err)
		}

		// Append assistant response to history
		messages = append(messages, claude.MessageParam{Role: "assistant", Content: msg.Content})

		// Check stop reason
		if msg.StopReason != "tool_use" {
			// Text response - extract and return
			for _, block := range msg.Content {
				if block.Type == "text" {
					return block.Text, nil
				}
			}
			return "", nil
		}

		// Execute tools and collect results
		var toolResults []claude.ToolResultBlock
		for _, block := range msg.Content {
			if block.Type != "tool_use" {
				continue
			}

			result := Execute(block.Name, block.Input)

			// Check for done signal
			if strings.HasPrefix(result.String(), DoneSignalPrefix) {
				summary := strings.TrimPrefix(result.String(), DoneSignalPrefix)
				return summary, nil
			}

			toolResults = append(toolResults, claude.ToolResultBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   result.String(),
			})
		}

		// Add tool results to messages
		messages = append(messages, claude.MessageParam{Role: "user", Content: toolResults})
	}

	return "", errors.New("max turns exceeded")
}
