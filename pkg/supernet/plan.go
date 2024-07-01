package supernet

type PlanStep struct {
	Action     Action
	TargetNode *CidrTrie
}
type ResolutionPlan struct {
	Conflicts []CidrTrie
	Steps     []*PlanStep
}

func (plan *ResolutionPlan) AddAction(action Action, on *CidrTrie) {
	plan.Steps = append(plan.Steps, &PlanStep{
		Action:     action,
		TargetNode: on,
	})
}
