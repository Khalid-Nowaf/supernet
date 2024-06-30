package supernet

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicsWithZeroCIDRMask(t *testing.T) {
	// Test with IPv4 zero mask
	_, cidrIPv4, _ := net.ParseCIDR("1.1.1.1/0")
	assert.Panics(t, func() {
		CidrToBits(cidrIPv4)
	}, "Should panic with IPv4 zero CIDR mask")

	// Test with IPv6 zero mask
	_, cidrIPv6, _ := net.ParseCIDR("2001:db8::ff00:42:8329/0")
	assert.Panics(t, func() {
		CidrToBits(cidrIPv6)
	}, "Should panic with IPv6 zero CIDR mask")
}

func TestCIDRToBitsConversion(t *testing.T) {
	testCases := []struct {
		cidr          string
		expectedBits  []int
		expectedDepth int
	}{
		{"1.1.1.1/8", []int{0, 0, 0, 0, 0, 0, 0, 1}, 7},
		{"3.1.1.1/8", []int{0, 0, 0, 0, 0, 0, 1, 1}, 7},
		{"2001:db8::ff00:42:8329/16", []int{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, 15},
	}

	for _, tc := range testCases {
		_, cidr, err := net.ParseCIDR(tc.cidr)
		bits, depth := CidrToBits(cidr)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedDepth, depth)
		assert.Equal(t, tc.expectedBits, bits)
	}
}

func TestBitsToCIDRConversion(t *testing.T) {
	testCases := []struct {
		cidr   string
		isIPv6 bool
	}{
		{"1.1.1.1/8", false},
		{"192.168.1.0/24", false},
		{"192.168.2.0/23", false},
		{"2001:db8::ff00:42:8329/16", true},
	}

	for _, tc := range testCases {
		_, cidr, _ := net.ParseCIDR(tc.cidr)
		bits, _ := CidrToBits(cidr)
		assert.Equal(t, cidr.String(), BitsToCidr(bits, tc.isIPv6).String())
	}
}

func TestTrieComparator(t *testing.T) {
	a := newPathNode()
	b := newPathNode()

	// Comparator scenarios
	comparisons := []struct {
		aPriority []uint8
		bPriority []uint8
		expected  bool
	}{
		{[]uint8{1, 1, 1}, []uint8{1, 1, 0}, true},
		{[]uint8{0, 1, 1}, []uint8{1, 0, 0}, false},
		{[]uint8{1, 1, 1}, []uint8{1, 1, 1}, true},
		{[]uint8{0, 0, 1}, []uint8{0, 1, 0}, false},
		{[]uint8{1, 0, 16}, []uint8{0, 0, 32}, true},
	}

	for _, comp := range comparisons {
		a.UpdateMetadata(&Metadata{Priority: comp.aPriority})
		b.UpdateMetadata(&Metadata{Priority: comp.bPriority})
		assert.Equal(t, comp.expected, DefaultComparator(a.Metadata(), b.Metadata()))
	}
}

func TestInsertAndRetrieveCidrs(t *testing.T) {
	super := NewSupernet()
	cidrs := []string{"1.1.1.1/8", "2.1.1.1/8", "3.1.1.1/8", "2001:db8::ff00:42:8329/16"}

	for _, cidrString := range cidrs {
		_, cidr, _ := net.ParseCIDR(cidrString)
		results := super.InsertCidr(cidr, nil)
		printPaths(super)
		printResults(results)
	}

	ipv4Results := []string{"1.0.0.0/8", "2.0.0.0/8", "3.0.0.0/8"}
	assert.ElementsMatch(t, ipv4Results, super.AllCidrsString(false), "IPv4 CIDR retrieval should match")

	ipv6ExpectedPath := []int{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.ElementsMatch(t, ipv6ExpectedPath, super.ipv6Cidrs.LeafsPaths()[0], "IPv6 path should match")
}

func TestEqualConflictLowPriory(t *testing.T) {

	root := NewSupernet()
	_, cidrHigh, _ := net.ParseCIDR("192.168.0.0/16")
	_, cidrLow, _ := net.ParseCIDR("192.168.0.0/16")

	root.InsertCidr(cidrHigh, &Metadata{Priority: []uint8{1}, originCIDR: cidrHigh, Attributes: makeCidrAtrr("high")})
	results := root.InsertCidr(cidrLow, &Metadata{Priority: []uint8{0}, originCIDR: cidrLow, Attributes: makeCidrAtrr("low")})
	printPaths(root)
	printResults(results)
	// subset
	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, root.AllCidrsString(false))

	assert.Equal(t, "high", root.ipv4Cidrs.Leafs()[0].Metadata().Attributes["cidr"])
}

