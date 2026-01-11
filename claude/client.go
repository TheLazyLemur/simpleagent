package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client configuration
type Client struct {
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
	Messages *MessagesService
}

type ClientOption func(*Client)

func WithAPIKey(key string) ClientOption {
	return func(c *Client) { c.APIKey = key }
}

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.BaseURL = url }
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) { c.Timeout = d }
}

// NewClient creates a new Anthropic API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL: "https://api.anthropic.com",
		Timeout: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Messages = &MessagesService{client: c}
	return c
}

// MessagesService handles message operations
type MessagesService struct {
	client *Client
}

// Create sends a message and returns the response
func (s *MessagesService) Create(params MessageCreateParams) (*Message, error) {
	resp, err := s.CreateWithResponse(params)
	if err != nil {
		return nil, err
	}
	return resp.Message, nil
}

// CreateWithResponse returns both message and raw HTTP response
func (s *MessagesService) CreateWithResponse(params MessageCreateParams) (*MessageResponse, error) {
	params.Stream = false
	body, _ := json.Marshal(params)

	req, _ := http.NewRequest("POST", s.client.BaseURL+"/v1/messages", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.client.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := &http.Client{Timeout: s.client.Timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &APIError{Status: resp.StatusCode, Message: string(respBody)}
	}

	var msg Message
	json.Unmarshal(respBody, &msg)
	return &MessageResponse{Message: &msg, Response: resp}, nil
}

// Stream returns a MessageStream for streaming responses
func (s *MessagesService) Stream(params MessageCreateParams) *MessageStream {
	ctx, cancel := context.WithCancel(context.Background())
	params.Stream = true
	return &MessageStream{
		service:    s,
		params:     params,
		handlers:   make(map[string][]func(any)),
		ctx:        ctx,
		controller: &streamController{cancel: cancel},
		done:       make(chan error, 1),
	}
}

// APIError represents an API error response
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Status, e.Message)
}
