package blocc

import (
	"bufio"
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

type HookItem struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

type Settings struct {
	Hooks struct {
		Stop []HookItem `json:"Stop"`
	} `json:"hooks"`
}

func getInteractiveCommands() ([]string, error) {
	fmt.Println("Enter commands to run (one per line, empty line to finish):")
	scanner := bufio.NewScanner(os.Stdin)
	var commands []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		commands = append(commands, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("no commands provided")
	}
	return commands, nil
}

func buildCommandString(commands []string, message string, includeStdout bool) string {
	quotedCommands := make([]string, len(commands))
	for i, cmd := range commands {
		quotedCommands[i] = fmt.Sprintf("'%s'", cmd)
	}
	commandStr := "blocc"
	if message != "" {
		commandStr += fmt.Sprintf(" --message \"%s\"", message)
	}
	if includeStdout {
		commandStr += " --stdout"
	}
	commandStr += " " + strings.Join(quotedCommands, " ")
	return commandStr
}

func createSettings(commandStr string) Settings {
	settings := Settings{}
	settings.Hooks.Stop = []HookItem{
		{
			Matcher: "",
			Hooks: []Hook{
				{
					Type:    "command",
					Command: commandStr,
				},
			},
		},
	}
	return settings
}

func InitSettings(commands []string, message string, includeStdout bool) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	claudeDir := filepath.Join(currentDir, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	// Check if file already exists before asking for input
	if _, statErr := os.Stat(settingsPath); statErr == nil {
		return fmt.Errorf("settings.local.json already exists at %s", settingsPath)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to check file existence: %w", statErr)
	}

	// If no commands provided, ask interactively
	if len(commands) == 0 {
		commands, err = getInteractiveCommands()
		if err != nil {
			return err
		}
	}

	// Create directory after all checks
	err = os.MkdirAll(claudeDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", claudeDir, err)
	}

	// Build command string
	commandStr := buildCommandString(commands, message, includeStdout)

	// Create settings structure
	settings := createSettings(commandStr)

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
