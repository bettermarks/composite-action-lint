package compositeactionlint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rhysd/actionlint"
	"gopkg.in/yaml.v3"
)

func nodeKindName(kind yaml.Kind) string {
	switch kind {
	case yaml.AliasNode:
		return "mapping"
	case yaml.DocumentNode:
		return "document"
	case yaml.MappingNode:
		return "mapping"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.ScalarNode:
		return "scalar"
	}
	return "<unknown, please raise bug>"
}

func posAt(n *yaml.Node) *Pos {
	return &Pos{Line: n.Line, Col: n.Column}
}

func isNull(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

func newString(n *yaml.Node) *String {
	quoted := n.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle) != 0
	return &String{Value: n.Value, Quoted: quoted, Pos: posAt(n)}
}

type workflowKeyVal struct {
	id  string
	key *String
	val *yaml.Node
}

type parser struct {
	errors []*Error
}

func (p *parser) error(n *yaml.Node, m string) {
	p.errors = append(p.errors, newError(m, "", n.Line, n.Column, "syntax-check"))
}
func (p *parser) errorAt(pos *Pos, m string) {
	p.errors = append(p.errors, newError(m, "", pos.Line, pos.Col, "syntax-check"))
}
func (p *parser) errorf(n *yaml.Node, format string, args ...any) {
	p.error(n, fmt.Sprintf(format, args...))
}
func (p *parser) errorfAt(pos *Pos, format string, args ...any) {
	p.errorAt(pos, fmt.Sprintf(format, args...))
}

func (p *parser) unexpectedKey(s *String, sec string, expected []string) {
	l := len(expected)
	var m string
	if l == 1 {
		m = fmt.Sprintf("expected %q key for %q section but got %q", expected[0], sec, s.Value)
	} else if l > 1 {
		m = fmt.Sprintf("unexpected key %q for %q section. expected one of [%v]", s.Value, sec, strings.Join(expected, ","))
	} else {
		m = fmt.Sprintf("unexpected key %q for %q section", s.Value, sec)
	}
	p.errorAt(s.Pos, m)
}

func (p *parser) checkNotEmpty(sec string, len int, n *yaml.Node) bool {
	if len == 0 {
		p.errorf(n, "%q section should not be empty", sec)
		return false
	}
	return true
}

func (p *parser) checkSequence(sec string, n *yaml.Node, allowEmpty bool) bool {
	if n.Kind != yaml.SequenceNode {
		p.errorf(n, "%q section must be sequence node but got %s node with %q tag", sec, nodeKindName(n.Kind), n.Tag)
		return false
	}
	return allowEmpty || p.checkNotEmpty(sec, len(n.Content), n)
}

func (p *parser) checkString(n *yaml.Node, allowEmpty bool) bool {
	if n.Kind != yaml.ScalarNode {
		p.errorf(n, "expected string but found %q node", nodeKindName(n.Kind))
		return false
	}
	if !allowEmpty && n.Value == "" {
		p.error(n, "string should not be empty")
		return false
	}
	return true
}

func (p *parser) missingExpression(n *yaml.Node, expecting string) {
	p.errorf(n, "expecting a single ${{...}} expression or %s, but found plain text node", expecting)
}

func (p *parser) parseExpression(n *yaml.Node, expecting string) *String {
	s := &String{Value: n.Value}
	if !s.IsExpressionAssigned() {
		p.missingExpression(n, expecting)
		return nil
	}
	return newString(n)
}

func (p *parser) parseString(n *yaml.Node, allowEmpty bool) *String {
	if !p.checkString(n, allowEmpty) {
		return &String{Value: "", Quoted: false, Pos: posAt(n)}
	}
	return newString(n)
}

func (p *parser) parseBool(n *yaml.Node) *Bool {
	if n.Kind != yaml.ScalarNode || (n.Tag != "!!bool" && n.Tag != "!!str") {
		p.errorf(n, "expected bool value but found %s node with %q tag", nodeKindName(n.Kind), n.Tag)
		return nil
	}

	if n.Tag == "!!str" {
		e := p.parseExpression(n, "boolean literal \"true\" or \"false\"")
		return &Bool{
			Expression: e,
			Pos:        posAt(n),
		}
	}

	return &Bool{
		Value: n.Value == "true",
		Pos:   posAt(n),
	}
}

