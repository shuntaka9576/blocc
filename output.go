package blocc

import (
	"encoding/json"
	"fmt"
	"os"
)

type ErrorOutput struct {
	Message string   `json:"message"`
	Results []Result `json:"results"`
}

type OutputOptions struct {
	Message       string
	IncludeStdout bool
}

func OutputError(message string, results []Result) error {
	options := OutputOptions{
		Message:       message,
		IncludeStdout: false,
	}
	return OutputErrorWithOptions(options, results)
}

func OutputErrorWithOptions(options OutputOptions, results []Result) error {
	if options.Message == "" {
		options.Message = fmt.Sprintf("%d command(s) failed", len(results))
	}

	// Filter results based on IncludeStdout option
	filteredResults := make([]Result, len(results))
	for i, result := range results {
		filteredResults[i] = result
		if !options.IncludeStdout {
			filteredResults[i].Stdout = ""
		}
	}

	output := ErrorOutput{
		Message: options.Message,
		Results: filteredResults,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal error output: %v\n", err)
		return err
	}

	fmt.Fprintln(os.Stderr, string(jsonBytes))
	return nil
}
