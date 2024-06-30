package supernet

import (
	"fmt"
	"net"
	"strings"

	"github.com/khalid-nowaf/supernet/pkg/trie"
)

// holds the properties for a CIDR node
type Metadata struct {
	originCIDR *net.IPNet        // copy of the CIDR, to track it, if it get splitted later due to conflict resolution
	IsV6       bool              // is it IPv6 CIDR
	Priority   []uint8           // min value 0, max value 255, and all CIDR in the tree must have the same length
	Attributes map[string]string // generic key value attributes to hold additional information about the CIDR
}

// construct a Metadata for a cidr
func NewMetadata(ipnet *net.IPNet) *Metadata {
	isV6 := false
	if ipnet.IP.To4() == nil {
		isV6 = true
	}
	return &Metadata{
		originCIDR: ipnet,
		IsV6:       isV6,
	}
}

// Supernet represents a structure containing both IPv4 and IPv6 CIDRs, each stored in a separate trie.
type Supernet struct {
	ipv4Cidrs  *trie.BinaryTrie[Metadata]
	ipv6Cidrs  *trie.BinaryTrie[Metadata]
	comparator ComparatorOption
	logger     LoggerOption
}

// initializes a new supernet instance with separate tries for IPv4 and IPv6 CIDRs.
func NewSupernet(options ...Option) *Supernet {
	super := DefaultOptions()
	for _, option := range options {
		super = option(super)
	}
	return super
}

// InsertCidr attempts to insert a new CIDR into the supernet, handling conflicts according to predefined priorities.
// It traverses through the trie, adding new nodes as needed and resolving conflicts when they occur.
func (super *Supernet) InsertCidr(ipnet *net.IPNet, metadata *Metadata) *InsertionResult {

	root := super.ipv4Cidrs
	path, depth := CidrToBits(ipnet)
	copyMetadata := metadata
	if copyMetadata == nil {
		copyMetadata = NewMetadata(ipnet)
	}

	if ipnet.IP.To4() == nil {
		copyMetadata.IsV6 = true
		root = super.ipv6Cidrs
	}

	if copyMetadata.IsV6 {
		root = super.ipv6Cidrs
	}

	// add size of the subnet as priory
	copyMetadata.Priority = append(copyMetadata.Priority, uint8(depth))
	copyMetadata.originCIDR = ipnet
	results := super.insertLeaf(
		root,
		path,
		trie.NewTrieWithMetadata(copyMetadata),
	)
	super.logger(results)
	return results
}

// LookupIP searches for the closest matching CIDR for a given IP address within the supernet.
func (super *Supernet) LookupIP(ip string) (*net.IPNet, error) {
	// Determine if the IP is IPv4 or IPv6 based on the presence of a colon.
	isV6 := strings.Contains(ip, ":")
	mask := 32
	supernet := super.ipv4Cidrs

	if isV6 {
		mask = 128
		supernet = super.ipv6Cidrs
	}

	// Parse the IP address with a full netmask to form a valid CIDR for bit conversion.
	_, parsedIP, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ip, mask))
	if err != nil {
		return nil, err
	}

	ipBits, _ := CidrToBits(parsedIP)

	// Traverse the trie to find the most specific matching CIDR.
	for i, bit := range ipBits {
		if supernet == nil {
			// Return nil if no matching CIDR is found in the trie.
			return nil, nil
		} else if supernet.IsLeaf() {
			return BitsToCidr(ipBits[:i], isV6), nil
		} else {
			supernet = supernet.Child(bit)
		}
	}

	// The loop should always return before reaching this point.
	panic("[BUG] LookupIP: reached an unexpected state, the CIDR trie traversal should not get here.")
}

// retrieves all CIDRs from the specified IPv4 or IPv6 trie within a supernet.
func (super *Supernet) AllCIDRS(forV6 bool) []*trie.BinaryTrie[Metadata] {
	supernet := super.ipv4Cidrs
	if forV6 {
		supernet = super.ipv6Cidrs
	}
	return supernet.Leafs()
}

// retrieves all CIDRs from the specified IPv4 or IPv6 trie within a supernet.
func (super *Supernet) AllCidrsString(forV6 bool) []string {
	supernet := super.ipv4Cidrs
	if forV6 {
		supernet = super.ipv6Cidrs
	}
	var cidrs []string
	for _, node := range supernet.Leafs() {
		cidrs = append(cidrs, BitsToCidr(node.Path(), forV6).String())
	}
	return cidrs
}

// creates a new trie node intended for path utilization without any associated metadata.
func newPathNode() *trie.BinaryTrie[Metadata] {
	return &trie.BinaryTrie[Metadata]{}
}

// build the CIDR path, and report any conflict
func buildPath(root *trie.BinaryTrie[Metadata], path []int) (lastNode *trie.BinaryTrie[Metadata], conflict ConflictType, remainingPath []int) {
	currentNode := root
	for currentDepth, bit := range path {
		// add a pathNode, if the current node is nil
		currentNode = currentNode.AttachChild(newPathNode(), bit)

		conflictType := isThereAConflict(currentNode, len(path))

		// if the there is a conflict, return the conflicting point node, and the remaining bits (path)
		if _, noConflict := conflictType.(NoConflict); !noConflict {
			return currentNode, conflictType, path[currentDepth+1:]
		}
	}
	return currentNode, NoConflict{}, []int{} // empty
}

// try to build the CIDR path, and handle any conflict if any
func (super Supernet) insertLeaf(root *trie.BinaryTrie[Metadata], path []int, newCidrNode *trie.BinaryTrie[Metadata]) *InsertionResult {
	insertionResults := &InsertionResult{
		CIDR: newCidrNode.Metadata().originCIDR,
	}

	// buildPath will tell us the strategy to resolve the conflict if there is
	// any.
	lastNode, conflictType, remainingPath := buildPath(root, path)
	insertionResults.ConflictType = conflictType

	// based on the conflict we will get resolve
	// and the resolver will return a resolution plan for each conflict
	plan := conflictType.Resolve(lastNode, newCidrNode, super.comparator)
	insertionResults.ConflictedWith = append(insertionResults.ConflictedWith, plan.Conflicts...)

	for _, step := range plan.Steps {
		// each plan has an action has an excitor, and return an action result
		result := step.Action.Execute(newCidrNode, lastNode, step.TargetNode, remainingPath)
		insertionResults.actions = append(insertionResults.actions, result)
	}

	return insertionResults
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

// evaluates two trie nodes, `a` and `b`, to determine if the new node `a` should replace the old node `b`
// based on their priority values. It is assumed that `a` is the new node and `b` is the old node.
//
// Note:
//   - The function assumes that if all priorities of `a` are equal to `b`, then `a` should be greater than `b`.
//   - The priorities are compared in a lexicographical order, similar to comparing version numbers or tuples.
func DefaultComparator(a *Metadata, b *Metadata) bool {
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
