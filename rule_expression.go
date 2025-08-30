package compositeactionlint

import (
	"fmt"
	"strings"

	al "github.com/rhysd/actionlint"
)

type typedExpr struct {
	ty  ExprType
	pos Pos
}

type RuleExpression struct {
	RuleBase
	metadata *ActionMetadata
	inputsTy *ObjectType
	stepsTy  *ObjectType
}

func NewRuleExpression() *RuleExpression {
	return &RuleExpression{
		RuleBase: RuleBase{
			name: "expression",
			desc: "Syntax and semantics checks for expressions embedded with ${{ }} syntax",
		},
	}
}

func (rule *RuleExpression) VisitActionMetadataPre(node *ActionMetadata) error {
	// Get the correct context availability error messages
	UpdateSpecialFunctionNames()

	rule.checkString(node.Name, "")
	rule.checkString(node.Author, "")
	rule.checkString(node.Description, "")

	ity := al.NewEmptyStrictObjectType()
	for id, i := range node.Inputs {
		rule.checkString(i.Description, "")
		rule.checkBool(i.Required, "")
		rule.checkString(i.Default, "")
		rule.checkString(i.DeprecationMessage, "")

		// This might be untrue. It might be that they're all strings...
		var ty ExprType = AnyType{}
		ity.Props[id] = ty
	}
	rule.inputsTy = ity

	rule.metadata = node
	rule.stepsTy = al.NewEmptyStrictObjectType()
	return nil
}

func (rule *RuleExpression) VisitActionMetadataPost(n *ActionMetadata) error {

	// Now we've visited the steps, we know about the step outputs and
	// can check the output expressions

	for _, output := range n.Outputs {
		rule.checkString(output.Value, "outputs.<output_id>")
	}

	rule.stepsTy = nil

	return nil
}

func (rule *RuleExpression) VisitStep(n *Step) error {
	rule.checkString(n.Name, "runs.steps.name")
	rule.checkIfCondition(n.If, "runs.steps.if")

	var spec *String
	switch e := n.Exec.(type) {
	case *al.ExecRun:
		rule.checkScriptString(e.Run, "runs.steps.run")
		rule.checkString(e.Shell, "")
		rule.checkString(e.WorkingDirectory, "runs.steps.working-directory")
	case *al.ExecAction:
		rule.checkString(e.Uses, "")
		for n, i := range e.Inputs {
			if e.Uses != nil && strings.HasPrefix(e.Uses.Value, "actions/github-script@") && n == "script" {
				rule.checkScriptString(i.Value, "runs.steps.with")
			} else {
				rule.checkString(i.Value, "runs.steps.with")
			}
		}
		spec = e.Uses
	}

	rule.checkEnv(n.Env, "runs.steps.env")
	rule.checkBool(n.ContinueOnError, "runs.steps.continue-on-error")

	if n.ID != nil {
		if n.ID.ContainsExpression() {
			rule.checkString(n.ID, "")
			rule.stepsTy.Loose()
		}
		// Step ID is case insensitive
		id := strings.ToLower(n.ID.Value)
		rule.stepsTy.Props[id] = al.NewStrictObjectType(map[string]ExprType{
			"outputs":    rule.getActionOutputsType(spec),
			"conclusion": StringType{},
			"outcome":    StringType{},
		})
	}

	return nil
}

// Get type of `outputs.<output name>`
func (rule *RuleExpression) getActionOutputsType(spec *String) *ObjectType {
	if spec == nil {
		return al.NewMapObjectType(StringType{})
	}

	//if strings.HasPrefix(spec.Value, "./") {
	//	meta, _, err := rule.localActions.FindMetadata(spec.Value)
	//	if err != nil {
	//		rule.Error(spec.Pos, err.Error())
	//		return al.NewMapObjectType(StringType{})
	//	}
	//	if meta == nil {
	//		return al.NewMapObjectType(StringType{})
	//	}

	//	return typeOfActionOutputs(meta)
	//}

	// github-script action allows to set any outputs through calling `core.setOutput` directly.
	// So any `outputs.*` properties should be accepted (#104)
	if strings.HasPrefix(spec.Value, "actions/github-script@") {
		return al.NewEmptyObjectType()
	}

	// When the action run at this step is a popular action, we know what outputs are set by it.
	// Set the output names to `steps.{step_id}.outputs.{name}`.
	if meta, ok := al.PopularActions[spec.Value]; ok {
		return typeOfActionOutputs(meta)
	}

	return al.NewMapObjectType(StringType{})
}

