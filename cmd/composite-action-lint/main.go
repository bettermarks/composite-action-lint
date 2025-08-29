package main

import (
	"os"

	compositeactionlint "github.com/bettermarks/composite-action-lint"
)

func main() {
	cmd := compositeactionlint.Command{}
	os.Exit(cmd.Main(os.Args))
}
