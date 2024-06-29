package supernet

import (
	"fmt"
	"net"
	"strings"

	"github.com/khalid_nowaf/supernet/pkg/trie"
)

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

func insertLeaf(root *trie.BinaryTrie[Metadata], path []int, newCidrNode *trie.BinaryTrie[Metadata]) *InsertionResult {
	insertionResults := &InsertionResult{
		CIDR: newCidrNode.Metadata().originCIDR,
	}

	// buildPath will tell us the strategy to resolve the conflict if there is
	// any.
	lastNode, conflictType, remainingPath := buildPath(root, path)
	insertionResults.ConflictType = conflictType

	// based on the conflict we will get resolve
	// and the resolver will return a resolution plan for each conflict
	plan := conflictType.Resolve(lastNode, newCidrNode)
	insertionResults.ConflictedWith = append(insertionResults.ConflictedWith, plan.Conflicts...)

	for _, step := range plan.Steps {
		// each plan has an action has an excitor, and return an action result
		result := step.Action.Execute(newCidrNode, lastNode, step.TargetNode, remainingPath)
		insertionResults.actions = append(insertionResults.actions, result)
	}

	return insertionResults
}

// records the outcome of attempting to insert a CIDR for reporting
type InsertionResult struct {
	CIDR           *net.IPNet                  // CIDR was attempted to be inserted.
	actions        []*ActionResult             // the result of each action is taken
	ConflictedWith []trie.BinaryTrie[Metadata] // array of conflicting nodes
	ConflictType                               // the type of the conflict
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

// holds the properties for a CIDR node within a trie, including IP version, priority, and additional attributes.
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

//	creates a Metadata instance with default values.
//
// Returns:
//   - A pointer to a Metadata instance initialized with default values.
// func NewDefaultMetadata() *Metadata {
// 	return &Metadata{}
// }

// Supernet represents a structure containing both IPv4 and IPv6 CIDRs, each stored in a separate trie.
type Supernet struct {
	ipv4Cidrs *trie.BinaryTrie[Metadata]
	ipv6Cidrs *trie.BinaryTrie[Metadata]
}

// initializes a new supernet instance with separate tries for IPv4 and IPv6 CIDRs.
//
// Returns:
//   - A pointer to a newly initialized supernet instance.
func NewSupernet() *Supernet {
	return &Supernet{
		ipv4Cidrs: &trie.BinaryTrie[Metadata]{},
		ipv6Cidrs: &trie.BinaryTrie[Metadata]{},
	}
}

// newPathNode creates a new trie node intended for path utilization without any associated metadata.
//
// Returns:
//   - A pointer to a newly created trie.BinaryTrie node with no metadata.
func newPathNode() *trie.BinaryTrie[Metadata] {
	return &trie.BinaryTrie[Metadata]{}
}

// retrieves all CIDRs from the specified IPv4 or IPv6 trie within a supernet.
//
// Parameters:
//   - forV6: A boolean flag if th CIDR is IPv6
//
// Returns:
//   - A slice of TrieNode, each representing a CIDR in the specified trie.
func (super *Supernet) GetAllV4Cidrs(forV6 bool) []*trie.BinaryTrie[Metadata] {
	supernet := super.ipv4Cidrs
	if forV6 {
		supernet = super.ipv6Cidrs
	}
	return supernet.GetLeafs()
}

// retrieves all CIDRs from the specified IPv4 or IPv6 trie within a supernet.
//
// Parameters:
//   - forV6: A boolean flag if th CIDR is IPv6
//
// Returns:
//   - A slice of strings, each representing a CIDR in the specified trie.
func (super *Supernet) getAllV4CidrsString(forV6 bool) []string {
	supernet := super.ipv4Cidrs
	if forV6 {
		supernet = super.ipv6Cidrs
	}
	var cidrs []string
	for _, node := range supernet.GetLeafs() {
		cidrs = append(cidrs, BitsToCidr(node.GetPath(), forV6).String())
	}
	return cidrs
}

// InsertCidr attempts to insert a new CIDR into the supernet, handling conflicts according to predefined priorities.
// It traverses through the trie, adding new nodes as needed and resolving conflicts when they occur.
//
// Parameters:
//   - ipnet: net.IPNet, the CIDR to be inserted.
//   - metadata: Metadata associated with the CIDR, used for conflict resolution and node creation.
//
// This function navigates through each bit of the new CIDR's path, trying to add a new node if it doesn't already exist,
// and handles various types of conflicts (EQUAL_CIDR, SUBCIDR, SUPERCIDR) by comparing the priorities of the involved CIDRs.

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
	results := insertLeaf(
		root,
		path,
		trie.NewTrieWithMetadata(copyMetadata),
	)

	return results
}

