package compositeactionlint

type Pass interface {
	// VisitStep is a callback called when visiting a step of a composite
	// action. It is not called for non-composite actions
	VisitStep(node *Step) error
	// VisitActionMetadataPre is called before passing over the steps, if any
	VisitActionMetadataPre(node *ActionMetadata) error
	// VisitActionMetadataPost is called after passing over the steps, if any
	VisitActionMetadataPost(node *ActionMetadata) error
}

type Visitor struct {
	passes []Pass
}

func (v *Visitor) AddPass(p Pass) {
	v.passes = append(v.passes, p)
}

func (v *Visitor) Visit(n *ActionMetadata) error {

	for _, pass := range v.passes {
		if err := pass.VisitActionMetadataPre(n); err != nil {
			return err
		}
	}

	if n.Runs != nil {
		for _, step := range n.Runs.Steps {
			for _, pass := range v.passes {
				if err := pass.VisitStep(step); err != nil {
					return nil
				}
			}
		}
	}

	for _, pass := range v.passes {
		if err := pass.VisitActionMetadataPost(n); err != nil {
			return err
		}
	}
	return nil
}
