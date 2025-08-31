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

func printUsageHeader(out io.Writer) {
	fmt.Fprintf(out, `usage: composite-action-lint FILES...

composite-action-lint is a linter for composite Github Actions.

To check any actions, pass the path to their metadata files as arguments.

  $ composite-action-lint path/to-action/action.yml another/action.yaml

It takes no options or configuration at present.
`)
}

type Command struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (cmd *Command) Main(args []string) int {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(cmd.Stderr)
	flags.Usage = func() {
		printUsageHeader(cmd.Stderr)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return ExitStatusSuccess
		}
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
