package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/khalid_nowaf/supernet/pkg/cli"
)

func main() {
	ctx := kong.Parse(&cli.CLI, kong.UsageOnError())
	if err := ctx.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
