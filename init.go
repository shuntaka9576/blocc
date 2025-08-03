package blocc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type PostToolUseItem struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

type Settings struct {
	Hooks struct {
		PostToolUse []PostToolUseItem `json:"PostToolUse"`
	} `json:"hooks"`
}

func InitSettings(commands []string, message string, includeStdout bool) error {
	// Use defaults if not provided
	defaultCommands := []string{"npx tsc --noEmit"}
	defaultMessage := "Hook execution completed with errors"

	if len(commands) == 0 {
		commands = defaultCommands
	}
	if message == "" {
		message = defaultMessage
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	claudeDir := filepath.Join(currentDir, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", claudeDir, err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	// Check if file already exists
	_, statErr := os.Stat(settingsPath)
	if statErr == nil {
		return fmt.Errorf("settings.local.json already exists at %s", settingsPath)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to check file existence: %w", statErr)
	}

	// Build command string
	quotedCommands := make([]string, len(commands))
	for i, cmd := range commands {
		quotedCommands[i] = fmt.Sprintf("\"%s\"", cmd)
	}
	commandStr := fmt.Sprintf("blocc --message \"%s\"", message)
	if includeStdout {
		commandStr += " --stdout"
	}
	commandStr += " " + strings.Join(quotedCommands, " ")

	// Create settings structure
	settings := Settings{}
	settings.Hooks.PostToolUse = []PostToolUseItem{
		{
			Matcher: "Write|Edit|MultiEdit",
			Hooks: []Hook{
				{
					Type:    "command",
					Command: commandStr,
				},
			},
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write the file
	if err := os.WriteFile(settingsPath, jsonBytes, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Replace home directory with ~ for display
	displayPath := settingsPath
	if homeDir, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(settingsPath, homeDir) {
			displayPath = "~" + settingsPath[len(homeDir):]
		}
	}

	fmt.Printf("Successfully created settings.local.json at %s\n", displayPath)
	return nil
}
