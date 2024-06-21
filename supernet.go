package supernet

import (
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/khalid_nowaf/supernet/pkg/trie"
)

// ConflictType defines the types of conflicts that may arise during the insertion of new CIDRs into a trie structure.
//
// Types of conflicts:
//   - EQUAL_CIDR: Indicates a conflict where the new CIDR exactly matches an existing CIDR in the trie.
//   - SUBCIDR: Indicates the new CIDR is a subrange of an existing CIDR in the trie.
//   - SUPERCIDR: Indicates the new CIDR encompasses one or more existing CIDRs in the trie.
//   - NONE: Indicates no conflict with existing CIDRs in the trie.
type ConflictType int

const (
	EQUAL_CIDR ConflictType = iota
	SUBCIDR
	SUPERCIDR
	NONE
)

//	holds the properties for a CIDR node within a trie, including IP version, priority, and additional attributes.
//
// Properties:
//   - isV6: True if the CIDR is an IPv6 address
//   - Priority: An array of uint8 representing the priority of the CIDR which aids in conflict resolution.
//   - Attributes: A map of string keys to string values providing additional information about the CIDR.
type Metadata struct {
	isV6       bool
	Priority   []uint8
	Attributes map[string]string
}

//	creates a Metadata instance with default values.
//
// Returns:
//   - A pointer to a Metadata instance initialized with default values.
func NewDefaultMetadata() *Metadata {
	return &Metadata{}
}

// supernet represents a structure containing both IPv4 and IPv6 CIDRs, each stored in a separate trie.
type supernet struct {
	ipv4Cidrs *trie.BinaryTrie[Metadata]
	ipv6Cidrs *trie.BinaryTrie[Metadata]
}

// initializes a new supernet instance with separate tries for IPv4 and IPv6 CIDRs.
//
// Returns:
//   - A pointer to a newly initialized supernet instance.
func NewSupernet() *supernet {
	return &supernet{
		ipv4Cidrs: &trie.BinaryTrie[Metadata]{},
		ipv6Cidrs: &trie.BinaryTrie[Metadata]{},
	}
}

// newPathTrie creates a new trie node intended for path utilization without any associated metadata.
//
// Returns:
//   - A pointer to a newly created trie.BinaryTrie node with no metadata.
func newPathTrie() *trie.BinaryTrie[Metadata] {
	return &trie.BinaryTrie[Metadata]{}
}

// GetAllV4Cidrs retrieves all CIDRs from the specified IPv4 or IPv6 trie within a supernet.
//
// Parameters:
//   - forV6: A boolean flag if th CIDR is IPv6
//
// Returns:
//   - A slice of strings, each representing a CIDR in the specified trie.
func (super *supernet) getAllV4Cidrs(forV6 bool) []string {
	supernet := super.ipv4Cidrs
	if forV6 {
		supernet = super.ipv6Cidrs
	}
	var cidrs []string
	for _, node := range supernet.GetLeafs() {
		cidrs = append(cidrs, bitsToCidr(node.GetPath(), forV6).String())
	}
	return cidrs
}

