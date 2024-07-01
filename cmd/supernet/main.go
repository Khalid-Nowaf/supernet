package main

import (
	"github.com/khalid-nowaf/supernet/pkg/cli"
	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

func main() {
	cli.NewCLI(supernet.NewSupernet())
}
