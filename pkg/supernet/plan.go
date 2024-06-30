package supernet

import "github.com/khalid-nowaf/supernet/pkg/trie"

type PlanStep struct {
	Action     Action
	TargetNode *trie.BinaryTrie[Metadata]
}
type ResolutionPlan struct {
	Conflicts []trie.BinaryTrie[Metadata]
	Steps     []*PlanStep
}

func (plan *ResolutionPlan) AddAction(action Action, on *trie.BinaryTrie[Metadata]) {
	plan.Steps = append(plan.Steps, &PlanStep{
		Action:     action,
		TargetNode: on,
	})
}
