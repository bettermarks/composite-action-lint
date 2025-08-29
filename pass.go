package compositeactionlint

type Pass interface {
	VisitStep(node *Step)
}
