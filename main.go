package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
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

	shouldContinue, err := RunCLI(listFlag, deleteFlag, resumeFlag)
	if err != nil {
		fmt.Println(tools.Error(err.Error()))
		os.Exit(1)
	}
	if !shouldContinue {
		return
	}

	if err := AgentSession(resumeFlag); err != nil {
		fmt.Println(tools.Error(err.Error()))
		os.Exit(1)
	}
}
