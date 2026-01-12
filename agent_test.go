package main

import (
	"bufio"
	"strings"
	"testing"

	"simpleagent/claude"
	"simpleagent/tools"
)

func TestNewAgent_CreateNew(t *testing.T) {
	// given - new session (nil SessionFile)
	sessionID := "test-session-id"
	client := &claude.Client{}
	reader := bufio.NewReader(strings.NewReader(""))
	systemPrompt := "test prompt"
	model := "test-model"
	var todos []tools.Todo

	// when
	agent, err := NewAgent(sessionID, nil, client, reader, systemPrompt, model, nil, &todos)

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent, got nil")
	}
	if agent.sessionID != sessionID {
		t.Errorf("expected sessionID %q, got %q", sessionID, agent.sessionID)
	}
	if len(agent.messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(agent.messages))
	}
	if agent.planMode {
		t.Error("expected planMode false, got true")
	}
	if agent.permissionsMode != "prompt" {
		t.Errorf("expected permissionsMode 'prompt', got %q", agent.permissionsMode)
	}
}

func TestNewAgent_Resume(t *testing.T) {
	// given - existing session
	sessionID := "existing-id"
	sess := &SessionFile{
		Messages: []claude.MessageParam{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
		Todos: []tools.Todo{
			{Content: "task", ActiveForm: "doing task", Status: "pending"},
		},
		PlanMode:        true,
		PermissionsMode: "accept_all",
	}
	client := &claude.Client{}
	reader := bufio.NewReader(strings.NewReader(""))
	var todos []tools.Todo

	// when
	agent, err := NewAgent(sessionID, sess, client, reader, "prompt", "model", nil, &todos)

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(agent.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(agent.messages))
	}
	if !agent.planMode {
		t.Error("expected planMode true, got false")
	}
	if agent.permissionsMode != "accept_all" {
		t.Errorf("expected permissionsMode 'accept_all', got %q", agent.permissionsMode)
	}
	if len(*agent.todos) != 1 {
		t.Errorf("expected 1 todo, got %d", len(*agent.todos))
	}
}

func TestNewAgent_NilSession(t *testing.T) {
	// given - nil session AND empty sessionID
	client := &claude.Client{}
	reader := bufio.NewReader(strings.NewReader(""))
	var todos []tools.Todo

	// when
	agent, err := NewAgent("", nil, client, reader, "prompt", "model", nil, &todos)

	// then - should still work (creates new session)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent, got nil")
	}
}

func TestAgent_HandleInput_PlanToggle(t *testing.T) {
	// given - agent with planMode false
	agent := &Agent{
		planMode: false,
		messages: []claude.MessageParam{},
	}

	// when - /plan command
	shouldInfer, err := agent.HandleInput("/plan")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if shouldInfer {
		t.Error("expected false (no inference), got true")
	}
	if !agent.planMode {
		t.Error("expected planMode true, got false")
	}
}

func TestAgent_HandleInput_BashContext(t *testing.T) {
	// given - agent with empty messages
	agent := &Agent{
		messages: []claude.MessageParam{},
	}

	// when - !! command
	shouldInfer, err := agent.HandleInput("!!echo test")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if shouldInfer {
		t.Error("expected false (no inference for bash), got true")
	}
	if len(agent.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(agent.messages))
	}
	msg := agent.messages[0]
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	content, ok := msg.Content.(string)
	if !ok {
		t.Fatal("expected string content")
	}
	if !strings.Contains(content, "echo test") {
		t.Errorf("expected content to contain 'echo test', got %q", content)
	}
}

func TestAgent_HandleInput_BashNoContext(t *testing.T) {
	// given - agent
	agent := &Agent{
		messages: []claude.MessageParam{},
	}

	// when - ! command
	shouldInfer, err := agent.HandleInput("!echo test")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if shouldInfer {
		t.Error("expected false (no inference for bash), got true")
	}
	if len(agent.messages) != 0 {
		t.Errorf("expected 0 messages (no context), got %d", len(agent.messages))
	}
}

func TestAgent_HandleInput_Normal(t *testing.T) {
	// given - agent
	agent := &Agent{
		messages: []claude.MessageParam{},
	}

	// when - normal input
	shouldInfer, err := agent.HandleInput("hello")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !shouldInfer {
		t.Error("expected true (should infer), got false")
	}
	if len(agent.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(agent.messages))
	}
	msg := agent.messages[0]
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}
	content, ok := msg.Content.(string)
	if !ok {
		t.Fatal("expected string content")
	}
	if content != "hello" {
		t.Errorf("expected content 'hello', got %q", content)
	}
}

func TestAgent_Save_Success(t *testing.T) {
	// given - agent with session ID and state
	sessionID := newSessionID()
	var todos []tools.Todo
	agent := &Agent{
		sessionID:           sessionID,
		messages:            []claude.MessageParam{{Role: "user", Content: "test"}},
		todos:               &todos,
		planMode:            false,
		permissionsMode:     "prompt",
		turnsSinceTodoWrite: 0,
	}

	// when
	err := agent.Save()

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if agent.turnsSinceTodoWrite != 1 {
		t.Errorf("expected turnsSinceTodoWrite 1, got %d", agent.turnsSinceTodoWrite)
	}

	// verify session was saved
	sess, err := loadSession(sessionID)
	if err != nil {
		t.Fatalf("failed to load saved session: %v", err)
	}
	if len(sess.Messages) != 1 {
		t.Errorf("expected 1 message in saved session, got %d", len(sess.Messages))
	}

	// cleanup
	deleteSession(sessionID)
}

func TestAgent_Save_EmptySessionID(t *testing.T) {
	// given - agent with empty sessionID
	var todos []tools.Todo
	agent := &Agent{
		sessionID: "",
		todos:     &todos,
	}

	// when
	err := agent.Save()

	// then - should not error, just skip save
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
