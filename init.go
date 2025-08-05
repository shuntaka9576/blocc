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

func askYesNo(scanner *bufio.Scanner, prompt string) bool {
	fmt.Print(prompt)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	return false
}

func askFilterCommand(scanner *bufio.Scanner, filterType string) string {
	if askYesNo(scanner, fmt.Sprintf("Add %s filter? (y/N): ", filterType)) {
		fmt.Printf("Enter %s filter command: ", filterType)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text())
		}
	}
	return ""
}

func getInteractiveSettings() ([]string, bool, string, string, bool, error) {
	scanner := bufio.NewScanner(os.Stdin)

	// Ask about stdout
	includeStdout := askYesNo(scanner, "Include stdout in error output? (y/N): ")

	// Ask about filters
	stdoutFilter := askFilterCommand(scanner, "stdout")
	stderrFilter := askFilterCommand(scanner, "stderr")

	// Ask about no-stderr option
	noStderr := askYesNo(scanner, "Exclude stderr from error output? (y/N): ")

	// Then ask for commands
	commands, err := getInteractiveCommandsFromReader(os.Stdin)
	if err != nil {
		return nil, false, "", "", false, err
	}

	return commands, includeStdout, stdoutFilter, stderrFilter, noStderr, nil
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
	noStderr bool,
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
	if noStderr {
		commandStr += " --no-stderr"
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

func InitSettings(
	commands []string,
	message string,
	includeStdout bool,
	stdoutFilter, stderrFilter string,
	noStderr bool,
) error {
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
		commands, includeStdout, stdoutFilter, stderrFilter, noStderr, err = getInteractiveSettings()
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
	commandStr := buildCommandString(commands, message, includeStdout, stdoutFilter, stderrFilter, noStderr)

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
