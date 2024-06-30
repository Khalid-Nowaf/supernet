package supernet

import (
	"github.com/khalid-nowaf/supernet/pkg/trie"
)

type ConflictType interface {
	String() string
	Resolve(conflictedCidr *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata], comparator func(a *Metadata, b *Metadata) bool) *ResolutionPlan
}

type (
	NoConflict struct{} // there is no conflict
	EqualCIDR  struct{} // the new CIDR equal an existing CIDR
	SuperCIDR  struct{} // The new CIDR is a super CIDR of one or more existing sub CIDRs
	SubCIDR    struct{} // the new CIDR is a sub CIDR of an existing super CIDR
)

func (_ NoConflict) Resolve(at *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata], comparator func(a *Metadata, b *Metadata) bool) *ResolutionPlan {
	plan := &ResolutionPlan{}
	plan.AddAction(InsertNewCIDR{}, at)
	return plan
}

func (_ NoConflict) String() string {
	return "No Conflict"
}

func (_ EqualCIDR) Resolve(conflictedCidr *trie.BinaryTrie[Metadata], newCidr *trie.BinaryTrie[Metadata], comparator func(a *Metadata, b *Metadata) bool) *ResolutionPlan {
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

func (_ SuperCIDR) Resolve(conflictPoint *trie.BinaryTrie[Metadata], newSuperCidr *trie.BinaryTrie[Metadata], comparator func(a *Metadata, b *Metadata) bool) *ResolutionPlan {
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

func (_ SubCIDR) Resolve(existingSuperCidr *trie.BinaryTrie[Metadata], newSubCidr *trie.BinaryTrie[Metadata], comparator func(a *Metadata, b *Metadata) bool) *ResolutionPlan {
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
