package blocc

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
)

type Result struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
	Stderr   string `json:"stderr,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
}

type Executor struct {
	includeStdout bool
	stdoutFilter  string
	stderrFilter  string
	noStderr      bool
}

func NewExecutor(includeStdout bool, stdoutFilter, stderrFilter string, noStderr bool) *Executor {
	return &Executor{
		includeStdout: includeStdout,
		stdoutFilter:  stdoutFilter,
		stderrFilter:  stderrFilter,
		noStderr:      noStderr,
	}
}

func (e *Executor) ExecuteSequential(commands []string) ([]Result, error) {
	var failedResults []Result

	for _, cmdStr := range commands {
		result := e.executeCommand(cmdStr)
		if result.ExitCode != 0 {
			failedResults = append(failedResults, result)
			if result.ExitCode == 2 {
				// Exit immediately if exit code is 2
				return failedResults, nil
			}
		}
	}

	return failedResults, nil
}

func (e *Executor) ExecuteParallel(commands []string) ([]Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan Result, len(commands))
	var wg sync.WaitGroup

	for _, cmdStr := range commands {
		wg.Add(1)
		go func(cmd string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				result := e.executeCommand(cmd)
				if result.ExitCode == 2 {
					cancel() // Cancel other goroutines
				}
				resultChan <- result
			}
		}(cmdStr)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var failedResults []Result
	for result := range resultChan {
		if result.ExitCode != 0 {
			failedResults = append(failedResults, result)
		}
	}

	return failedResults, nil
}

func (e *Executor) executeCommand(cmdStr string) Result {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		result := Result{
			Command:  cmdStr,
			ExitCode: 1,
			Stderr:   "empty command",
		}
		if e.noStderr {
			result.Stderr = ""
		}
		return result
	}

	// #nosec G204 - This is a CLI tool designed to execute user-provided commands
	cmd := exec.Command(parts[0], parts[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Apply filters to outputs
	filteredStderr := e.applyFilter(stderr.String(), e.stderrFilter)
	filteredStdout := stdout.String()
	if e.includeStdout {
		filteredStdout = e.applyFilter(stdout.String(), e.stdoutFilter)
	}

	result := Result{
		Command:  cmdStr,
		ExitCode: 0,
		Stderr:   filteredStderr,
	}

	if e.noStderr {
		result.Stderr = ""
	}

	if e.includeStdout {
		result.Stdout = filteredStdout
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
			// If there's no stderr output, use the error message
			if result.Stderr == "" && !e.noStderr {
				result.Stderr = err.Error()
			}
		}
	}

	return result
}

func (e *Executor) applyFilter(input, filterCmd string) string {
	if filterCmd == "" || input == "" {
		return input
	}

	// #nosec G204 - This is a CLI tool designed to execute user-provided filter commands
	cmd := exec.Command("sh", "-c", filterCmd)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		// If filter command fails, return original input
		return input
	}

	return out.String()
}
