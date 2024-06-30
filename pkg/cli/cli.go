package cli

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/khalid_nowaf/supernet/pkg/supernet"
)

// ResolveCmd represents the command to resolve CIDR conflicts.
type Context struct {
	super *supernet.Supernet
}

var cli struct {
	Resolve ResolveCmd `cmd:"" help:"Resolve CIDR conflicts"`
}

func NewCLI(super *supernet.Supernet) {
	ctx := kong.Parse(cli, kong.UsageOnError())
	if err := ctx.Run(&Context{super: super}); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
