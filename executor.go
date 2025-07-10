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
	Stderr   string `json:"stderr"`
	Stdout   string `json:"stdout,omitempty"`
}

type Executor struct {
}

func NewExecutor() *Executor {
	return &Executor{}
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
		return Result{
			Command:  cmdStr,
			ExitCode: 1,
			Stderr:   "empty command",
		}
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Command:  cmdStr,
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
			result.Stderr = err.Error()
		}
	}

	return result
}