// Conflict Types:
//   - SUPERCIDR: The current node is a supernet relative to the targeted CIDR.
//   - SUBCIDR: The current node is a subnetwork relative to the targeted CIDR.
//   - EQUAL_CIDR: The current node and the targeted CIDR are at the same depth.
//   - NONE: There is no conflict.
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

// splitAround adjusts a super CIDR's trie structure around a specified sub CIDR by inserting sibling nodes.
// This process involves branching off at each step from the SUB-CIDR node upwards towards the SUPER-CIDR node,
// ensuring that the appropriate splits in the trie are made to represent the network structure correctly.
//
// Parameters:
//   - super: super CIDR.
//   - sub: sub CIDR to be surrounded.
//   - splittedCidrMetadata: Metadata for the new CIDR nodes that will be created during the split.
//
// Returns:
//   - A slice of pointers to nodes that were newly added during the splitting process.
//
// Panics:
//   - If splittedCidrMetadata is nil, as metadata is essential for creating new trie nodes.
//
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
			Attributes: splittedCidrMetadata.Attributes, // Additional attributes from metadata.
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

// LookupIP searches for the closest matching CIDR for a given IP address within the supernet.
//
// Parameters:
// - ip: A string representing the IP address
//
// Returns:
//   - net.IPNet representing the closest matching CIDR, if found, or nil
//   - An error if the IP address cannot be parsed
//
// The function parses the input IP address into a CIDR with a full netmask (32 for IPv4, 128 for IPv6).
// It then converts this CIDR into a slice of bits and traverses the corresponding trie (IPv4 or IPv6)
// to find the most specific matching CIDR. If the trie node representing the CIDR is a leaf or no further
// children exist for matching, the search concludes, returning the found CIDR or nil if no match exists.
func (super *Supernet) LookupIP(ip string) (*net.IPNet, error) {
	// Determine if the IP is IPv4 or IPv6 based on the presence of a colon.
	isV6 := strings.Contains(ip, ":")
	mask := 32
	supernet := super.ipv4Cidrs // Default to IPv4 supernet.

	if isV6 {
		mask = 128
		supernet = super.ipv6Cidrs // Use IPv6 supernet if the IP is IPv6.
	}

	// Parse the IP address with a full netmask to form a valid CIDR for bit conversion.
	_, parsedIP, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ip, mask))
	if err != nil {
		return nil, err // Return parsing errors.
	}

	ipBits, _ := CidrToBits(parsedIP) // Convert the parsed CIDR to a slice of bits.

	// Traverse the trie to find the most specific matching CIDR.
	for i, bit := range ipBits {
		if supernet == nil {
			// Return nil if no matching CIDR is found in the trie.
			return nil, nil
		} else if supernet.IsLeaf() {
			// Return the CIDR up to the current bit index if a leaf node is reached.
			return BitsToCidr(ipBits[:i], isV6), nil
		} else {
			// Move to the next child node based on the current bit.
			supernet = supernet.Child(bit)
		}
	}

	// The loop should always return before reaching this point.
	panic("[BUG] LookupIP: reached an unexpected state, the CIDR trie traversal should not get here.")
}

// comparator evaluates two trie nodes, `a` and `b`, to determine if the new node `a` should replace the old node `b`
// based on their priority values. It is assumed that `a` is the new node and `b` is the old node.
//
// Parameters:
//   - a:  new CIDR entrya.
//   - b:  the old CIDR entry.
//
// Returns:
//   - true if `a` should replace `b` or if they are considered equal in priority; false otherwise.
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
