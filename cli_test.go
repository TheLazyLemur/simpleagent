package main

import (
	"flag"
	"testing"
)

func TestRunCLI_ListSessions(t *testing.T) {
	// given - -sessions flag set
	oldArgs := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	listFlag := flag.Bool("sessions", true, "")
	deleteFlag := flag.String("delete", "", "")
	resumeFlag := flag.String("resume", "", "")

	// when
	shouldContinue, err := RunCLI(listFlag, deleteFlag, resumeFlag)

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if shouldContinue {
		t.Error("expected false (should not continue), got true")
	}

	// cleanup
	flag.CommandLine = oldArgs
}

func TestRunCLI_DeleteSession(t *testing.T) {
	// given - -delete flag set
	oldArgs := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	listFlag := flag.Bool("sessions", false, "")
	deleteFlag := flag.String("delete", "test-id", "")
	resumeFlag := flag.String("resume", "", "")

	// when
	shouldContinue, err := RunCLI(listFlag, deleteFlag, resumeFlag)

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if shouldContinue {
		t.Error("expected false (should not continue), got true")
	}

	// cleanup
	flag.CommandLine = oldArgs
}

func TestRunCLI_Continue(t *testing.T) {
	// given - no flags set (normal run)
	oldArgs := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	listFlag := flag.Bool("sessions", false, "")
	deleteFlag := flag.String("delete", "", "")
	resumeFlag := flag.String("resume", "", "")

	// when
	shouldContinue, err := RunCLI(listFlag, deleteFlag, resumeFlag)

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !shouldContinue {
		t.Error("expected true (should continue), got false")
	}

	// cleanup
	flag.CommandLine = oldArgs
}