func (p *parser) parseMapping(what string, n *yaml.Node, allowEmpty bool) []workflowKeyVal {
	isNull := isNull(n)

	if !isNull && n.Kind != yaml.MappingNode {
		p.errorf(n, "%s is %s node but mapping node is expected", what, nodeKindName(n.Kind))
		return nil
	}

	if !allowEmpty && isNull {
		p.errorf(n, "%s should not be empty. please remove this section if it's unnecessary", what)
		return nil
	}

	l := len(n.Content) / 2
	keys := make(map[string]*Pos, l)
	m := make([]workflowKeyVal, 0, l)
	for i := 0; i < len(n.Content); i += 2 {
		k := p.parseString(n.Content[i], false)
		if k == nil {
			continue
		}

		id := k.Value

		if pos, ok := keys[id]; ok {
			var note string
			p.errorfAt(k.Pos, "key %q is duplicated in %s. previously defined at %s%s", k.Value, what, pos.String(), note)
			continue
		}
		m = append(m, workflowKeyVal{id, k, n.Content[i+1]})
		keys[id] = k.Pos
	}

	if !allowEmpty && len(m) == 0 {
		p.errorf(n, "%s should not be empty. please remove this section if it's unnecessary", what)
	}

	return m
}

func (p *parser) parseStep(n *yaml.Node) *Step {
	ret := &Step{Pos: posAt(n)}

	run := &actionlint.ExecRun{}
	action := &actionlint.ExecAction{}

	var actionOnlyKey *String
	var runOnlyKey *String

	for _, kv := range p.parseMapping("input", n, false) {
		k, v := kv.key, kv.val
		switch kv.id {
		case "if":
			ret.If = p.parseString(v, false)
		case "id":
			ret.ID = p.parseString(v, false)
		case "name":
			ret.Name = p.parseString(v, false)
		case "continue-on-error":
			ret.ContinueOnError = p.parseBool(v)
		case "env":
			env := p.parseMapping("env", v, false)
			ret.Env = make(map[string]*actionlint.EnvVar, len(env))
			for _, envvar := range env {
				ret.Env[envvar.id] = &actionlint.EnvVar{
					Name: envvar.key, Value: p.parseString(envvar.val, true),
				}
			}
		case "uses":
			action.Uses = p.parseString(v, false)
			actionOnlyKey = k
		case "with":
			actionOnlyKey = k
			with := p.parseMapping("with", v, false)
			action.Inputs = make(map[string]*actionlint.Input, len(with))
			for _, input := range with {
				// In the actions metadata docs, entrypoint and args are not
				// mentioned. So, skipped here as yet.
				action.Inputs[input.id] = &actionlint.Input{
					Name: input.key, Value: p.parseString(input.val, true),
				}
			}
		case "run":
			run.RunPos = k.Pos
			run.Run = p.parseString(v, false)
			runOnlyKey = k
		case "shell":
			run.Shell = p.parseString(v, false)
			runOnlyKey = k
		case "working-directory":
			run.WorkingDirectory = p.parseString(v, false)
		default:
			p.unexpectedKey(k, "step", []string{
				"if",
				"id",
				"name",
				"continue-on-error",
				"env",
				"uses",
				"with",
				"run",
				"shell",
				"working-directory",
			})
		}
	}

	if actionOnlyKey != nil {
		ret.Exec = action
		if action.Uses == nil {
			p.error(n, "\"with\" without \"uses\"")
		}
		if runOnlyKey != nil {
			p.errorf(n, "step has both run and action keys, %q and %q",
				runOnlyKey.Value, actionOnlyKey.Value)
		}
	} else if runOnlyKey != nil {
		ret.Exec = run
		if run.Run == nil {
			p.error(n, "run step missing \"run\"")
		}
		if run.Shell == nil {
			p.error(n, "run step missing \"shell\"")
		}
	} else {
		p.error(n, "step missing both \"run\" and \"uses\"")
	}
	return ret
}

func (p *parser) parseSteps(n *yaml.Node) []*Step {
	if ok := p.checkSequence("steps", n, false); !ok {
		return nil
	}

	ret := make([]*Step, 0, len(n.Content))

	for _, c := range n.Content {
		if s := p.parseStep(c); s != nil {
			ret = append(ret, s)
		}
	}

	return ret
}

func (p *parser) parseOutput(id *String, n *yaml.Node) *Output {
	ret := &Output{ID: id}
	for _, kv := range p.parseMapping("output", n, false) {
		k, v := kv.key, kv.val
		switch kv.id {
		case "description":
			ret.Description = p.parseString(v, false)
		case "value":
			ret.Value = p.parseString(v, true)
		default:
			p.unexpectedKey(k, "output", []string{
				"description",
				"value",
			})
		}
	}
	if ret.Description == nil {
		p.errorfAt(id.Pos, "\"description\" is missing for output %q", id.Value)
	}
	return ret
}

// Only applies if action is composite
func (p *parser) postCheckOutput(o *Output) {
	if o.Value == nil {
		p.errorfAt(o.ID.Pos, "\"value\" is missing for output %q", o.ID.Value)
	}
}