// insertInit prepares the necessary nodes and metadata for inserting a new CIDR into the supernet.
//
// Parameters:
//   - ipnet: net.IPNet representing the CIDR to be inserted.
//   - metadata: Metadata associated with the CIDR. If nil, default metadata is used.
//
// Returns:
//   - currentNode: The root node of the appropriate trie (IPv4 or IPv6) based on the IP address in ipnet.
//   - newCidrNode: A new trie node initialized with the metadata intended for the new CIDR.
//   - path: A slice of integers representing the binary path derived from the CIDR.
//   - depth: The depth in the trie at which the CIDR should be inserted, determined by the number of bits in the CIDR's netmask.
func (super *supernet) insertInit(ipnet *net.IPNet, metadata *Metadata) (
	currentNode *trie.BinaryTrie[Metadata],
	newCidrNode *trie.BinaryTrie[Metadata],
	path []int,
	depth int,
) {
	// Create a copy of the provided metadata or initialize it with defaults if nil.
	copyMetadata := metadata
	if copyMetadata == nil {
		copyMetadata = NewDefaultMetadata()
	}

	// Determine the appropriate supernet (IPv4 or IPv6) based on the IP address format.
	if ipnet.IP.To4() != nil {
		// IPv4 CIDR.
		currentNode = super.ipv4Cidrs
	} else if ipnet.IP.To16() != nil {
		// IPv6 CIDR.
		currentNode = super.ipv6Cidrs
		copyMetadata.isV6 = true // Ensure metadata reflects the IP version.
	}

	// Initialize a new trie node with the copied or default metadata.
	newCidrNode = trie.NewTrieWithMetadata(copyMetadata)

	// Convert the CIDR to a binary path and calculate the depth.
	path, depth = cidrToBits(ipnet)
	return currentNode, newCidrNode, path, depth
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
func (super *supernet) InsertCidr(ipnet *net.IPNet, metadata *Metadata) {
	currentNode, newCidrNode, path, depth := super.insertInit(ipnet, metadata)
	var supernetToSplitLater *trie.BinaryTrie[Metadata]

	for currentDepth, bit := range path {
		currentNode = currentNode.AddChildAtIfNotExist(newPathTrie(), bit)

		switch isThereAConflict(currentNode, depth) {
		case EQUAL_CIDR:
			slog.Info("Conflict detected (EQUAL_CIDR)")
			if comparator(newCidrNode, currentNode) {
				currentNode.Parent.AddChildOrReplaceAt(newCidrNode, bit)
				slog.Info("New CIDR replaced existing CIDR: " + NodeToCidr(currentNode, metadata.isV6))
			} else {
				slog.Info("New CIDR ignored in favor of existing CIDR: " + NodeToCidr(currentNode, metadata.isV6))
			}
			return

		case SUBCIDR:
			slog.Info("Conflict detected (SUBCIDR)")
			if comparator(newCidrNode, currentNode) {
				supernetToSplitLater = currentNode
				slog.Info("New CIDR will split existing CIDR: " + NodeToCidr(currentNode, metadata.isV6))
			} else {
				slog.Info("New CIDR ignored in favor of existing CIDR: " + NodeToCidr(currentNode, metadata.isV6))
				return
			}

		case SUPERCIDR:
			slog.Info("Conflict detected (SUPERCIDR)")
			currentNode.Metadata = newCidrNode.Metadata
			conflictedCidrs := currentNode.GetLeafs()

			var anyConflictedCidrHasPriority bool
			for _, conflictedCidr := range conflictedCidrs {
				if comparator(conflictedCidr, newCidrNode) {
					slog.Info("New CIDR will be split around: " + NodeToCidr(conflictedCidr, metadata.isV6))
					newCidrs := splitSuperAroundSub(currentNode, conflictedCidr, newCidrNode.Metadata)
					for _, splittedCidr := range newCidrs {
						slog.Info("Splitted CIDRS: " + NodeToCidr(splittedCidr, metadata.isV6))
					}
					anyConflictedCidrHasPriority = true
				}
			}
			if !anyConflictedCidrHasPriority {
				currentNode.Parent.AddChildOrReplaceAt(newCidrNode, bit)
			} else {
				currentNode.Metadata = nil // Revert metadata change if new CIDR is not accepted.
			}
			return

		case NONE:
			if currentDepth == depth {
				added := currentNode.Parent.AddChildOrReplaceAt(newCidrNode, bit)
				if added != newCidrNode {
					panic("New CIDR failed to be added at the expected location.")
				}
				if supernetToSplitLater != nil {
					newCidrs := splitSuperAroundSub(supernetToSplitLater, added, supernetToSplitLater.Metadata)
					for _, splittedCidr := range newCidrs {
						slog.Info("Splitted CIDRS: " + NodeToCidr(splittedCidr, metadata.isV6))
					}
				}
			}
			// Continue traversal if no conflict and not the last bit.
		}
	}
}

// isThereAConflict determines if there is a conflict between the current trie node and a new CIDR insertion attempt,
// categorizing the conflict type based on the targeted depth and the current node's characteristics.
//
// Parameters:
//   - currentNode: current node in the trie.
//   - targetedDepth: The depth in the trie at which the new CIDR is intended to be inserted.
//
// Returns:
//   - A ConflictType value indicating the type of conflict, if any.
//
// The function evaluates the current node's metadata and its position in the trie relative to the targeted depth.
// It identifies if the current node represents a supernet, subnet, or equal CIDR conflict based on the insertion depth.
//
// Conflict Types:
//   - SUPERCIDR: The current node is a supernet relative to the targeted CIDR.
//   - SUBCIDR: The current node is a subnetwork relative to the targeted CIDR.
//   - EQUAL_CIDR: The current node and the targeted CIDR are at the same depth.
//   - NONE: There is no conflict.
func isThereAConflict(currentNode *trie.BinaryTrie[Metadata], targetedDepth int) ConflictType {
	// Check if the current node is a new or path node without specific metadata.
	if currentNode.Metadata == nil {
		// Determine if the current node is a supernet of the targeted CIDR.
		if targetedDepth < currentNode.GetDepth() && !currentNode.IsLeaf() {
			return ConflictType(SUPERCIDR) // The node spans over the area of the new CIDR.
		} else {
			return ConflictType(NONE) // No conflict detected.
		}
	} else {
		// Evaluate the relationship based on depths.
		if currentNode.GetDepth()-1 == targetedDepth {
			return ConflictType(EQUAL_CIDR) // The node is at the same level as the targeted CIDR.
		}
		if currentNode.GetDepth() < targetedDepth {
			return ConflictType(SUBCIDR) // The node is a subnetwork of the targeted CIDR.
		}
	}

	// If none of the conditions are met, there's an unhandled case.
	panic("isThereAConflict: unhandled edge case encountered")
}

// splitSuperAroundSub adjusts a super CIDR's trie structure around a specified sub CIDR by inserting sibling nodes.
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
func splitSuperAroundSub(super *trie.BinaryTrie[Metadata], sub *trie.BinaryTrie[Metadata], splittedCidrMetadata *Metadata) []*trie.BinaryTrie[Metadata] {
	if splittedCidrMetadata == nil {
		panic("Metadata is required to split a supernet")
	}

	var splittedCidrs []*trie.BinaryTrie[Metadata]

	sub.ForEachStepUp(func(current *trie.BinaryTrie[Metadata]) {
		// Determine the opposite position to branch at (XOR with 1).
		oppositePosition := current.GetPos() ^ 1
		parent := current.Parent

		// Create a new trie node with the same metadata as the splittedCidrMetadata.
		newCidr := trie.NewTrieWithMetadata(&Metadata{
			Priority:   splittedCidrMetadata.Priority,
			Attributes: splittedCidrMetadata.Attributes, // Additional attributes from metadata.
		})

		// Try to add the new node as a sibling if it does not already exist.
		added := parent.AddChildAtIfNotExist(newCidr, oppositePosition)
		if added == newCidr {
			// If the node was successfully added, append it to the list of split CIDRs.
			splittedCidrs = append(splittedCidrs, added)
		} else {
		}
	}, func(nextNode *trie.BinaryTrie[Metadata]) bool {
		// Stop propagation when reaching the depth of the super-CIDR.
		return nextNode.GetDepth() > super.GetDepth()
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
func (super *supernet) LookupIP(ip string) (*net.IPNet, error) {
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

	ipBits, _ := cidrToBits(parsedIP) // Convert the parsed CIDR to a slice of bits.

	// Traverse the trie to find the most specific matching CIDR.
	for i, bit := range ipBits {
		if supernet == nil {
			// Return nil if no matching CIDR is found in the trie.
			return nil, nil
		} else if supernet.IsLeaf() {
			// Return the CIDR up to the current bit index if a leaf node is reached.
			return bitsToCidr(ipBits[:i], isV6), nil
		} else {
			// Move to the next child node based on the current bit.
			supernet = supernet.GetChildAt(bit)
		}
	}

	// The loop should always return before reaching this point.
	panic("LookupIP: reached an unexpected state: the CIDR trie traversal should not get here.")
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
func comparator(a *trie.BinaryTrie[Metadata], b *trie.BinaryTrie[Metadata]) bool {
	// Default to true, assuming 'a' is equal or greater unless proven otherwise.
	result := true

	// Compare priority values lexicographically.
	for i := range a.Metadata.Priority {
		if a.Metadata.Priority[i] < b.Metadata.Priority[i] {
			// If any priority of 'a' is less than 'b', return false immediately.
			return false
		}
	}
	// If all priorities of 'a' are greater than or equal to those of 'b', return true.
	return result
}

// bitsToCidr converts a slice of binary bits into a net.IPNet structure that represents a CIDR.
// This is used to form the IP address and subnet mask from a binary representation.
//
// Parameters:
//   - bits: A slice of integers (0 or 1) representing the binary form of the IP address.
//   - ipV6: A boolean flag indicating whether the address is IPv6 (true) or IPv4 (false).
//
// Returns:
//   - A pointer to a net.IPNet structure that includes both the IP address and the subnet mask.
//
// This function dynamically constructs the IP and mask based on the length of the bits slice and the type of IP (IPv4 or IPv6).
// It supports a flexible number of bits and automatically adjusts for IPv4 (up to 32 bits) and IPv6 (up to 128 bits).
//
// Example:
//
//	For a bits slice representing "192.168.1.1" and ipV6 set to false, the function would return an IPNet with the IP "192.168.1.1"
//	and a full subnet mask "255.255.255.255" if all bits are provided.
func bitsToCidr(bits []int, ipV6 bool) *net.IPNet {
	maxBytes := 4
	if ipV6 {
		maxBytes = 16 // Set the byte limit to 16 for IPv6
	}

	ipBytes := make([]byte, 0, maxBytes)
	maskBytes := make([]byte, 0, maxBytes)
	currentBit := 0
	bitsLen := len(bits) - 1

	for iByte := 0; iByte < maxBytes; iByte++ {
		var ipByte byte
		var maskByte byte
		for i := 0; i < 8; i++ {
			if currentBit <= bitsLen {
				ipByte = ipByte<<1 | byte(bits[currentBit])
				maskByte = maskByte<<1 | 1 // Add a bit to the mask for each bit processed
				currentBit++
			} else {
				ipByte = ipByte << 1 // Shift the byte to the left, filling with zeros
				maskByte = maskByte << 1
			}
		}
		ipBytes = append(ipBytes, ipByte)
		maskBytes = append(maskBytes, maskByte)
	}

	return &net.IPNet{
		IP:   net.IP(ipBytes),
		Mask: net.IPMask(maskBytes),
	}
}

// NodeToCidr converts a given trie node into a CIDR (Classless Inter-Domain Routing) string representation.
// This function uses the node's path to generate the CIDR string.
//
// Parameters:
//   - t: Pointer to a trie.BinaryTrie node of type Metadata. It must contain valid metadata and a path.
//   - isV6: A boolean indicating whether the IP version is IPv6. True means IPv6, false means IPv4.
//
// Returns:
//   - A string representing the CIDR notation of the node's IP address.
//
// Panics:
//   - If the node's metadata is nil, indicating that it is a path node without associated CIDR data,
//     this function will panic with a specific error message.
//
// Example:
//
//	Given a trie node representing an IP address with metadata, this function will output the address in CIDR format,
//	 like "192.168.1.0/24" for IPv4 or "2001:db8::/32" for IPv6.
func NodeToCidr(t *trie.BinaryTrie[Metadata], isV6 bool) string {
	if t.Metadata == nil {
		panic("Cannot convert a trie path node to CIDR: metadata is missing")
	}
	// Convert the binary path of the trie node to CIDR format using the bitsToCidr function,
	// then convert the resulting net.IPNet object to a string.
	return bitsToCidr(t.GetPath(), isV6).String()
}

// cidrToBits converts a net.IPNet object into a slice of integers representing the binary bits of the network address.
// Additionally, it returns the depth of the network mask.
//
// The function panics if:
//   - ipnet is nil, indicating invalid input.
//   - the network mask is /0, which is technically valid but not supported by this library.
//
// Parameters:
//   - ipnet: Pointer to a net.IPNet object containing the IP address and the network mask.
//
// Returns:
//
//   - A slice of integers representing the binary format of the IP address up to the length of the network mask.
//
//   - An integer representing the number of bits in the network mask minus one.
//
//     Example:
//     For IP address "192.168.1.1/24", this function would return a slice with the first 24 bits of the address in binary form,
//     and the number 23 as the depth.
func cidrToBits(ipnet *net.IPNet) ([]int, int) {
	if ipnet == nil {
		panic("cidrToBits: IPNet is nil: validate the input before calling cidrToBits")
	}

	maskSize, _ := ipnet.Mask.Size()
	if maskSize == 0 {
		panic("cidrToBits: network Mask /0 not valid: " + ipnet.String())
	}

	path := make([]int, maskSize)
	currentBit := 0

	// Process each byte of the IP address to convert it into bits.
	for _, byteVal := range ipnet.IP {
		// Iterate over each bit in the byte.
		for bitPosition := 0; bitPosition < 8; bitPosition++ {
			// Shift the byte to the right to place the bit at the most significant position (leftmost),
			// and mask it with 1 to isolate the bit.
			bit := (byteVal >> (7 - bitPosition)) & 1
			path[currentBit] = int(bit)

			// If we have processed bits equal to the size of the network mask, return the result.
			if currentBit == (maskSize - 1) {
				return path, maskSize - 1
			}
			currentBit++
		}
	}

	// This line should not be reached; if it is, there is an error in bit calculation.
	panic("cidrToBits: bit calculation error - did not process enough bits for the mask size")
}