func TestEqualConflictHighPriory(t *testing.T) {

	root := NewSupernet()
	_, cidrHigh, _ := net.ParseCIDR("192.168.0.0/16")
	_, cidrLow, _ := net.ParseCIDR("192.168.0.0/16")

	root.InsertCidr(cidrLow, &Metadata{Priority: []uint8{0}, originCIDR: cidrLow, Attributes: makeCidrAtrr("low")})
	result := root.InsertCidr(cidrHigh, &Metadata{Priority: []uint8{1}, originCIDR: cidrHigh, Attributes: makeCidrAtrr("high")})
	printResults(result)
	// subset
	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, root.AllCidrsString(false))

	assert.Equal(t, "high", root.ipv4Cidrs.Leafs()[0].Metadata().Attributes["cidr"])

}

func TestSubConflictLowPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(super, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(super.String())})
	root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(sub.String())})

	// subset
	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, root.AllCidrsString(false))
}

func TestSubConflictHighPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	results := root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, originCIDR: sub, Attributes: makeCidrAtrr(sub.String())})
	printPaths(root)
	printResults(results)
	allCidrs := root.AllCidrsString(false)

	assert.Equal(t, len(allCidrs), 24-16+1)
	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/17",
	}, root.AllCidrsString(false))
}
func TestSubConflictEqualPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, originCIDR: sub, Attributes: makeCidrAtrr(sub.String())})

	allCidrs := root.AllCidrsString(false)

	assert.Equal(t, 24-16+1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/17",
	}, allCidrs)
}
func TestSuperConflictLowPriority(t *testing.T) {
	root := NewSupernet()
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")
	_, super, _ := net.ParseCIDR("192.168.0.0/16")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

	allCidrs := root.AllCidrsString(false)

	assert.Equal(t, 24-16+1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/17",
	}, allCidrs)
}

func TestSuperConflictHighPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, originCIDR: sub, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{1}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})

	allCidrs := root.AllCidrsString(false)

	assert.Equal(t, 1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, allCidrs)
}

func TestSuperConflictEqualPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, originCIDR: sub, Attributes: makeCidrAtrr(sub.String())})
	result := root.InsertCidr(super, &Metadata{Priority: []uint8{0}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	printResults(result)
	allCidrs := root.AllCidrsString(false)

	assert.Equal(t, 24-16+1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/17",
	}, allCidrs)
}

func TestLookIPv4(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

	cidr, err := root.LookupIP("192.168.25.154")

	assert.NoError(t, err)
	assert.NotNil(t, cidr)
	assert.Equal(t, "192.168.16.0/20", cidr.String())
}

func TestLookIPv6(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("2001:db8:abcd:12::/64")
	_, sub, _ := net.ParseCIDR("2001:db8:abcd:12:1234::/80")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

	cidr, err := root.LookupIP("2001:0db8:abcd:12:1234::")

	assert.NoError(t, err)
	assert.NotNil(t, cidr)
	assert.Equal(t, "2001:db8:abcd:12:1234::/80", cidr.String())

	cidr, err = root.LookupIP("2001:db8:abcd:12:1234::abcd")

	assert.NoError(t, err)
	assert.NotNil(t, cidr)
	assert.Equal(t, "2001:db8:abcd:12:1234::/80", cidr.String())

	cidr, err = root.LookupIP("2001:db8:abcd:12:0000::1")

	assert.NoError(t, err)
	assert.NotNil(t, cidr)
	assert.Equal(t, "2001:db8:abcd:12::/68", cidr.String())

}

// func TestEqualConflictResults(t *testing.T) {
// 	root := NewSupernet()
// 	_, cidr1, _ := net.ParseCIDR("192.168.1.1/24")
// 	_, cidr2, _ := net.ParseCIDR("192.168.1.1/24")

// 	results := root.InsertCidr(cidr1, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(cidr1.String())})

// 	assert.Equal(t, len(results), 1)
// 	assert.Equal(t, cidr1.String(), results[0].CIDR.String())
// 	assert.Equal(t, NONE, results[0].ConflictType)
// 	assert.Equal(t, INSERT_NEW_CIDR, results[0].ResolutionAction)

// 	results = root.InsertCidr(cidr2, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(cidr2.String())})

// 	assert.Equal(t, results[0].ConflictType, EQUAL_CIDR)
// 	assert.Equal(t, results[0].ResolutionAction, REMOVE_EXISTING_CIDR)
// 	assert.Equal(t, len(results), 1)
// 	assert.Equal(t, cidr2.String(), results[0].CIDR.String())

// 	assert.Equal(t, NodeToCidr(&(results[0].RemovedCIDRs[0])), cidr1.String())
// 	assert.Equal(t, NodeToCidr(&(results[0].AddedCIDRs[0])), cidr2.String())
// }