func (rule *RuleExpression) checkIfCondition(str *String, workflowKey string) {
	if str == nil {
		return
	}

	// Note:
	// https://docs.github.com/en/actions/learn-github-actions/workflow-syntax-for-github-actions#jobsjob_idif
	//
	// > When you use expressions in an if conditional, you may omit the expression syntax (${{ }})
	// > because GitHub automatically evaluates the if conditional as an expression, unless the
	// > expression contains any operators. If the expression contains any operators, the expression
	// > must be contained within ${{ }} to explicitly mark it for evaluation.
	//
	// This document is actually wrong. I confirmed that any strings without surrounding in ${{ }}
	// are evaluated.
	//
	// - run: echo 'run'
	//   if: '!false'
	// - run: echo 'not run'
	//   if: '!true'
	// - run: echo 'run'
	//   if: false || true
	// - run: echo 'run'
	//   if: true && true
	// - run: echo 'not run'
	//   if: true && false

	var condTy ExprType
	if str.ContainsExpression() {
		ts := rule.checkString(str, workflowKey)

		if len(ts) == 1 {
			if str.IsExpressionAssigned() {
				condTy = ts[0].ty
			}
		}
	} else {
		src := str.Value + "}}" // }} is necessary since lexer lexes it as end of tokens
		line, col := str.Pos.Line, str.Pos.Col

		p := al.NewExprParser()
		expr, err := p.Parse(al.NewExprLexer(src))
		if err != nil {
			rule.exprError(err, line, col)
			return
		}

		if ty, ok := rule.checkSemanticsOfExprNode(expr, line, col, false, workflowKey); ok {
			condTy = ty
		}
	}

	if condTy != nil && !(BoolType{}).Assignable(condTy) {
		rule.Errorf(str.Pos, "\"if\" condition should be type \"bool\" but got type %q", condTy.String())
	}
}

func (rule *RuleExpression) checkTemplateEvaluatedType(ts []typedExpr) {
	for _, t := range ts {
		switch t.ty.(type) {
		case *ObjectType, *ArrayType, NullType:
			rule.Errorf(&t.pos, "object, array, and null values should not be evaluated in template with ${{ }} but evaluating the value of type %s", t.ty)
		}
	}
}

func (rule *RuleExpression) checkString(str *String, workflowKey string) []typedExpr {
	if str == nil {
		return nil
	}

	ts, ok := rule.checkExprsIn(str.Value, str.Pos, str.Quoted, false, workflowKey)
	if !ok {
		return nil
	}

	rule.checkTemplateEvaluatedType(ts)
	return ts
}

func (rule *RuleExpression) checkScriptString(str *String, workflowKey string) {
	if str == nil {
		return
	}

	ts, ok := rule.checkExprsIn(str.Value, str.Pos, str.Quoted, true, workflowKey)
	if !ok {
		return
	}

	rule.checkTemplateEvaluatedType(ts)
}

func (rule *RuleExpression) checkBool(b *Bool, workflowKey string) {
	if b == nil || b.Expression == nil {
		return
	}

	ty := rule.checkOneExpression(b.Expression, "bool value", workflowKey)
	if ty == nil {
		return
	}

	switch ty.(type) {
	case BoolType, AnyType:
		// ok
	default:
		rule.Errorf(b.Expression.Pos, "type of expression must be bool but found type %s", ty.String())
	}
}