func (p *parser) parseOutputs(n *yaml.Node) map[string]*Output {
	outputs := p.parseMapping("outputs section", n, false)
	ret := make(map[string]*Output, len(outputs))
	for _, kv := range outputs {
		ret[kv.id] = p.parseOutput(kv.key, kv.val)
	}
	return ret
}

func (p *parser) parseInput(id *String, n *yaml.Node) *Input {
	i := &Input{ID: id, Pos: id.Pos}
	for _, kv := range p.parseMapping("input", n, false) {
		_, v := kv.key, kv.val
		switch kv.id {
		case "description":
			i.Description = p.parseString(v, false)
		case "required":
			i.Required = p.parseBool(v)
		case "default":
			i.Default = p.parseString(v, true)
		case "deprecationMessage":
			i.DeprecationMessage = p.parseString(v, false)
		}
	}
	if i.Description == nil {
		p.errorfAt(i.Pos, "\"description\" is from input %q", id.Value)
	}
	// TODO: Not sure it's allowable to use expessions in the input spec
	// So maybe we should check for that.
	return i
}

func (p *parser) parseInputs(n *yaml.Node) map[string]*Input {
	inputs := p.parseMapping("inputs section", n, false)
	ret := make(map[string]*Input, len(inputs))
	for _, kv := range inputs {
		ret[kv.id] = p.parseInput(kv.key, kv.val)
	}
	return ret
}

func (p *parser) parseRuns(pos *Pos, n *yaml.Node) *Runs {
	ret := &Runs{}
	var stepsPos *Pos
	for _, kv := range p.parseMapping("runs section", n, false) {
		_, v := kv.key, kv.val
		switch kv.id {
		case "using":
			ret.Using = p.parseString(v, false)
		case "steps":
			ret.Steps = p.parseSteps(v)
			stepsPos = kv.key.Pos
		}
	}

	if ret.Using == nil {
		p.errorAt(pos, "\"using\" is missing from runs section")
	} else {
		// We don't check the using value exhaustively because new values like
		// 'node24' may appear.
		if ret.Using.Value == "composite" && ret.Steps == nil {
			p.errorAt(pos, "\"steps\" missing from composite action \"runs\" section")
		}
		if ret.Steps != nil && ret.Using.Value != "composite" {
			p.errorfAt(stepsPos,
				"unexpected \"steps\" section for non-composite %q action ",
				ret.Using.Value)
		}
	}
	return ret
}

func (p *parser) parse(n *yaml.Node) *ActionMetadata {
	a := &ActionMetadata{}

	if n.Line == 0 {
		n.Line = 1
	}
	if n.Column == 0 {
		n.Column = 1
	}

	if len(n.Content) == 0 {
		p.error(n, "action metadata file is empty")
		return a
	}

	for _, kv := range p.parseMapping("action metadata", n.Content[0], false) {
		k, v := kv.key, kv.val
		switch kv.id {
		case "name":
			a.Name = p.parseString(v, false)
		case "author":
			a.Author = p.parseString(v, false)
		case "description":
			a.Description = p.parseString(v, false)
		case "inputs":
			a.Inputs = p.parseInputs(v)
		case "outputs":
			a.Outputs = p.parseOutputs(v)
		case "runs":
			a.Runs = p.parseRuns(k.Pos, v)
		default:
			p.unexpectedKey(k, "action metadata", []string{
				"name",
				"author",
				"description",
				"inputs",
				"outputs",
				"runs",
				"branding", // Not parsed
			})
		}
	}

	if a.Name == nil {
		p.error(n, "\"name\" is missing in action metadata")
	}
	if a.Description == nil {
		p.error(n, "\"description\" is missing in action metadata")
	}
	if a.Runs == nil {
		p.error(n, "\"runs\" section is missing in action metadata")
	} else if a.Runs.Steps != nil {
		for _, o := range a.Outputs {
			p.postCheckOutput(o)
		}
	}
	return a
}

func handleYAMLError(err error) []*Error {
	re := regexp.MustCompile(`\bline (\d+):`)

	yamlErr := func(msg string) *Error {
		l := 0
		if ss := re.FindStringSubmatch(msg); len(ss) > 1 {
			l, _ = strconv.Atoi(ss[1])
		}
		msg = fmt.Sprintf("could not parse as YAML: %s", msg)
		return newError(msg, "", l, 0, "syntax-check")
	}

	if te, ok := err.(*yaml.TypeError); ok {
		errs := make([]*Error, 0, len(te.Errors))
		for _, msg := range te.Errors {
			errs = append(errs, yamlErr(msg))
		}
		return errs
	}

	return []*Error{yamlErr(err.Error())}
}

func Parse(b []byte) (*ActionMetadata, []*Error) {
	var n yaml.Node

	if err := yaml.Unmarshal(b, &n); err != nil {
		return nil, handleYAMLError(err)
	}

	p := &parser{}
	w := p.parse(&n)

	return w, p.errors
}
