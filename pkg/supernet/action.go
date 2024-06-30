package supernet

import (
	"github.com/khalid-nowaf/supernet/pkg/trie"
)

type Action interface {
	Execute(newCidr *trie.BinaryTrie[Metadata], conflictedPoint *trie.BinaryTrie[Metadata], targetNode *trie.BinaryTrie[Metadata], remainingPath []int) *ActionResult
	String() string
}

type (
	IgnoreInsertion    struct{} // no op Action
	InsertNewCIDR      struct{} // Inserted the new CIDR `on` specific node
	RemoveExistingCIDR struct{} // remove existing CIDR `on` specific node
	SplitInsertedCIDR  struct{} // split the new CIDR `on` specific node
	SplitExistingCIDR  struct{} // split the existing CIDR `on` specific node
)

func (action IgnoreInsertion) Execute(_ *trie.BinaryTrie[Metadata], _ *trie.BinaryTrie[Metadata], _ *trie.BinaryTrie[Metadata], _ []int) *ActionResult {
	return &ActionResult{
		Action: action,
	}
}

func (_ IgnoreInsertion) String() string {
	return "Ignore Insertion"
}

func (action InsertNewCIDR) Execute(newCidr *trie.BinaryTrie[Metadata], conflictedPoint *trie.BinaryTrie[Metadata], _ *trie.BinaryTrie[Metadata], remainingPath []int) *ActionResult {

	actionResult := &ActionResult{
		Action: action,
	}

	// sanity checks
	if conflictedPoint == nil {
		panic("[BUG] Action[InsertNewCIDR].Execute:: conflictedPoint Node must not be nil")
	}
	if !conflictedPoint.IsLeaf() {
		panic("[BUG] Action[InsertNewCIDR].Execute:: conflictedPoint node must be a leaf")
	}

	lastNode, conflictType, _ := buildPath(conflictedPoint, remainingPath)
	if _, noConflict := conflictType.(NoConflict); !noConflict {
		panic("[BUG] Action[InsertNewCIDR].Execute:: Can not insert CIDR while there is a conflict unresolved")
	}

	// what if last node has metadata!
	if lastNode.Metadata() != nil {
		panic("[BUG] Action[InsertNewCIDR].Execute:: Last node must be path node without metadata")
	}
	lastNode.Parent().ReplaceChild(newCidr, lastNode.Pos())
	actionResult.appendAddedCidr(newCidr)
	return actionResult

}

func (_ InsertNewCIDR) String() string {
	return "Insert New CIDR"
}

func (action RemoveExistingCIDR) Execute(newCidr *trie.BinaryTrie[Metadata], _ *trie.BinaryTrie[Metadata], targetNode *trie.BinaryTrie[Metadata], _ []int) *ActionResult {

	actionResult := &ActionResult{
		Action: action,
	}

	actionResult.appendRemovedCidr(targetNode)
	newCidrDepth, _ := newCidr.Metadata().originCIDR.Mask.Size()

	if newCidrDepth >= targetNode.Depth() {
		targetNode.UpdateMetadata(nil)
	} else if newCidrDepth < targetNode.Depth() {
		targetNode.DetachBranch(newCidrDepth + 1)
	}

	return actionResult

}

func (_ RemoveExistingCIDR) String() string {
	return "Remove Existing CIDR"
}

func (action SplitInsertedCIDR) Execute(newCidr *trie.BinaryTrie[Metadata], conflictedPoint *trie.BinaryTrie[Metadata], targetNode *trie.BinaryTrie[Metadata], _ []int) *ActionResult {

	actionResult := &ActionResult{
		Action: action,
	}

	splittedCidr := splitAround(targetNode, newCidr.Metadata(), conflictedPoint.Depth())

	for _, addedCidr := range splittedCidr {
		actionResult.appendAddedCidr(addedCidr)
	}

	return actionResult

}

func (_ SplitInsertedCIDR) String() string {
	return "Split Inserted CIDR"
}

func (action SplitExistingCIDR) Execute(newCidr *trie.BinaryTrie[Metadata], conflictedPoint *trie.BinaryTrie[Metadata], targetNode *trie.BinaryTrie[Metadata], _ []int) *ActionResult {
	// init inserted result
	actionResult := &ActionResult{
		Action: action,
	}

	splittedCidrs := splitAround(newCidr, targetNode.Metadata(), conflictedPoint.Depth())

	for _, splittedCidr := range splittedCidrs {
		actionResult.appendAddedCidr(splittedCidr)
	}
	return actionResult

}

func (_ SplitExistingCIDR) String() string {
	return "Split Existing CIDR"
}

// to keep track of all removed CIDRs from resolving a conflict.
func (ar *ActionResult) appendRemovedCidr(cidr *trie.BinaryTrie[Metadata]) {
	ar.RemoveCidrs = append(ar.RemoveCidrs, *cidr)
}

// The function traverses from the sub-CIDR node upwards, attempting to insert a sibling node at each step.
// If a sibling node at a given position does not exist, it is created and added. The traversal and modifications
// stop when reaching the depth of the super-CIDR node.
func splitAround(sub *trie.BinaryTrie[Metadata], newCidrMetadata *Metadata, limitDepth int) []*trie.BinaryTrie[Metadata] {
	splittedCidrMetadata := newCidrMetadata

	if splittedCidrMetadata == nil {
		panic("[BUG] splitAround: Metadata is required to split a supernet")
	}

	var splittedCidrs []*trie.BinaryTrie[Metadata]

	sub.ForEachStepUp(func(current *trie.BinaryTrie[Metadata]) {

		// Create a new trie node with the same metadata as the splittedCidrMetadata.
		newCidr := trie.NewTrieWithMetadata(&Metadata{
			IsV6:       splittedCidrMetadata.IsV6,
			originCIDR: splittedCidrMetadata.originCIDR,
			Priority:   splittedCidrMetadata.Priority,
			Attributes: splittedCidrMetadata.Attributes,
		})

		added := current.AttachSibling(newCidr)

		if added == newCidr {
			// If the node was successfully added, append it to the list of split CIDRs.
			splittedCidrs = append(splittedCidrs, added)
		} else {
		}
	}, func(nextNode *trie.BinaryTrie[Metadata]) bool {
		// Stop propagation when reaching the depth of the super-CIDR.
		return nextNode.Depth() > limitDepth
	})

	return splittedCidrs
}
