package main

import (
	"fmt"
	"os"

	"github.com/shuntaka9576/blocc"
	"github.com/shuntaka9576/blocc/cli"
)

func main() {
	cliOptions, ctx := cli.Parse()

	if cliOptions.Init {
		if err := blocc.InitSettings(cliOptions.Commands, cliOptions.Message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			ctx.Exit(1)
		}
		ctx.Exit(0)
	}

	// Default behavior: run commands
	if len(cliOptions.Commands) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no commands provided\n")
		ctx.Exit(1)
	}

	executor := blocc.NewExecutor()

	var results []blocc.Result
	var err error

	if cliOptions.Parallel {
		results, err = executor.ExecuteParallel(cliOptions.Commands)
	} else {
		results, err = executor.ExecuteSequential(cliOptions.Commands)
	}

	if len(results) > 0 {
		options := blocc.OutputOptions{
			Message:       cliOptions.Message,
			IncludeStdout: cliOptions.Stdout,
		}
		if outputErr := blocc.OutputErrorWithOptions(options, results); outputErr != nil {
			ctx.Exit(1)
		}
		ctx.Exit(2)
	}

	if err != nil {
		ctx.Exit(1)
	}
}
