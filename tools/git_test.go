package tools

import (
	"testing"
)

func TestValidateGitCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		// Read-only commands - should be allowed
		{"status", []string{"status"}, false},
		{"log", []string{"log"}, false},
		{"log with options", []string{"log", "--oneline"}, false},
		{"show", []string{"show", "HEAD"}, false},
		{"diff", []string{"diff"}, false},
		{"branch list", []string{"branch"}, false},
		{"branch with verbose", []string{"branch", "-v"}, false},
		{"tag list", []string{"tag"}, false},
		{"remote list", []string{"remote"}, false},
		{"fetch", []string{"fetch", "origin"}, false},
		{"ls-files", []string{"ls-files"}, false},
		{"config list", []string{"config", "--list"}, false},
		{"rev-parse", []string{"rev-parse", "HEAD"}, false},
		{"blame", []string{"blame", "main.go"}, false},
		{"grep", []string{"grep", "TODO"}, false},

		// Write commands - should be blocked
		{"commit", []string{"commit", "-m", "test"}, true},
		{"checkout -b", []string{"checkout", "-b", "test-branch"}, true},
		{"checkout -B", []string{"checkout", "-B", "test-branch"}, true},
		{"branch -d", []string{"branch", "-d", "test"}, true},
		{"branch -D", []string{"branch", "-D", "test"}, true},
		{"branch --delete", []string{"branch", "--delete", "test"}, true},
		{"tag -d", []string{"tag", "-d", "v1.0"}, true},
		{"remote add", []string{"remote", "add", "origin", "url"}, true},
		{"remote remove", []string{"remote", "remove", "origin"}, true},
		{"push", []string{"push", "origin", "main"}, true},
		{"push tags", []string{"push", "--tags"}, true},
		{"merge", []string{"merge", "feature"}, true},
		{"reset hard", []string{"reset", "--hard", "HEAD"}, true},
		{"reset soft", []string{"reset", "--soft", "HEAD~1"}, true},
		{"rebase", []string{"rebase", "main"}, true},
		{"cherry-pick", []string{"cherry-pick", "abc123"}, true},
		{"revert", []string{"revert", "abc123"}, true},
		{"stash", []string{"stash"}, true},
		{"stash push", []string{"stash", "push", "-m", "message"}, true},
		{"clean", []string{"clean", "-fd"}, true},
		{"restore", []string{"restore", "file.txt"}, true},
		{"switch", []string{"switch", "main"}, true},
		{"switch -c", []string{"switch", "-c", "test"}, true},

		// Edge cases
		{"empty args", []string{}, true},
		{"unknown command", []string{"unknown-cmd"}, true},
		{"checkout main", []string{"checkout", "main"}, false},
		{"checkout -- file", []string{"checkout", "--", "file"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitCommand(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitCommand(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}
