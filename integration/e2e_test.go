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

	cmd := exec.Command("../blocc", "--stdout", scriptPath)
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
	// Verify stdout is included when --stdout flag is used
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
