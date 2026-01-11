package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"simpleagent/claude"
)

const defaultTimeout = 30 * time.Second

func init() {
	register(claude.Tool{
		Name:        "bash",
		Description: "Run a bash command",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"args": {
					Type:        "array",
					Description: "Command and arguments",
					Items:       &claude.Property{Type: "string"},
				},
				"timeout_sec": {
					Type:        "integer",
					Description: "Timeout in seconds (default: 30)",
				},
				"cwd": {
					Type:        "string",
					Description: "Working directory (optional)",
				},
			},
			Required: []string{"args"},
		},
	}, bash)
}

type bashResult struct {
	output string
}

func (r bashResult) String() string { return r.output }
func (r bashResult) Render()        { fmt.Print(r.output) }

func bash(input json.RawMessage) Result {
	var args struct {
		Args       []string `json:"args"`
		TimeoutSec *int     `json:"timeout_sec"`
		Cwd        string   `json:"cwd"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("bash", Error(err.Error()))
	}
	if len(args.Args) == 0 {
		return newResult("bash", Error("args is required"))
	}

	timeout := defaultTimeout
	if args.TimeoutSec != nil {
		timeout = time.Duration(*args.TimeoutSec) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if len(args.Args) == 1 {
		cmd = exec.CommandContext(ctx, "sh", "-c", args.Args[0])
	} else {
		cmd = exec.CommandContext(ctx, args.Args[0], args.Args[1:]...)
	}
	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return bashResult{
		output: fmt.Sprintf("%s\n%s", string(out), Status(fmt.Sprintf("exit code: %d", exitCode))),
	}
}