// func TestSubConflictResults(t *testing.T) {
// 	root := NewSupernet()
// 	_, super, _ := net.ParseCIDR("192.168.0.0/16")
// 	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

// 	results := root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

// 	assert.Equal(t, len(results), 1)
// 	assert.Equal(t, super.String(), results[0].CIDR.String())
// 	assert.Equal(t, NONE, results[0].ConflictType)
// 	assert.Equal(t, INSERT_NEW_CIDR, results[0].ResolutionAction)

// 	results = root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
// 	allCidrs := root.getAllV4CidrsString(false)
// 	printPaths(root)
// 	printResults(results)
// 	assert.Equal(t, results[0].ConflictType, SUBCIDR)
// 	assert.Equal(t, results[0].ResolutionAction, SPLIT_EXISTING_CIDR)
// 	assert.Equal(t, len(results), 1)
// 	assert.Equal(t, sub.String(), results[0].CIDR.String())

// 	assert.Equal(t, len(results[0].AddedCIDRs), len(allCidrs))
// 	assert.Equal(t, len(results[0].RemovedCIDRs), 1)

// }

func TestSuperConflictResults(t *testing.T) {
	root := NewSupernet()
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")
	_, super, _ := net.ParseCIDR("192.168.0.0/16")

	results := root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, originCIDR: sub, Attributes: makeCidrAtrr(super.String())})

	assert.Equal(t, len(results.actions), 1)
	assert.Equal(t, sub.String(), results.CIDR.String())
	assert.Equal(t, NoConflict{}, results.ConflictType)
	assert.Equal(t, InsertNewCIDR{}, results.actions[0].Action)

	results = root.InsertCidr(super, &Metadata{Priority: []uint8{1}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	printResults(results)
	assert.Equal(t, results.ConflictType, SuperCIDR{})
	assert.Equal(t, results.actions[0].Action, RemoveExistingCIDR{})
	assert.Equal(t, results.actions[1].Action, InsertNewCIDR{})
	assert.Equal(t, 2, len(results.actions), "it should have 2 actions")
	assert.Equal(t, super.String(), results.CIDR.String())

	assert.Equal(t, 1, len(results.actions[1].AddedCidrs), "Added CIDR must be 1")
	assert.Equal(t, 1, len(results.actions[0].RemoveCidrs), "Removed CIDR must be 1")

}

func TestSuperConflictResultsWithSplit(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	results := root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, originCIDR: sub, Attributes: makeCidrAtrr(sub.String())})

	assert.Equal(t, len(results.actions), 1)
	assert.Equal(t, sub.String(), results.CIDR.String())
	assert.Equal(t, NoConflict{}, results.ConflictType)
	assert.Equal(t, InsertNewCIDR{}, results.actions[0].Action)

	results = root.InsertCidr(super, &Metadata{Priority: []uint8{0}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	printResults(results)
	assert.Equal(t, results.ConflictType, SuperCIDR{})
	assert.Equal(t, results.actions[0].Action, SplitInsertedCIDR{})
	assert.Equal(t, 1, len(results.actions), "it should have one result")

	assert.Equal(t, 8, len(results.actions[0].AddedCidrs), "Added CIDR must be 8")
	assert.Equal(t, 0, len(results.actions[0].RemoveCidrs), "Removed CIDR must be 0")

	addedCidrs := []string{}
	for _, added := range results.actions[0].AddedCidrs {
		addedCidrs = append(addedCidrs, NodeToCidr(&added))
	}

	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/17"}, addedCidrs)

}

func TestNestedConflictResolution1(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")

	deepCidrs := []struct {
		cidr       string
		priorities []uint8
	}{
		{cidr: "192.168.0.0/24", priorities: []uint8{3}},
		{cidr: "192.168.2.0/23", priorities: []uint8{1}},
		{cidr: "192.168.16.0/22", priorities: []uint8{1}},
		{cidr: "192.168.128.0/19", priorities: []uint8{3}},
		{cidr: "192.168.128.0/18", priorities: []uint8{3}},
	}

	for _, deepCidr := range deepCidrs {
		_, ipnet, _ := net.ParseCIDR(deepCidr.cidr)
		results := root.InsertCidr(ipnet, &Metadata{Priority: deepCidr.priorities, originCIDR: ipnet, Attributes: makeCidrAtrr(deepCidr.cidr)})
		printResults(results)
		printPaths(root)
	}
	results := root.InsertCidr(super, &Metadata{Priority: []uint8{2}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	printResults(results)
	printPaths(root)
	// THIS TEST IS A BIT NOSY, BLGTM
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 0 0] [192.168.0.0/24] -> from [192.168.0.0/24]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 0 1] [192.168.1.0/24] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 1] [192.168.2.0/23] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 1] [192.168.4.0/22] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 1] [192.168.8.0/21] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 1] [192.168.16.0/20] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 1] [192.168.32.0/19] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 1] [192.168.64.0/18] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 0 0] [192.168.128.0/19] -> from [192.168.128.0/19]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 0 1] [192.168.160.0/19] -> from [192.168.128.0/18]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 1] [192.168.192.0/18] -> from [192.168.0.0/16]
	//
	// we noticed subnet 17 could not make it becase of subnets in 192.0/18
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 1] 192.0/18
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0(1)] /17 the least significant bit is blocking 192.0/18
	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.192.0/18",
		"192.168.128.0/19",
		"192.168.160.0/19",
	}, root.AllCidrsString(false))
}

