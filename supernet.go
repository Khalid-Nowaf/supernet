package supernet

import (
	"net"

	"github.com/khalid_nowaf/supernet/pkg/trie"
)

type ConflictType int

const (
	EQUAL_CIDR ConflictType = iota
	SUBCIDR
	SUPERCIR
	NONE
)

type ResolutionStrategy int

const (
	IGNORE_INSERTION ResolutionStrategy = iota
	EXCLUDE_CIDR
	REPLACE_CIDR
)

type Metadata struct {
	Priority   []uint8
	Attributes map[string]string
}

func NewDefaultMetadata() *Metadata {
	return &Metadata{}
}

type supernet struct {
	seedID    uint64
	ipv4Cidrs *trie.Trie[Metadata]
	ipv6Cidrs *trie.Trie[Metadata]
}

func NewSupernet() *supernet {
	return &supernet{
		ipv4Cidrs: &trie.Trie[Metadata]{},
		ipv6Cidrs: &trie.Trie[Metadata]{},
	}
}

func newPathTrie() *trie.Trie[Metadata] {
	return &trie.Trie[Metadata]{}
}

func (super *supernet) getAllV4Cidrs() []string {
	cidrs := []string{}
	for _, node := range super.ipv4Cidrs.GetLeafs() {
		cidrs = append(cidrs, bitsToCidr(node.GetPath(), false).String())
	}
	return cidrs
}

func (super *supernet) InsertCidr(ipnet *net.IPNet, metadata *Metadata) {
	var currentNode *trie.Trie[Metadata]
	var newCidrNode *trie.Trie[Metadata]

	if ipnet.IP.To4() != nil {
		currentNode = super.ipv4Cidrs
	} else if ipnet.IP.To16() != nil {
		currentNode = super.ipv6Cidrs
	}

	copyMetadata := metadata
	if copyMetadata == nil {
		copyMetadata = NewDefaultMetadata()
	}

	newCidrNode = trie.NewTrieWithMetadata(copyMetadata)

	path, depth := cidrToBits(ipnet)

	var supernetToSplitLater *trie.Trie[Metadata]

	for currentDepth, bit := range path {
		currentNode = currentNode.AddChildAtIfNotExist(newPathTrie(), bit)
		// we check conflict the child at the next bit
		switch isThereAConflict(currentNode, depth) {
		case EQUAL_CIDR:
			if isNewHasPriority := comparator(newCidrNode, currentNode); isNewHasPriority {
				currentNode.AddChildOrReplaceAt(newCidrNode, bit)
			}
			return // this is the last bit
		case SUBCIDR:
			if isNewHasPriority := comparator(newCidrNode, currentNode); isNewHasPriority {
				// since, we do not insert all the bits for newCidrNode
				// we will deal with conflict resolution later at the last bit
				supernetToSplitLater = currentNode
			} else {
				// since the currentNode is supernet and have a higher priority
				// we will simply ignore the inserting it
				return
			}
		case SUPERCIR:
			// we setup the new cidr, as marker, for now
			// it get splitted later or stay as is
			currentNode.Metadata = newCidrNode.Metadata
			conflictedCidrs := currentNode.GetLeafs()

			anyConflictedCidrHasPriority := false
			for _, conflictedCidr := range conflictedCidrs {
				if conflictedCidrHasPriority := comparator(conflictedCidr, newCidrNode); conflictedCidrHasPriority {
					splitSuperAroundSub(currentNode, conflictedCidr, newCidrNode.Metadata)
					anyConflictedCidrHasPriority = true
				}
			}

			// so if the new supernet wins over all conflicted
			// we detach the rest of the tree by making it a leaf
			if !anyConflictedCidrHasPriority {
				currentNode.AddChildOrReplaceAt(newCidrNode, bit)
			}
			return // last bit
		case NONE:
			// if this the last bit, and there is no conflict
			if currentDepth == depth {
				// we replaced the current path node, with new cidr
				added := currentNode.Parent.AddChildOrReplaceAt(newCidrNode, bit)
				// when the newCidr is a subnet of supernet
				// we need to do the split at the end of constructing
				// the newCidrNode
				if supernetToSplitLater != nil {
					splitSuperAroundSub(supernetToSplitLater, added, supernetToSplitLater.Metadata)
				}

				// sanity check
				if added != newCidrNode {
					panic("New CIDR is not added at the end")
				}

			} else {
				// TODO: explain more why we add path node
				// we always start by adding path node
				// currentNode = currentNode.AddChildAtIfNotExist(newPathTrie(), bit)
			}
		}
	}
}

