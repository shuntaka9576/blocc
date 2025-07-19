package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type ErrorOutput struct {
	Message string `json:"message"`
	Results []struct {
		Command  string `json:"command"`
		ExitCode int    `json:"exitCode"`
		Stderr   string `json:"stderr"`
		Stdout   string `json:"stdout,omitempty"`
	} `json:"results"`
}

func TestMain(m *testing.M) {
	// Build the binary before running tests
	cmd := exec.Command("go", "build", "-o", "../blocc", "../cmd/blocc")
	if err := cmd.Run(); err != nil {
		panic("Failed to build blocc binary: " + err.Error())
	}

	code := m.Run()

	// Clean up
	os.Remove("../blocc")

	os.Exit(code)
}

func TestBlocc_SequentialExecution(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantErrCount int
		validateOut  func(t *testing.T, stderr string)
	}{
		{
			name:         "all commands succeed",
			args:         []string{"echo hello", "echo world"},
			wantExitCode: 0,
			wantErrCount: 0,
		},
		{
			name:         "one command fails",
			args:         []string{"echo hello", "false", "echo world"},
			wantExitCode: 2,
			wantErrCount: 1,
		},
		{
			name:         "exit 2 stops early",
			args:         []string{"echo hello", "exit 2", "echo world"},
			wantExitCode: 2,
			wantErrCount: 1,
			validateOut: func(t *testing.T, stderr string) {
				var errOut ErrorOutput
				if err := json.Unmarshal([]byte(stderr), &errOut); err != nil {
					t.Fatalf("Failed to unmarshal stderr: %v", err)
				}
				// We might get 1 or 2 results depending on if exit 2 is recognized
				if len(errOut.Results) < 1 {
					t.Errorf("Expected at least 1 failed result, got %d", len(errOut.Results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("../blocc", tt.args...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.wantExitCode == 0 && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			} else if tt.wantExitCode != 0 {
				if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != tt.wantExitCode {
					t.Errorf("Expected exit code %d, got %v", tt.wantExitCode, err)
				}
			}

			if tt.wantErrCount > 0 {
				var errOut ErrorOutput
				if err := json.Unmarshal(stderr.Bytes(), &errOut); err != nil {
					t.Fatalf("Failed to unmarshal stderr: %v", err)
				}
				if len(errOut.Results) != tt.wantErrCount {
					t.Errorf("Expected %d failed commands, got %d", tt.wantErrCount, len(errOut.Results))
				}
			}

			if tt.validateOut != nil {
				tt.validateOut(t, stderr.String())
			}
		})
	}
}

func TestBlocc_ParallelExecution(t *testing.T) {
	cmd := exec.Command("../blocc", "--parallel", "echo hello", "false", "echo world")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
		t.Errorf("Expected exit code 2, got %v", err)
	}

	var errOut ErrorOutput
	if err := json.Unmarshal(stderr.Bytes(), &errOut); err != nil {
		t.Fatalf("Failed to unmarshal stderr: %v", err)
	}

	if len(errOut.Results) != 1 {
		t.Errorf("Expected 1 failed command, got %d", len(errOut.Results))
	}
}

func TestBlocc_AlwaysIncludesStdout(t *testing.T) {
	// Create a test script that outputs to both stdout and stderr
	scriptPath := filepath.Join(t.TempDir(), "test.sh")
	script := `#!/bin/sh
echo "stdout line"
echo "stderr line" >&2
exit 1`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("../blocc", scriptPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // We expect this to fail

	var errOut ErrorOutput
	if err := json.Unmarshal(stderr.Bytes(), &errOut); err != nil {
		t.Fatalf("Failed to unmarshal stderr: %v", err)
	}

	if len(errOut.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(errOut.Results))
	}

	result := errOut.Results[0]
	// Verify stdout is always included
	if !strings.Contains(result.Stdout, "stdout line") {
		t.Errorf("Expected stdout to contain 'stdout line', got %q", result.Stdout)
	}

	if !strings.Contains(result.Stderr, "stderr line") {
		t.Errorf("Expected stderr to contain 'stderr line', got %q", result.Stderr)
	}
}

func TestBlocc_CustomMessage(t *testing.T) {
	customMsg := "Custom error occurred"
	cmd := exec.Command("../blocc", "--message", customMsg, "false")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // We expect this to fail

	var errOut ErrorOutput
	if err := json.Unmarshal(stderr.Bytes(), &errOut); err != nil {
		t.Fatalf("Failed to unmarshal stderr: %v", err)
	}

	if errOut.Message != customMsg {
		t.Errorf("Expected message %q, got %q", customMsg, errOut.Message)
	}
}

func TestBlocc_Version(t *testing.T) {
	cmd := exec.Command("../blocc", "--version")
	output, err := cmd.Output()

	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}

	if !strings.Contains(string(output), "blocc version") {
		t.Errorf("Expected version output to contain 'blocc version', got %q", string(output))
	}
}