func TestNestedConflictResolution2(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")

	deepCidrs := []struct {
		cidr       string
		priorities []uint8
	}{
		{cidr: "192.168.0.0/24", priorities: []uint8{3}},
		{cidr: "192.168.2.0/23", priorities: []uint8{1}},
		{cidr: "192.168.16.0/22", priorities: []uint8{1}},
		{cidr: "192.168.128.0/19", priorities: []uint8{1}},
		{cidr: "192.168.128.0/18", priorities: []uint8{3}},
	}

	for _, deepCidr := range deepCidrs {
		_, ipnet, _ := net.ParseCIDR(deepCidr.cidr)
		results := root.InsertCidr(ipnet, &Metadata{Priority: deepCidr.priorities, originCIDR: ipnet, Attributes: makeCidrAtrr(deepCidr.cidr)})
		printResults(results)
		printPaths(root)
	}
	results := root.InsertCidr(super, &Metadata{Priority: []uint8{2}, originCIDR: super, Attributes: makeCidrAtrr(super.String())})
	printResults(results)
	printPaths(root)
	// THIS TEST IS A BIT NOSY, BLGTM
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 0 0] [192.168.0.0/24] -> from [192.168.0.0/24]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 0 1] [192.168.1.0/24] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 0 1] [192.168.2.0/23] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 0 1] [192.168.4.0/22] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 0 1] [192.168.8.0/21] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 0 1] [192.168.16.0/20] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 0 1] [192.168.32.0/19] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 0 1] [192.168.64.0/18] -> from [192.168.0.0/16]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 0] [192.168.128.0/18] -> from [192.168.128.0/18]
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 1] [192.168.192.0/18] -> from [192.168.0.0/16]
	//
	// we noticed subnet 17 could not make it because of subnets in 192.0/18
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0 1 1] 192.0/18
	// [1 1 0 0 0 0 0 0 1 0 1 0 1 0 0 0(1)] /17 the least significant bit is blocking 192.0/18
	assert.ElementsMatch(t, []string{
		"192.168.0.0/24",
		"192.168.1.0/24",
		"192.168.2.0/23",
		"192.168.4.0/22",
		"192.168.8.0/21",
		"192.168.16.0/20",
		"192.168.32.0/19",
		"192.168.64.0/18",
		"192.168.128.0/18",
		"192.168.192.0/18",
	}, root.AllCidrsString(false))
}

func TestRemoveBranchWithLimit(t *testing.T) {

	root := NewSupernet()
	deepCidrs := []struct {
		cidr       string
		priorities []uint8
	}{
		{cidr: "192.168.128.0/19", priorities: []uint8{1}},
		{cidr: "192.168.128.0/18", priorities: []uint8{3}},
	}

	for _, deepCidr := range deepCidrs {
		_, ipnet, _ := net.ParseCIDR(deepCidr.cidr)
		results := root.InsertCidr(ipnet, &Metadata{Priority: deepCidr.priorities, originCIDR: ipnet, Attributes: makeCidrAtrr(deepCidr.cidr)})
		printResults(results)
		printPaths(root)
	}

	assert.ElementsMatch(t, []string{"192.168.128.0/18"}, root.AllCidrsString(false))
}

func makeCidrAtrr(cidr string) map[string]string {
	attr := make(map[string]string)
	attr["cidr"] = cidr
	return attr
}

func printResults(results *InsertionResult) {
	fmt.Println(results.String())
}

func printPaths(root *Supernet) {
	for _, node := range root.ipv4Cidrs.Leafs() {
		if node.Metadata() != nil {
			fmt.Printf("%v [%s] -> from [%s]\n", node.Path(), BitsToCidr(node.Path(), false).String(), node.Metadata().Attributes["cidr"])
		} else {
			fmt.Printf("%v <-!!-- [%s] \n", node.Path(), BitsToCidr(node.Path(), false).String())
		}
	}
}
