package blocc

import (
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
			name:            "default settings",
			commands:        nil,
			message:         "",
			expectedCommand: `blocc --message "Hook execution completed with errors" "npx tsc --noEmit"`,
		},
		{
			name:            "custom single command",
			commands:        []string{"npm run lint"},
			message:         "",
			expectedCommand: `blocc --message "Hook execution completed with errors" "npm run lint"`,
		},
		{
			name:            "custom multiple commands",
			commands:        []string{"npm run lint", "npm run test"},
			message:         "",
			expectedCommand: `blocc --message "Hook execution completed with errors" "npm run lint" "npm run test"`,
		},
		{
			name:            "custom message and commands",
			commands:        []string{"pnpm lint", "pnpm test"},
			message:         "Custom error message",
			expectedCommand: `blocc --message "Custom error message" "pnpm lint" "pnpm test"`,
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
			err = InitSettings(tt.commands, tt.message)
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
			if len(settings.Hooks.PostToolUse) != 1 {
				t.Errorf("Expected 1 PostToolUse hook, got %d", len(settings.Hooks.PostToolUse))
			}

			postToolUse := settings.Hooks.PostToolUse[0]
			if postToolUse.Matcher != "Write|Edit|MultiEdit" {
				t.Errorf("Expected matcher 'Write|Edit|MultiEdit', got %q", postToolUse.Matcher)
			}

			if len(postToolUse.Hooks) != 1 {
				t.Errorf("Expected 1 hook, got %d", len(postToolUse.Hooks))
			}

			hook := postToolUse.Hooks[0]
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
	err = InitSettings(nil, "")
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
	err = InitSettings(nil, "")
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

	err = InitSettings([]string{"echo test"}, "Test message")
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

	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok {
		t.Error("Missing 'PostToolUse' field")
	}

	if len(postToolUse) != 1 {
		t.Errorf("Expected 1 PostToolUse entry, got %d", len(postToolUse))
	}
}
