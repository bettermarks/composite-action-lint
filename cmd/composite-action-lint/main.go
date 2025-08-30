package main

import (
	"os"

	compositeactionlint "github.com/bettermarks/composite-action-lint"
)

func main() {
	cmd := compositeactionlint.Command{
		Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin,
	}
	os.Exit(cmd.Main(os.Args))
}
