package supernet

import "github.com/khalid_nowaf/supernet/pkg/trie"

type ConflictType interface {
	String() string
	Resolve(conflictedCidr *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata]) *ResolutionPlan
}

type (
	NoConflict struct{} // there is no conflict
	EqualCIDR  struct{} // the new CIDR equal an existing CIDR
	SuperCIDR  struct{} // The new CIDR is a super CIDR of one or more existing sub CIDRs
	SubCIDR    struct{} // the new CIDR is a sub CIDR of an existing super CIDR
)

func (_ NoConflict) Resolve(at *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata]) *ResolutionPlan {
	plan := &ResolutionPlan{}
	plan.AddAction(InsertNewCIDR{}, at)
	return plan
}

func (_ NoConflict) String() string {
	return "No Conflict"
}

func (_ EqualCIDR) Resolve(conflictedCidr *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata]) *ResolutionPlan {
	plan := &ResolutionPlan{}
	plan.Conflicts = append(plan.Conflicts, *conflictedCidr)
	if comparator(newCidr.Metadata(), conflictedCidr.Metadata()) {

		plan.AddAction(RemoveExistingCIDR{}, conflictedCidr)
		plan.AddAction(InsertNewCIDR{}, conflictedCidr)
	} else {
		plan.AddAction(IgnoreInsertion{}, newCidr)

	}
	return plan
}

func (_ EqualCIDR) String() string {
	return "Equal CIDR"
}

func (_ SuperCIDR) Resolve(conflictPoint *trie.BinaryTrie[Metadata], newSuperCidr *trie.BinaryTrie[Metadata]) *ResolutionPlan {
	plan := &ResolutionPlan{}

	// since this is a super, we do not know how many subcidrs yet conflicting with this super
	// let us get all subCidrs
	conflictedSubCidrs := conflictPoint.Leafs()

	subCidrsWithLowPriority := []*trie.BinaryTrie[Metadata]{}
	subCidrsWithHighPriority := []*trie.BinaryTrie[Metadata]{}

	for _, conflictedSubCidr := range conflictedSubCidrs {
		plan.Conflicts = append(plan.Conflicts, *conflictedSubCidr)
		if comparator(newSuperCidr.Metadata(), conflictedSubCidr.Metadata()) {
			subCidrsWithLowPriority = append(subCidrsWithLowPriority, conflictedSubCidr)
			// new cidr has higher priority
		} else {
			subCidrsWithHighPriority = append(subCidrsWithHighPriority, conflictedSubCidr)
		}
	}

	// now we deal with conflicted cidrs that needed to be removed
	for _, toBeRemoved := range subCidrsWithLowPriority {
		plan.AddAction(RemoveExistingCIDR{}, toBeRemoved)
	}

	// then we split the removed cidrs
	for _, toBeSplittedAround := range subCidrsWithHighPriority {
		plan.AddAction(SplitInsertedCIDR{}, toBeSplittedAround)
	}

	// lastly, we can insert the new cidr without conflict
	if len(subCidrsWithHighPriority) == 0 {
		plan.AddAction(InsertNewCIDR{}, conflictPoint)
	}

	return plan
}

func (_ SuperCIDR) String() string {
	return "Super CIDR"
}

func (_ SubCIDR) Resolve(existingSuperCidr *trie.BinaryTrie[Metadata], newSubCidr *trie.BinaryTrie[Metadata]) *ResolutionPlan {
	plan := &ResolutionPlan{}
	plan.Conflicts = append(plan.Conflicts, *existingSuperCidr)
	// since this is a SubCidr, we have 2 option
	// - ignore it, if the SubCidr has low priority
	// - split the super around this subCidr if Subcidr has low priority

	if comparator(newSubCidr.Metadata(), existingSuperCidr.Metadata()) {
		// subcidr has higher priority
		plan.AddAction(InsertNewCIDR{}, newSubCidr)
		plan.AddAction(SplitExistingCIDR{}, existingSuperCidr)
		plan.AddAction(RemoveExistingCIDR{}, existingSuperCidr)
	} else {
		// subcidr has low priority
		plan.AddAction(IgnoreInsertion{}, newSubCidr)
	}
	return plan
}

func (_ SubCIDR) String() string {
	return "Sub CIDR"
}

// CIDR conflict detection, it check the current node if it conflicts with other CIDRS
func isThereAConflict(currentNode *trie.BinaryTrie[Metadata], targetedDepth int) ConflictType {
	// Check if the current node is a new or path node without specific metadata.
	if currentNode.Metadata() == nil {
		// Determine if the current node is a supernet of the targeted CIDR.
		if targetedDepth == currentNode.Depth() && !currentNode.IsLeaf() {
			return SuperCIDR{} // The node spans over the area of the new CIDR.
		} else {
			return NoConflict{} // No conflict detected.
		}
	} else {
		// Evaluate the relationship based on depths.
		if currentNode.Depth() == targetedDepth {
			return EqualCIDR{} // The node is at the same level as the targeted CIDR.
		}
		if currentNode.Depth() < targetedDepth {
			return SubCIDR{} // The node is a subnetwork of the targeted CIDR.
		}
	}

	// If none of the conditions are met, there's an unhandled case.
	panic("[BUG] isThereAConflict: unhandled edge case encountered")
}

// comparator evaluates two trie nodes, `a` and `b`, to determine if the new node `a` should replace the old node `b`
// based on their priority values. It is assumed that `a` is the new node and `b` is the old node.
//
// Note:
//   - The function assumes that if all priorities of `a` are equal to `b`, then `a` should be greater than `b`.
//   - The priorities are compared in a lexicographical order, similar to comparing version numbers or tuples.
func comparator(a *Metadata, b *Metadata) bool {
	// Compare priority values lexicographically.
	for i := range a.Priority {
		if a.Priority[i] > b.Priority[i] {

			// If any priority of 'a' is less than 'b', return false immediately.
			return true
		} else if a.Priority[i] < b.Priority[i] {
			return false
		}
	}
	// they are equal, so a is greater
	return true
}
