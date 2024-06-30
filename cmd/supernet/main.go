package main

import (
	"github.com/khalid_nowaf/supernet/pkg/cli"
	"github.com/khalid_nowaf/supernet/pkg/supernet"
)

func main() {
	cli.NewCLI(supernet.NewSupernet(supernet.WithSimpleLogger()))
}
