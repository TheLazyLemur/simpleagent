package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"simpleagent/claude"
)

// readOnlyGitCommands contains git subcommands that are safe to run in plan mode
var readOnlyGitCommands = map[string]bool{
	"status":        true,
	"log":           true,
	"show":          true,
	"diff":          true,
	"branch":        true,
	"tag":           true,
	"remote":        true,
	"fetch":         true,
	"ls-files":      true,
	"ls-remote":     true,
	"rev-parse":     true,
	"describe":      true,
	"blame":         true,
	"archive":       true,
	"config":        true, // only --list
	"grep":          true, // conflicts with grep tool, but safe
	"shortlog":      true,
	"count-objects": true,
	"cat-file":      true,
	"hash-object":   true,
	"verify-pack":   true,
	"show-ref":      true,
	"symbolic-ref":  true,
	"for-each-ref":  true,
	"checkout":      true, // allowed, but -b/-B blocked via blockedSubcommands
}

// blockedSubcommands contains commands that are completely blocked (not in allowlist)
var blockedSubcommands = map[string]bool{
	"commit":      true,
	"merge":       true,
	"rebase":      true,
	"cherry-pick": true,
	"revert":      true,
	"push":        true,
	"pull":        true,
	"stash":       true,
	"clean":       true,
	"restore":     true,
	"switch":      true,
	"am":          true,
	"bisect":      true,
}

// blockedFlags contains flags that make a command a write operation even if the base command is read-only
var blockedFlags = map[string]map[string]bool{
	"branch": {
		"-d": true, "-D": true, "-m": true, "-M": true, "--delete": true, "--move": true, "--rename": true,
	},
	"tag": {
		"-d": true, "-D": true, "--delete": true,
	},
	"remote": {
		"add": true, "remove": true, "set-head": true, "set-branches": true, "set-url": true, "prune": true, "update": true,
	},
	"config": {
		"--set": true, "--add": true, "--unset": true, "--remove-section": true, "--rename-section": true,
	},
	"checkout": {
		"-b": true, "-B": true,
	},
	"log": {
		"--reverse": true,
	},
}

func init() {
	register(claude.Tool{
		Name:        "git",
		Description: "Run git commands (read-only operations only in plan mode)",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"args": {Type: "array", Description: "Git arguments (in order)", Items: &claude.Property{Type: "string"}},
			},
			Required: []string{"args"},
		},
	}, git)
}

func git(input json.RawMessage) Result {
	var args struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("git", Error(err.Error()))
	}

	if err := validateGitCommand(args.Args); err != nil {
		return newResult("git", Error(err.Error()))
	}

	cmd := exec.Command("git", args.Args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return newResult("git", Error(fmt.Sprintf("%v\n%s", err, string(out))))
	}
	return newResult("git", string(out))
}

// validateGitCommand checks if a git command is safe to run in plan mode
func validateGitCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no git command specified")
	}

	cmd := args[0]

	// Check if the command is completely blocked (write operations)
	if blockedSubcommands[cmd] {
		return fmt.Errorf("git command '%s' is not allowed in read-only mode", cmd)
	}

	// Check if the command is in our read-only allowlist
	if !readOnlyGitCommands[cmd] {
		return fmt.Errorf("git command '%s' is not allowed in read-only mode", cmd)
	}

	// Check for blocked flags on commands that have restrictions
	if blocked, ok := blockedFlags[cmd]; ok {
		for _, arg := range args[1:] {
			if blocked[arg] {
				return fmt.Errorf("git command '%s' with flag '%s' is not allowed in read-only mode", cmd, arg)
			}
		}
	}

	return nil
}
