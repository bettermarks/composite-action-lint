package compositeactionlint

import "github.com/rhysd/actionlint"

type Pos = actionlint.Pos
type String = actionlint.String
type Bool = actionlint.Bool

// TODO: not sure if defining our own or reusing the actionlint job steps is
// best...

// Step is a step configuration in a composite action.
// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runssteps
type Step struct {
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsif
	If *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsid
	ID *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsname
	Name *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsenv
	Env map[string]*actionlint.EnvVar
	// Exec represents either a shell step or the use of another action
	Exec actionlint.Exec
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepscontinue-on-error
	ContinueOnError *Bool
	// Pos is the position of the step in the yaml source
	Pos *Pos
}

// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runs-for-composite-actions
type Runs struct {
	// Using indicates whether this is a composite, javascript or docker action
	Using *String
	// Steps is a list of steps that make up composite action, if this is one
	Steps []*Step
}

// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputs
type Input struct {
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_id
	ID *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_iddescription
	Description *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_idrequired
	Required *Bool
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_iddefault
	Default *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_iddeprecationmessage
	DeprecationMessage *String

	Pos *Pos
}

// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#outputs-for-composite-actions
type Output struct {
	ID          *String
	Description *String
	Value       *String
}

// ActionMetadata represent the structure of an action.yaml/action.yml
// https://docs.github.com/en/actions/creating-actions/metadata-syntax-for-github-actions
type ActionMetadata struct {
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#name
	Name *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#author
	Author *String
	// https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#description
	Description *String
	// Inputs is a map of parameters to the action
	Inputs map[string]*Input
	// Outpus is a map of outputs of the action
	Outputs map[string]*Output
	// Specifies what kind of action this is and contains steps for composite actions.
	Runs *Runs
}
