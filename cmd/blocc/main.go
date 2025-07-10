package main

import (
	"github.com/shuntaka9576/blocc"
	"github.com/shuntaka9576/blocc/cli"
)

func main() {
	cliOptions, ctx := cli.Parse()

	executor := blocc.NewExecutor()

	var results []blocc.Result
	var err error

	if cliOptions.Parallel {
		results, err = executor.ExecuteParallel(cliOptions.Commands)
	} else {
		results, err = executor.ExecuteSequential(cliOptions.Commands)
	}

	if len(results) > 0 {
		blocc.OutputError(cliOptions.Message, results)
		ctx.Exit(2)
	}

	if err != nil {
		ctx.Exit(1)
	}
}
