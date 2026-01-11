package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// streamController allows aborting the stream (unexported)
type streamController struct {
	cancel  context.CancelFunc
	aborted bool
}

// MessageStream handles streaming responses with event callbacks
type MessageStream struct {
	service    *MessagesService
	params     MessageCreateParams
	handlers   map[string][]func(any)
	message    *Message
	response   *http.Response
	ctx        context.Context
	controller *streamController
	done       chan error
	doneOnce   sync.Once
	finalText  string
}

// On registers an event handler (matches TS sdk .on('event', cb))
func (ms *MessageStream) On(event string, handler func(any)) *MessageStream {
	ms.handlers[event] = append(ms.handlers[event], handler)
	return ms
}

// Off removes an event handler
func (ms *MessageStream) Off(event string, handler func(any)) *MessageStream {
	handlers := ms.handlers[event]
	targetPtr := fmt.Sprintf("%p", handler)
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == targetPtr {
			ms.handlers[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
	return ms
}

// Once registers a handler that fires only once then auto-removes
func (ms *MessageStream) Once(event string, handler func(any)) *MessageStream {
	var wrapper func(any)
	wrapper = func(v any) {
		ms.Off(event, wrapper)
		handler(v)
	}
	ms.handlers[event] = append(ms.handlers[event], wrapper)
	return ms
}

// Abort cancels the stream
func (ms *MessageStream) Abort() {
	ms.controller.aborted = true
	ms.emit("abort", nil)
	ms.controller.cancel()
}

// Done returns error when stream completes (Go equivalent of Promise<void>)
func (ms *MessageStream) Done() <-chan error {
	return ms.done
}

// FinalText returns accumulated text from all text blocks
func (ms *MessageStream) FinalText() string {
	return ms.finalText
}

// OnText handles text delta events
func (ms *MessageStream) OnText(handler func(string)) *MessageStream {
	return ms.On("text", func(v any) { handler(v.(string)) })
}

// OnThinking handles thinking delta events
func (ms *MessageStream) OnThinking(handler func(string)) *MessageStream {
	return ms.On("thinking", func(v any) { handler(v.(string)) })
}

// OnInputJson handles tool input JSON delta events
func (ms *MessageStream) OnInputJson(handler func(string)) *MessageStream {
	return ms.On("inputJson", func(v any) { handler(v.(string)) })
}

// OnContentBlockStart handles content block start events
func (ms *MessageStream) OnContentBlockStart(handler func(ContentBlock)) *MessageStream {
	return ms.On("contentBlockStart", func(v any) { handler(v.(ContentBlock)) })
}

// OnContentBlockStop handles content block stop events
func (ms *MessageStream) OnContentBlockStop(handler func(ContentBlock)) *MessageStream {
	return ms.On("contentBlockStop", func(v any) { handler(v.(ContentBlock)) })
}

// OnMessage handles final message event
func (ms *MessageStream) OnMessage(handler func(*Message)) *MessageStream {
	return ms.On("message", func(v any) { handler(v.(*Message)) })
}

// OnError handles error events
func (ms *MessageStream) OnError(handler func(error)) *MessageStream {
	return ms.On("error", func(v any) { handler(v.(error)) })
}

// OnContentBlock handles content block events (complete block)
func (ms *MessageStream) OnContentBlock(handler func(ContentBlock)) *MessageStream {
	return ms.On("contentBlock", func(v any) { handler(v.(ContentBlock)) })
}

// OnEnd handles stream completion event
func (ms *MessageStream) OnEnd(handler func()) *MessageStream {
	return ms.On("end", func(v any) { handler() })
}

// OnAbort handles stream abort event
func (ms *MessageStream) OnAbort(handler func()) *MessageStream {
	return ms.On("abort", func(v any) { handler() })
}

func (ms *MessageStream) emit(event string, data any) {
	// Copy handlers slice since Once handlers may modify it during iteration
	handlers := make([]func(any), len(ms.handlers[event]))
	copy(handlers, ms.handlers[event])
	for _, h := range handlers {
		h(data)
	}
}

func (ms *MessageStream) signalDone(err error) {
	ms.doneOnce.Do(func() {
		ms.done <- err
	})
}

// Event represents a stream event
type Event struct {
	Type string
	Data sseEvent
}

// Events returns a channel of raw SSE events (Go equivalent of async iterable)
func (ms *MessageStream) Events() (<-chan Event, error) {
	ch := make(chan Event, 100)

	body, _ := json.Marshal(ms.params)
	req, _ := http.NewRequestWithContext(ms.ctx, "POST", ms.service.client.BaseURL+"/v1/messages", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ms.service.client.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := &http.Client{Timeout: ms.service.client.Timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		close(ch)
		return ch, err
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		close(ch)
		return ch, &APIError{Status: resp.StatusCode, Message: string(respBody)}
	}

	ms.response = resp

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ms.ctx.Done():
				ms.signalDone(ms.ctx.Err())
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var sse sseEvent
			if err := json.Unmarshal([]byte(data), &sse); err != nil {
				continue
			}

			ch <- Event{Type: sse.Type, Data: sse}
		}
		if err := scanner.Err(); err != nil {
			ms.signalDone(err)
			return
		}
		ms.signalDone(nil)
	}()

	return ch, nil
}

// FinalMessage starts streaming and returns the complete message
func (ms *MessageStream) FinalMessage() (*Message, error) {
	body, _ := json.Marshal(ms.params)

	req, _ := http.NewRequestWithContext(ms.ctx, "POST", ms.service.client.BaseURL+"/v1/messages", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ms.service.client.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := &http.Client{Timeout: ms.service.client.Timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		ms.emit("error", err)
		ms.signalDone(err)
		return nil, err
	}
	defer resp.Body.Close()
	ms.response = resp

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		err := &APIError{Status: resp.StatusCode, Message: string(respBody)}
		ms.emit("error", err)
		ms.signalDone(err)
		return nil, err
	}

	ms.message = &Message{Content: []ContentBlock{}}
	var currentBlock *ContentBlock
	var toolInputJSON string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ms.ctx.Done():
			ms.signalDone(ms.ctx.Err())
			return ms.message, ms.ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				ms.message.ID = event.Message.ID
				ms.message.Model = event.Message.Model
				ms.message.Role = event.Message.Role
			}
		case "content_block_start":
			currentBlock = event.ContentBlock
			toolInputJSON = ""
			if currentBlock != nil {
				ms.emit("contentBlockStart", *currentBlock)
			}
		case "content_block_delta":
			if currentBlock != nil {
				if event.Delta.Type == "text_delta" {
					currentBlock.Text += event.Delta.Text
					ms.finalText += event.Delta.Text
					ms.emit("text", event.Delta.Text)
				} else if event.Delta.Type == "input_json_delta" {
					toolInputJSON += event.Delta.PartialJSON
					ms.emit("inputJson", event.Delta.PartialJSON)
				} else if event.Delta.Type == "thinking_delta" {
					currentBlock.Thinking += event.Delta.Thinking
					ms.emit("thinking", event.Delta.Thinking)
				}
			}
		case "content_block_stop":
			if currentBlock != nil {
				if currentBlock.Type == "tool_use" && toolInputJSON != "" {
					currentBlock.Input = json.RawMessage(toolInputJSON)
				}
				ms.emit("contentBlockStop", *currentBlock)
				ms.emit("contentBlock", *currentBlock)
				ms.message.Content = append(ms.message.Content, *currentBlock)
				currentBlock = nil
			}
		case "message_delta":
			if event.Delta.StopReason != "" {
				ms.message.StopReason = event.Delta.StopReason
			}
			if event.Delta.ReasoningContent != "" {
				ms.message.ReasoningContent += event.Delta.ReasoningContent
				ms.emit("thinking", event.Delta.ReasoningContent)
			}
		case "message_stop":
			ms.emit("message", ms.message)
		}
	}

	if err := scanner.Err(); err != nil {
		ms.emit("error", err)
		ms.signalDone(err)
		return nil, err
	}

	ms.emit("end", nil)
	ms.signalDone(nil)
	return ms.message, nil
}

// Response returns the raw HTTP response (call after FinalMessage)
func (ms *MessageStream) Response() *http.Response {
	return ms.response
}

// SSE event types (internal)
type sseEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message *struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Role  string `json:"role"`
	} `json:"message"`
	Delta struct {
		Type             string `json:"type"`
		Text             string `json:"text"`
		PartialJSON      string `json:"partial_json"`
		StopReason       string `json:"stop_reason"`
		Thinking         string `json:"thinking"`
		ReasoningContent string `json:"reasoning_content"` // GLM-4.7
	} `json:"delta"`
	ContentBlock *ContentBlock `json:"content_block"`
}
