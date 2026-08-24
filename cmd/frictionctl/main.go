package main

import (
	"os"

	"github.com/leeclarkuk/frictionctl/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
