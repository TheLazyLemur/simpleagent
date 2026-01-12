package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"simpleagent/claude"
	"simpleagent/tools"
)

var sessionDir = filepath.Join(os.Getenv("HOME"), ".config", "agent", "sessions")

type SessionMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
}

type SessionFile struct {
	Meta            SessionMeta           `json:"meta"`
	Messages        []claude.MessageParam `json:"messages"`
	Todos           []tools.Todo          `json:"todos,omitempty"`
	PlanMode        bool                  `json:"plan_mode,omitempty"`
	PermissionsMode string                `json:"permissions_mode,omitempty"` // "prompt" or "accept_all"
}

func newSessionID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

func sessionPath(id string) string {
	return filepath.Join(sessionDir, id+".json")
}

func saveSession(id string, messages []claude.MessageParam, todos []tools.Todo, planMode bool, permissionsMode, provider, model string) error {
	sess := SessionFile{
		Meta: SessionMeta{
			ID:        id,
			UpdatedAt: time.Now(),
			Provider:  provider,
			Model:     model,
		},
		Messages:        messages,
		Todos:           todos,
		PlanMode:        planMode,
		PermissionsMode: permissionsMode,
	}

	if existing, err := loadSession(id); err == nil {
		sess.Meta.CreatedAt = existing.Meta.CreatedAt
	} else {
		sess.Meta.CreatedAt = time.Now()
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(id), data, 0644)
}

func loadSession(id string) (*SessionFile, error) {
	data, err := os.ReadFile(sessionPath(id))
	if err != nil {
		return nil, err
	}
	var sess SessionFile
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func listSessions() {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(tools.Dim("No sessions saved"))
			return
		}
		fmt.Println(tools.Error(fmt.Sprintf("reading sessions: %v", err)))
		return
	}

	var sessions []SessionFile
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		sess, err := loadSession(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		sessions = append(sessions, *sess)
	}

	if len(sessions) == 0 {
		fmt.Println(tools.Dim("No sessions saved"))
		return
	}

	slices.SortFunc(sessions, func(a, b SessionFile) int {
		return b.Meta.UpdatedAt.Compare(a.Meta.UpdatedAt)
	})

	// Table styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9FAFB"))
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	cellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))

	fmt.Println()
	fmt.Printf("  %s  %s  %s  %s\n",
		headerStyle.Width(36).Render("ID"),
		headerStyle.Width(16).Render("Created"),
		headerStyle.Width(16).Render("Updated"),
		headerStyle.Render("Msgs"),
	)
	fmt.Println(borderStyle.Render("  " + strings.Repeat("─", 78)))
	for _, s := range sessions {
		fmt.Printf("  %s  %s  %s  %s\n",
			idStyle.Width(36).Render(s.Meta.ID),
			cellStyle.Width(16).Render(s.Meta.CreatedAt.Format("2006-01-02 15:04")),
			cellStyle.Width(16).Render(s.Meta.UpdatedAt.Format("2006-01-02 15:04")),
			cellStyle.Render(fmt.Sprintf("%d", len(s.Messages))),
		)
	}
	fmt.Println()
}

func deleteSession(id string) {
	path := sessionPath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(tools.Error(fmt.Sprintf("session %s not found", id)))
		return
	}
	if err := os.Remove(path); err != nil {
		fmt.Println(tools.Error(fmt.Sprintf("delete failed: %v", err)))
		return
	}
	fmt.Println(tools.Success("deleted " + id))
}
