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

func OutputError(message string, results []Result) error {
	if message == "" {
		message = fmt.Sprintf("%d command(s) failed", len(results))
	}

	output := ErrorOutput{
		Message: message,
		Results: results,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal error output: %v\n", err)
		return err
	}

	fmt.Fprintln(os.Stderr, string(jsonBytes))
	return fmt.Errorf("commands failed")
}