func isThereAConflict(currentNode *trie.Trie[Metadata], targetedDepth int) ConflictType {

	// new brand node, or path node
	if currentNode.Metadata == nil {
		// if this is the last node,
		// we know other cidrs under us exists
		if targetedDepth == currentNode.GetDepth() && !currentNode.IsLeaf() {
			return ConflictType(SUPERCIR)
		} else {
			return ConflictType(NONE)
		}
	} else {
		if currentNode.GetDepth() == targetedDepth {
			return ConflictType(EQUAL_CIDR)
		}
		if currentNode.GetDepth() < targetedDepth {
			return ConflictType(SUBCIDR)
		}
	}

	// sanity check
	panic("Edge Case has not been covered (func isThereAConflict)")
}

func splitSuperAroundSub(super *trie.Trie[Metadata], sub *trie.Trie[Metadata], splittedCidrMetadata *Metadata) []*trie.Trie[Metadata] {

	if splittedCidrMetadata == nil {
		panic("you can not split a supernet without metadata")
	}
	splittedCidrs := []*trie.Trie[Metadata]{}

	sub.ForEachStepUp(func(current *trie.Trie[Metadata]) {
		// we try to branch (if the current at 0 we go to 1 and vice versa)
		// using XOR to add a new cidr as sibling
		// if the the sibling node exist we ignore the insertion
		parent := current.Parent

		newCidr := trie.NewTrieWithMetadata(&Metadata{
			Priority: splittedCidrMetadata.Priority,
			// TODO: add this
			Attributes: splittedCidrMetadata.Attributes,
		})

		added := parent.AddChildAtIfNotExist(newCidr, current.GetPos()^1)
		if added == newCidr {
			splittedCidrs = append(splittedCidrs, added)
		}
		// we break the propagation when we reach the super cidr
	}, func(nextNode *trie.Trie[Metadata]) bool {
		return nextNode.GetDepth() > super.GetDepth()
	})

	return splittedCidrs

}

// func LookupCidr(ip net.Addr) {}

// func GetCidrs() []net.IPAddr {}

// will return true of A has or equal propriety to B
// note: we assume A is a new, and B is old therefore we return true if they are equal
func comparator(a *trie.Trie[Metadata], b *trie.Trie[Metadata]) bool {
	// we assume a is greater, so of they are equal we return true
	result := true
	// what if [=,-]
	for i := range a.Metadata.Priority {
		if a.Metadata.Priority[i] < b.Metadata.Priority[i] {
			return false
		}
	}
	return result
}

func bitsToCidr(bits []int, ipV6 bool) *net.IPNet {
	var bitsLen = len(bits) - 1
	var maskByte byte
	var ipByte byte
	maxBytes := 4
	currentBit := 0
	ipBytes := []byte{}
	maskBytes := []byte{}

	if ipV6 {
		maxBytes = 16
	}

	for iByte := 0; iByte < maxBytes; iByte++ {
		ipByte = 0
		maskByte = 0
		for i := 0; i < 8; i++ {
			if currentBit <= bitsLen {
				ipByte = ipByte<<1 | byte(bits[currentBit])
				maskByte = maskByte<<1 | byte(1)
				currentBit++
			} else {
				ipByte = ipByte << 1
				maskByte = maskByte << 1
			}

		}
		ipBytes = append(ipBytes, ipByte)
		maskBytes = append(maskBytes, maskByte)
	}

	return &net.IPNet{
		IP:   ipBytes,
		Mask: maskBytes,
	}

}
func cidrToBits(ipnet *net.IPNet) ([]int, int) {
	if ipnet == nil {
		panic("IPNet is nil, validate the input")
	}
	depth, _ := ipnet.Mask.Size()
	path := make([]int, depth)

	// technically speaking /0 is a valid CIDR,
	// but I do not see any valid use case for it
	// in this library
	if depth == 0 {
		panic("Network Mask /0 not valid: " + ipnet.String())
	}

	currentBit := 0
	for _, b := range ipnet.IP {
		// since IP constructed as array of byte
		// we process each byte

		// for each byte we shift the whole byte to the right
		// so we make the most significant bit at the start
		// then we mask it with 1,
		// to get the value at the least significant bit
		//      0011 >> 3
		//    = 0001
		//    & 0001
		//    = 0001

		for j := 0; j < 8; j++ {
			bit := (b >> (7 - j)) & 1
			path[currentBit] = int(bit)
			// if we fill the needed bits based
			// on the network mask we return and skip
			// the rest of the loop
			if (len(path) - 1) == currentBit {
				return path, depth - 1
			}
			currentBit++

		}

	}
	// sanity check
	panic("bits has not fully calculated")
}
