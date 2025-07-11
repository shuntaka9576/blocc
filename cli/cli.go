package cli

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var Version string
var Revision = "HEAD"

var embedVersion = "0.1.1"

type VersionFlag string

func (v VersionFlag) Decode(_ *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                       { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, _ kong.Vars) error {
	if Version == "" {
		Version = embedVersion
	}
	fmt.Printf("blocc version %s (rev:%s)\n", Version, Revision)
	app.Exit(0)

	return nil
}

type CLI struct {
	Version  VersionFlag `name:"version" help:"Show version information" short:"v"`
	Commands []string    `arg:"" name:"commands" help:"Commands to execute" required:""`
	Parallel bool        `help:"Execute commands in parallel" short:"p"`
	Message  string      `help:"Custom error message" short:"m"`
}

func Parse() (*CLI, *kong.Context) {
	var cli CLI
	ctx := kong.Parse(&cli)
	return &cli, ctx
}
