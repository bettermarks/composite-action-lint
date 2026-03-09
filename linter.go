package compositeactionlint

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Linter struct {
	out        io.Writer
	shellcheck string // empty = disabled; otherwise looked up via PATH or used as absolute path
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
		rules := []Rule{
			NewRuleExpression(),
		}
		if l.shellcheck != "" {
			if scPath, err := exec.LookPath(l.shellcheck); err == nil {
				rules = append(rules, NewRuleShellcheck(scPath))
			}
		}

		v := Visitor{}
		for _, rule := range rules {
			v.AddPass(rule)
		}
		if err := v.Visit(a); err != nil {
			return nil, err
		}
		for _, rule := range rules {
			errs := rule.Errs()
			all = append(all, errs...)
		}
	}

	for _, err := range all {
		err.Filepath = path
	}

	return all, nil
}

func (l *Linter) printErrors(errs []*Error, src []byte) {
	for _, err := range errs {
		err.PrettyPrint(l.out, src)
	}
}
