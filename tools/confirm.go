package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var permissionsMode = "prompt" // "prompt" or "accept_all"

// SetPermissionsMode sets the global permissions mode
func SetPermissionsMode(mode string) {
	permissionsMode = mode
}

// GetPermissionsMode returns the current permissions mode
func GetPermissionsMode() string {
	return permissionsMode
}

// RequestPermission prompts user for permission
func RequestPermission(op, path, details string) (bool, string, bool) {
	return RequestPermissionWithDiff(op, path, details, "")
}

// RequestPermissionWithDiff prompts user with optional diff preview
// Returns (allowed, reason, setAcceptAll)
func RequestPermissionWithDiff(op, path, details, diff string) (bool, string, bool) {
	// Check if we're in accept_all mode
	if permissionsMode == "accept_all" {
		return true, "auto-accepted (session mode)", false
	}

	fmt.Println()
	fmt.Println(Status("permission"))
	fmt.Println(KeyValue("operation", op))
	fmt.Println(KeyValue("path", path))
	if details != "" {
		fmt.Println(KeyValue("details", details))
	}
	if diff != "" {
		fmt.Printf("\n%s\n", diff)
	}
	fmt.Print(Prompt() + Dim("[y/n/a] "))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "a" || response == "all" {
		return true, "permission granted (accept all this session)", true
	}
	if response == "y" || response == "yes" {
		return true, "permission granted", false
	}
	return false, "permission denied", false
}

// truncateLines returns first max lines, or all if shorter
func truncateLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	return lines[:max]
}