func (rule *RuleExpression) checkExprsIn(s string, pos *Pos, quoted, checkUntrusted bool, workflowKey string) ([]typedExpr, bool) {
	// TODO: Line number is not correct when the string contains newlines.

	line, col := pos.Line, pos.Col
	if quoted {
		col++ // when the string is quoted like 'foo' or "foo", column should be incremented
	}
	offset := 0
	ts := []typedExpr{}
	for {
		idx := strings.Index(s, "${{")
		if idx == -1 {
			break
		}

		start := idx + 3 // 3 means removing "${{"
		s = s[start:]
		offset += start
		col := col + offset

		ty, offsetAfter, ok := rule.checkSemantics(s, line, col, checkUntrusted, workflowKey)
		if !ok {
			return nil, false
		}
		if ty == nil || offsetAfter == 0 {
			return nil, true
		}
		ts = append(ts, typedExpr{ty, Pos{Line: line, Col: col - 3}})

		s = s[offsetAfter:]
		offset += offsetAfter
	}

	return ts, true
}

func (rule *RuleExpression) exprError(err *ExprError, lineBase, colBase int) {
	pos := convertExprLineColToPos(err.Line, err.Column, lineBase, colBase)
	rule.Error(pos, err.Message)
}

func (rule *RuleExpression) checkSemanticsOfExprNode(expr ExprNode, line, col int, checkUntrusted bool, workflowKey string) (ExprType, bool) {
	var v []string
	//if rule.config != nil {
	//	v = rule.config.ConfigVariables
	//}
	c := al.NewExprSemanticsChecker(checkUntrusted, v)
	if rule.stepsTy != nil {
		c.UpdateSteps(rule.stepsTy)
	}
	if rule.inputsTy != nil {
		c.UpdateInputs(rule.inputsTy)
	}
	if workflowKey != "" {
		ctx, sp := MetadataKeyAvailability(workflowKey)
		if len(ctx) == 0 {
			// rule.Debug("No context availability was found for workflow key %q", workflowKey)
			panic(fmt.Errorf("no context availability was found for workflow key %q", workflowKey))
		}
		c.SetContextAvailability(ctx)
		c.SetSpecialFunctionAvailability(sp)
	}

	ty, errs := c.Check(expr)
	for _, err := range errs {
		rule.exprError(err, line, col)
	}

	return ty, len(errs) == 0
}

func (rule *RuleExpression) checkSemantics(src string, line, col int, checkUntrusted bool, workflowKey string) (ExprType, int, bool) {
	l := al.NewExprLexer(src)
	p := al.NewExprParser()
	expr, err := p.Parse(l)
	if err != nil {
		rule.exprError(err, line, col)
		return nil, l.Offset(), false
	}
	t, ok := rule.checkSemanticsOfExprNode(expr, line, col, checkUntrusted, workflowKey)
	return t, l.Offset(), ok
}

func (rule *RuleExpression) checkOneExpression(s *String, what, workflowKey string) ExprType {
	// checkString is not available since it checks types for embedding values into a string
	if s == nil {
		return nil
	}

	ts, ok := rule.checkExprsIn(s.Value, s.Pos, s.Quoted, false, workflowKey)
	if !ok {
		return nil
	}

	if len(ts) != 1 {
		// This case should be unreachable since only one ${{ }} is included is checked by parser
		rule.Errorf(s.Pos, "one ${{ }} expression should be included in %q value but got %d expressions", what, len(ts))
		return nil
	}

	return ts[0].ty
}

func (rule *RuleExpression) checkEnv(env map[string]*al.EnvVar, workflowKey string) {
	if env == nil {
		return
	}

	for _, e := range env {
		rule.checkString(e.Name, workflowKey)
		rule.checkString(e.Value, workflowKey)
	}
}

func convertExprLineColToPos(line, col, lineBase, colBase int) *Pos {
	// Line and column in ExprError are 1-based
	return &Pos{
		Line: line - 1 + lineBase,
		Col:  col - 1 + colBase,
	}
}

func typeOfActionOutputs(meta *al.ActionMetadata) *ObjectType {
	// Some action sets outputs dynamically. Such outputs are not defined in action.yml. actionlint
	// cannot check such outputs statically so it allows any props (#18)
	if meta.SkipOutputs {
		return al.NewEmptyObjectType()
	}
	props := make(map[string]ExprType, len(meta.Outputs))
	for n := range meta.Outputs {
		props[strings.ToLower(n)] = StringType{}
	}
	return al.NewStrictObjectType(props)
}
