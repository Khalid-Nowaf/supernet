package supernet

import (
	"fmt"
	"net"
)

// records the outcome of attempting to insert a CIDR for reporting
type InsertionResult struct {
	CIDR           *net.IPNet      // CIDR was attempted to be inserted.
	actions        []*ActionResult // the result of each action is taken
	ConflictedWith []CidrTrie      // array of conflicting nodes
	ConflictType                   // the type of the conflict
}

func (ir *InsertionResult) String() string {
	str := ""

	if _, ok := ir.ConflictType.(NoConflict); !ok {
		str += fmt.Sprintf("Detect %s conflict |", ir.ConflictType)
		str += fmt.Sprintf("New CIDR %s conflicted with [", ir.CIDR)
		for _, conflictedCidr := range ir.ConflictedWith {
			str += fmt.Sprintf("%s ", NodeToCidr(&conflictedCidr))
		}
		str += "] | "
	}

	for _, action := range ir.actions {
		str += fmt.Sprintf("%s", action.String())
	}

	return str
}

type ActionResult struct {
	Action      Action
	AddedCidrs  []CidrTrie
	RemoveCidrs []CidrTrie
}

func (ar ActionResult) String() string {
	addedCidrs := []string{}
	removedCidrs := []string{}

	for _, added := range ar.AddedCidrs {
		addedCidrs = append(addedCidrs, NodeToCidr(&added))
	}

	for _, removed := range ar.RemoveCidrs {
		removedCidrs = append(removedCidrs, NodeToCidr(&removed))
	}

	return fmt.Sprintf("Action Taken: %s, Added CIDRs: %v, Removed CIDRs: %v", ar.Action, addedCidrs, removedCidrs)
}

// to keep track of all the added CIDRs from resolving a conflict.
func (ar *ActionResult) appendAddedCidr(cidr *CidrTrie) {
	ar.AddedCidrs = append(ar.AddedCidrs, *cidr)
}