// validateInitResult validates the generated settings file structure
func validateInitResult(t *testing.T, tmpDir, expectedCommand string) {
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Errorf("settings.local.json was not created at %s", settingsPath)
		return
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Generated file is not valid JSON: %v", err)
	}

	hooks, ok := result["hooks"].(map[string]interface{})
	if !ok {
		t.Error("Missing 'hooks' field")
		return
	}

	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok {
		t.Error("Missing 'PostToolUse' field")
		return
	}

	if len(postToolUse) != 1 {
		t.Errorf("Expected 1 PostToolUse entry, got %d", len(postToolUse))
		return
	}

	entry := postToolUse[0].(map[string]interface{})
	matcher, ok := entry["matcher"].(string)
	if !ok || matcher != "Write|Edit|MultiEdit" {
		t.Errorf("Expected matcher 'Write|Edit|MultiEdit', got %q", matcher)
	}

	entryHooks, ok := entry["hooks"].([]interface{})
	if !ok || len(entryHooks) != 1 {
		t.Errorf("Expected 1 hook entry, got %d", len(entryHooks))
		return
	}

	hook := entryHooks[0].(map[string]interface{})
	hookType, ok := hook["type"].(string)
	if !ok || hookType != "command" {
		t.Errorf("Expected hook type 'command', got %q", hookType)
	}

	command, ok := hook["command"].(string)
	if !ok || command != expectedCommand {
		t.Errorf("Expected command %q, got %q", expectedCommand, command)
	}
}

func TestBlocc_Init(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedCommand string
	}{
		{
			name:            "default init",
			args:            []string{"--init"},
			expectedCommand: `blocc --message "Hook execution completed with errors" "npx tsc --noEmit"`,
		},
		{
			name:            "init with custom command",
			args:            []string{"--init", "npm run lint"},
			expectedCommand: `blocc --message "Hook execution completed with errors" "npm run lint"`,
		},
		{
			name:            "init with custom message and commands",
			args:            []string{"--init", "--message", "Custom error", "npm run lint", "npm run test"},
			expectedCommand: `blocc --message "Custom error" "npm run lint" "npm run test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			bloccPath, err := filepath.Abs("../blocc")
			if err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(bloccPath, tt.args...)
			cmd.Dir = tmpDir
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = cmd.Run()
			if err != nil {
				t.Fatalf("Init command failed: %v, stderr: %s", err, stderr.String())
			}

			outputStr := stdout.String()
			if !strings.Contains(outputStr, "Successfully created settings.local.json") {
				t.Errorf("Expected success message, got %q", outputStr)
			}

			validateInitResult(t, tmpDir, tt.expectedCommand)
		})
	}
}

func TestBlocc_InitFileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the .claude directory and settings file first
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	bloccPath, err := filepath.Abs("../blocc")
	if err != nil {
		t.Fatal(err)
	}

	// Try to init again - should fail
	cmd := exec.Command(bloccPath, "--init")
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		t.Error("Expected command to fail when file already exists")
		return
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "already exists") {
		t.Errorf("Expected error message to contain 'already exists', got %q", stderrStr)
	}
}

func TestBlocc_InitPathDisplay(t *testing.T) {
	// Test that path display shows ~ when possible
	tmpDir := t.TempDir()

	bloccPath, err := filepath.Abs("../blocc")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bloccPath, "--init")
	cmd.Dir = tmpDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	outputStr := stdout.String()
	if !strings.Contains(outputStr, "Successfully created settings.local.json") {
		t.Errorf("Expected success message, got %q", outputStr)
	}

	// The path should show either full path or ~ format, but should be informative
	if !strings.Contains(outputStr, ".claude/settings.local.json") {
		t.Errorf("Expected path to contain '.claude/settings.local.json', got %q", outputStr)
	}
}
