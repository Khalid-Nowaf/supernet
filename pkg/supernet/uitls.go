package supernet

import (
	"net"

	"github.com/khalid_nowaf/supernet/pkg/trie"
)

// BitsToCidr converts a slice of binary bits into a net.IPNet structure that represents a CIDR.
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
func BitsToCidr(bits []int, ipV6 bool) *net.IPNet {
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
func NodeToCidr(t *trie.BinaryTrie[Metadata]) string {
	if t.Metadata() == nil {
		panic("[Bug] NodeToCidr: Cannot convert a trie path node to CIDR, metadata is missing")
	}
	// Convert the binary path of the trie node to CIDR format using the bitsToCidr function,
	// then convert the resulting net.IPNet object to a string.
	return BitsToCidr(t.GetPath(), t.Metadata().IsV6).String()
}

// CidrToBits converts a net.IPNet object into a slice of integers representing the binary bits of the network address.
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
func CidrToBits(ipnet *net.IPNet) ([]int, int) {
	if ipnet == nil {
		panic("[BUG] cidrToBits: IPNet is nil: validate the input before calling cidrToBits")
	}

	maskSize, _ := ipnet.Mask.Size()
	if maskSize == 0 {
		panic("[BUG] cidrToBits: network Mask /0 not valid: " + ipnet.String())
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
	panic("[BUG] cidrToBits: bit calculation error - did not process enough bits for the mask size")
}
