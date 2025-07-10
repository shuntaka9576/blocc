package blocc

import (
	"strings"
	"testing"
)

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		wantExitCode int
		wantStdout   string
		wantStderr   string
	}{
		{
			name:         "echo command success",
			command:      "echo hello",
			wantExitCode: 0,
			wantStdout:   "hello\n",
			wantStderr:   "",
		},
		{
			name:         "false command",
			command:      "false",
			wantExitCode: 1,
			wantStdout:   "",
			wantStderr:   "",
		},
		{
			name:         "non-existent command",
			command:      "nonexistentcommand123",
			wantExitCode: 1,
			wantStdout:   "",
			wantStderr:   "executable file not found",
		},
		{
			name:         "empty command",
			command:      "",
			wantExitCode: 1,
			wantStdout:   "",
			wantStderr:   "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			result := executor.executeCommand(tt.command)

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("executeCommand() exitCode = %v, want %v", result.ExitCode, tt.wantExitCode)
			}

			if result.Stdout != tt.wantStdout {
				t.Errorf("executeCommand() stdout = %v, want %v", result.Stdout, tt.wantStdout)
			}

			if tt.wantStderr != "" && !strings.Contains(result.Stderr, tt.wantStderr) {
				t.Errorf("executeCommand() stderr = %v, want contains %v", result.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestExecuteSequential(t *testing.T) {
	tests := []struct {
		name        string
		commands    []string
		wantResults int
	}{
		{
			name:        "all success commands",
			commands:    []string{"echo hello", "echo world"},
			wantResults: 0,
		},
		{
			name:        "one failing command",
			commands:    []string{"echo hello", "false"},
			wantResults: 1,
		},
		{
			name:        "true and false commands",
			commands:    []string{"true", "false", "echo hello"},
			wantResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			results, _ := executor.ExecuteSequential(tt.commands)

			if len(results) != tt.wantResults {
				t.Errorf("ExecuteSequential() results count = %v, want %v", len(results), tt.wantResults)
			}
		})
	}
}

func TestExecuteParallel(t *testing.T) {
	tests := []struct {
		name           string
		commands       []string
		wantMinResults int
	}{
		{
			name:           "all success commands",
			commands:       []string{"echo hello", "echo world"},
			wantMinResults: 0,
		},
		{
			name:           "mixed success and failure",
			commands:       []string{"echo hello", "false", "echo world"},
			wantMinResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			results, _ := executor.ExecuteParallel(tt.commands)

			if len(results) < tt.wantMinResults {
				t.Errorf("ExecuteParallel() results count = %v, want at least %v", len(results), tt.wantMinResults)
			}
		})
	}
}
