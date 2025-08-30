package compositeactionlint

import "fmt"

type RuleBase struct {
	name string
	desc string
	errs []*Error
}

// Errs returns errors found by the rule.
func (r *RuleBase) Errs() []*Error { return r.errs }

// Name returns the name of the rule.
func (r *RuleBase) Name() string { return r.name }

// Description returns the description of the rule.
func (r *RuleBase) Description() string { return r.desc }

func (r *RuleBase) Errorf(pos *Pos, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.errs = append(r.errs, newError(msg, "", pos.Line, pos.Col, r.name))
}
func (r *RuleBase) Error(pos *Pos, msg string) {
	r.errs = append(r.errs, newError(msg, "", pos.Line, pos.Col, r.name))
}

type Rule interface {
	Pass
	Errs() []*Error
	Name() string
	Description() string
}
