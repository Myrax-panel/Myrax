package main

import (
	"os"

	"myrax/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
