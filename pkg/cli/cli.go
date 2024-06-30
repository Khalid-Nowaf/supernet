package cli

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

// ResolveCmd represents the command to resolve CIDR conflicts.
type Context struct {
	super *supernet.Supernet
}

var cli struct {
	Log     bool       `help:"Print the details about the inserted CIDR and the conflicts if any"`
	Resolve ResolveCmd `cmd:"" help:"Resolve CIDR conflicts"`
}

func NewCLI(super *supernet.Supernet) {
	ctx := kong.Parse(&cli, kong.UsageOnError())
	if cli.Log {
		super = supernet.WithSimpleLogger()(super)
	}
	if err := ctx.Run(&Context{super: super}); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
