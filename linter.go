package compositeactionlint

import (
	"fmt"
	"io"
	"os"
)

type Linter struct {
	out io.Writer
}

func (l *Linter) LintFiles(paths []string) ([]*Error, error) {
	all := []*Error{}
	for _, path := range paths {
		errs, err := l.LintFile(path)
		if err != nil {
			return nil, err
		}
		all = append(all, errs...)
	}
	return all, nil
}

func (l *Linter) LintFile(path string) ([]*Error, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %q: %w", path, err)
	}
	errs, err := l.check(path, content)

	l.printErrors(errs, content)
	return errs, err
}

func (l *Linter) check(path string, content []byte) ([]*Error, error) {

	a, all := Parse(content)

	if a != nil {
		// TODO: run some check rules
	}
	return all, nil
}

func (l *Linter) printErrors(errs []*Error, src []byte) {
	for _, err := range errs {
		err.PrettyPrint(l.out, src)
	}
}
