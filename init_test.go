package blocc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitSettings(t *testing.T) {
	tests := []struct {
		name            string
		commands        []string
		message         string
		expectedCommand string
	}{
		{
			name:            "custom single command",
			commands:        []string{"npm run lint"},
			message:         "",
			expectedCommand: `blocc 'npm run lint'`,
		},
		{
			name:            "custom multiple commands",
			commands:        []string{"npm run lint", "npm run test"},
			message:         "",
			expectedCommand: `blocc 'npm run lint' 'npm run test'`,
		},
		{
			name:            "custom message and commands",
			commands:        []string{"pnpm lint", "pnpm test"},
			message:         "Custom error message",
			expectedCommand: `blocc --message "Custom error message" 'pnpm lint' 'pnpm test'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.Chdir(originalWd)
			}()

			if err = os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			// Run InitSettings
			err = InitSettings(tt.commands, tt.message, false, "", "", false)
			if err != nil {
				t.Fatalf("InitSettings failed: %v", err)
			}

			// Check if file was created
			settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
			if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
				t.Errorf("settings.local.json was not created at %s", settingsPath)
				return
			}

			// Read and validate the file content
			content, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("Failed to read settings file: %v", err)
			}

			var settings Settings
			if err := json.Unmarshal(content, &settings); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate structure
			if len(settings.Hooks.Stop) != 1 {
				t.Errorf("Expected 1 Stop hook, got %d", len(settings.Hooks.Stop))
			}

			stopHook := settings.Hooks.Stop[0]
			if stopHook.Matcher != "" {
				t.Errorf("Expected empty matcher, got %q", stopHook.Matcher)
			}

			if len(stopHook.Hooks) != 1 {
				t.Errorf("Expected 1 hook, got %d", len(stopHook.Hooks))
			}

			hook := stopHook.Hooks[0]
			if hook.Type != "command" {
				t.Errorf("Expected hook type 'command', got %q", hook.Type)
			}

			if hook.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, hook.Command)
			}
		})
	}
}

func TestInitSettings_FileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	if err = os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	claudeDir := filepath.Join(tmpDir, ".claude")
	if err = os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	if err = os.WriteFile(settingsPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Should fail when file already exists
	err = InitSettings([]string{"echo test"}, "", false, "", "", false)
	if err == nil {
		t.Error("Expected error when file already exists, got nil")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected error message to contain 'already exists', got %q", err.Error())
	}
}

func TestInitSettings_PathDisplay(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	if err = os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Capture output by redirecting stdout temporarily
	// Note: In real implementation, we would need to capture the output
	// For now, just ensure the function succeeds
	err = InitSettings([]string{"echo test"}, "", false, "", "", false)
	if err != nil {
		t.Fatalf("InitSettings failed: %v", err)
	}

	// Verify file was created
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Error("settings.local.json was not created")
	}
}

func TestInitSettings_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	if err = os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = InitSettings([]string{"echo test"}, "Test message", false, "", "", false)
	if err != nil {
		t.Fatalf("InitSettings failed: %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	// Validate it's proper JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Generated file is not valid JSON: %v", err)
	}

	// Check specific structure
	hooks, ok := result["hooks"].(map[string]interface{})
	if !ok {
		t.Error("Missing 'hooks' field")
	}

	stopHooks, ok := hooks["Stop"].([]interface{})
	if !ok {
		t.Error("Missing 'Stop' field")
	}

	if len(stopHooks) != 1 {
		t.Errorf("Expected 1 Stop entry, got %d", len(stopHooks))
	}
}

func TestInitSettings_WithStdout(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Test with stdout enabled
	err := InitSettings([]string{"echo test"}, "Test message", true, "", "", false)
	if err != nil {
		t.Fatalf("InitSettings failed: %v", err)
	}

	// Read and parse the created file
	settingsPath := filepath.Join(tempDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Navigate to the command in the JSON structure
	hooks := result["hooks"].(map[string]interface{})
	stopHooks := hooks["Stop"].([]interface{})
	firstItem := stopHooks[0].(map[string]interface{})
	hooksArray := firstItem["hooks"].([]interface{})
	firstHook := hooksArray[0].(map[string]interface{})
	command := firstHook["command"].(string)

	// Check that --stdout is included in the command
	expectedCommand := `blocc --message "Test message" --stdout 'echo test'`
	if command != expectedCommand {
		t.Errorf("Expected command %q, got %q", expectedCommand, command)
	}
}

func TestAskIncludeStdoutFromReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "yes lowercase",
			input:    "yes\n",
			expected: true,
		},
		{
			name:     "y lowercase",
			input:    "y\n",
			expected: true,
		},
		{
			name:     "YES uppercase",
			input:    "YES\n",
			expected: true,
		},
		{
			name:     "Y uppercase",
			input:    "Y\n",
			expected: true,
		},
		{
			name:     "no",
			input:    "no\n",
			expected: false,
		},
		{
			name:     "n",
			input:    "n\n",
			expected: false,
		},
		{
			name:     "empty",
			input:    "\n",
			expected: false,
		},
		{
			name:     "other input",
			input:    "maybe\n",
			expected: false,
		},
		{
			name:     "with spaces",
			input:    "  y  \n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			result, err := askIncludeStdoutFromReader(reader)
			if err != nil {
				t.Fatalf("askIncludeStdoutFromReader failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetInteractiveCommandsFromReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single command",
			input:    "npm run test\n\n",
			expected: []string{"npm run test"},
			wantErr:  false,
		},
		{
			name:     "multiple commands",
			input:    "npm run lint\nnpm run test\n\n",
			expected: []string{"npm run lint", "npm run test"},
			wantErr:  false,
		},
		{
			name:     "commands with spaces",
			input:    "  npm run lint  \n  npm run test  \n\n",
			expected: []string{"npm run lint", "npm run test"},
			wantErr:  false,
		},
		{
			name:     "empty input",
			input:    "\n",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "only spaces",
			input:    "  \n",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			result, err := getInteractiveCommandsFromReader(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("getInteractiveCommandsFromReader error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d commands, got %d", len(tt.expected), len(result))
					return
				}
				for i, cmd := range result {
					if cmd != tt.expected[i] {
						t.Errorf("Command %d: expected %q, got %q", i, tt.expected[i], cmd)
					}
				}
			}
		})
	}
}

func TestInitSettings_WithFilters(t *testing.T) {
	tests := []struct {
		name            string
		commands        []string
		message         string
		includeStdout   bool
		stdoutFilter    string
		stderrFilter    string
		expectedCommand string
	}{
		{
			name:            "with stdout filter",
			commands:        []string{"npm run test"},
			message:         "",
			includeStdout:   true,
			stdoutFilter:    "grep ERROR",
			stderrFilter:    "",
			expectedCommand: `blocc --stdout --stdout-filter "grep ERROR" 'npm run test'`,
		},
		{
			name:            "with stderr filter",
			commands:        []string{"npm run test"},
			message:         "",
			includeStdout:   false,
			stdoutFilter:    "",
			stderrFilter:    "sed 's/error/ERROR/'",
			expectedCommand: `blocc --stderr-filter "sed 's/error/ERROR/'" 'npm run test'`,
		},
		{
			name:            "with both filters",
			commands:        []string{"npm run test"},
			message:         "",
			includeStdout:   true,
			stdoutFilter:    "grep ERROR",
			stderrFilter:    "grep WARNING",
			expectedCommand: `blocc --stdout --stdout-filter "grep ERROR" --stderr-filter "grep WARNING" 'npm run test'`,
		},
		{
			name:          "with all options",
			commands:      []string{"npm run test", "npm run lint"},
			message:       "Tests failed",
			includeStdout: true,
			stdoutFilter:  "head -n 10",
			stderrFilter:  "tail -n 20",
			expectedCommand: `blocc --message "Tests failed" --stdout ` +
				`--stdout-filter "head -n 10" --stderr-filter "tail -n 20" ` +
				`'npm run test' 'npm run lint'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			// Run InitSettings
			err := InitSettings(tt.commands, tt.message, tt.includeStdout, tt.stdoutFilter, tt.stderrFilter, false)
			if err != nil {
				t.Fatalf("InitSettings failed: %v", err)
			}

			// Check if file was created
			settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
			content, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("Failed to read settings file: %v", err)
			}

			var settings Settings
			if err := json.Unmarshal(content, &settings); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate command
			hook := settings.Hooks.Stop[0].Hooks[0]
			if hook.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, hook.Command)
			}
		})
	}
}
