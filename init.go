package blocc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

func getInteractiveCommandsFromReader(reader io.Reader) ([]string, error) {
	fmt.Println("Enter commands to run (one per line, empty line to finish):")
	scanner := bufio.NewScanner(reader)
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

func getInteractiveSettings() ([]string, bool, string, string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	// Ask about stdout first
	fmt.Print("Include stdout in error output? (y/N): ")
	includeStdout := false
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		includeStdout = response == "y" || response == "yes"
	}

	// Ask about stdout filter
	fmt.Print("Add stdout filter? (y/N): ")
	var stdoutFilter string
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "y" || response == "yes" {
			fmt.Print("Enter stdout filter command: ")
			if scanner.Scan() {
				stdoutFilter = strings.TrimSpace(scanner.Text())
			}
		}
	}

	// Ask about stderr filter
	fmt.Print("Add stderr filter? (y/N): ")
	var stderrFilter string
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "y" || response == "yes" {
			fmt.Print("Enter stderr filter command: ")
			if scanner.Scan() {
				stderrFilter = strings.TrimSpace(scanner.Text())
			}
		}
	}

	// Then ask for commands
	fmt.Println("Enter commands to run (one per line, empty line to finish):")
	var commands []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		commands = append(commands, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, false, "", "", fmt.Errorf("failed to read input: %w", err)
	}
	if len(commands) == 0 {
		return nil, false, "", "", fmt.Errorf("no commands provided")
	}

	return commands, includeStdout, stdoutFilter, stderrFilter, nil
}

func askIncludeStdoutFromReader(reader io.Reader) (bool, error) {
	fmt.Print("Include stdout in error output? (y/N): ")
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		result := response == "y" || response == "yes"
		return result, nil
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	return false, nil
}

func buildCommandString(
	commands []string,
	message string,
	includeStdout bool,
	stdoutFilter, stderrFilter string,
) string {
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
	if stdoutFilter != "" {
		commandStr += fmt.Sprintf(" --stdout-filter \"%s\"", stdoutFilter)
	}
	if stderrFilter != "" {
		commandStr += fmt.Sprintf(" --stderr-filter \"%s\"", stderrFilter)
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

func InitSettings(commands []string, message string, includeStdout bool, stdoutFilter, stderrFilter string) error {
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
		commands, includeStdout, stdoutFilter, stderrFilter, err = getInteractiveSettings()
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
	commandStr := buildCommandString(commands, message, includeStdout, stdoutFilter, stderrFilter)

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
