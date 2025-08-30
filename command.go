package compositeactionlint

import (
	"flag"
	"fmt"
	"io"
)

const (
	ExitStatusSuccess           = 0
	ExitStatusProblemFound      = 1
	ExitStatusInvalidInvocation = 2
	ExitStatusFailure           = 3
)

type Command struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (cmd *Command) Main(args []string) int {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(cmd.Stderr)
	if err := flags.Parse(args[1:]); err != nil {
		return ExitStatusInvalidInvocation
	}

	l := &Linter{out: cmd.Stdout}
	errs, err := l.LintFiles(flags.Args())
	if err != nil {
		_, _ = fmt.Fprintln(cmd.Stderr, err.Error())
		return ExitStatusFailure
	}

	if len(errs) > 0 {
		return ExitStatusProblemFound
	}
	return ExitStatusSuccess
}
