package supernet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicsWithZeroCIDRMask(t *testing.T) {
	// Test with IPv4 zero mask
	_, cidrIPv4, _ := net.ParseCIDR("1.1.1.1/0")
	assert.Panics(t, func() {
		cidrToBits(cidrIPv4)
	}, "Should panic with IPv4 zero CIDR mask")

	// Test with IPv6 zero mask
	_, cidrIPv6, _ := net.ParseCIDR("2001:db8::ff00:42:8329/0")
	assert.Panics(t, func() {
		cidrToBits(cidrIPv6)
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
		bits, depth := cidrToBits(cidr)
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
		bits, _ := cidrToBits(cidr)
		assert.Equal(t, cidr.String(), bitsToCidr(bits, tc.isIPv6).String())
	}
}

func TestTrieComparator(t *testing.T) {
	a := newPathTrie()
	b := newPathTrie()

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
	}

	for _, comp := range comparisons {
		a.Metadata = &Metadata{Priority: comp.aPriority}
		b.Metadata = &Metadata{Priority: comp.bPriority}
		assert.Equal(t, comp.expected, comparator(a, b))
	}
}

func TestInsertAndRetrieveCidrs(t *testing.T) {
	super := NewSupernet()
	cidrs := []string{"1.1.1.1/8", "2.1.1.1/8", "3.1.1.1/8", "2001:db8::ff00:42:8329/16"}

	for _, cidrString := range cidrs {
		_, cidr, _ := net.ParseCIDR(cidrString)
		super.InsertCidr(cidr, nil)
	}

	ipv4Results := []string{"1.0.0.0/8", "2.0.0.0/8", "3.0.0.0/8"}
	assert.ElementsMatch(t, ipv4Results, super.getAllV4Cidrs(false), "IPv4 CIDR retrieval should match")

	ipv6ExpectedPath := []int{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.ElementsMatch(t, ipv6ExpectedPath, super.ipv6Cidrs.GetLeafsPaths()[0], "IPv6 path should match")
}

func TestEqualConflictLowPriory(t *testing.T) {

	root := NewSupernet()
	_, cidrHigh, _ := net.ParseCIDR("192.168.0.0/16")
	_, cidrLow, _ := net.ParseCIDR("192.168.0.0/16")

	root.InsertCidr(cidrHigh, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr("high")})
	root.InsertCidr(cidrLow, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr("low")})

	// subset
	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, root.getAllV4Cidrs(false))

	assert.Equal(t, "high", root.ipv4Cidrs.GetLeafs()[0].Metadata.Attributes["cidr"])
}

func TestEqualConflictHighPriory(t *testing.T) {

	root := NewSupernet()
	_, cidrHigh, _ := net.ParseCIDR("192.168.0.0/16")
	_, cidrLow, _ := net.ParseCIDR("192.168.0.0/16")

	root.InsertCidr(cidrLow, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr("low")})
	root.InsertCidr(cidrHigh, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr("high")})
	// subset
	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, root.getAllV4Cidrs(false))

	assert.Equal(t, "high", root.ipv4Cidrs.GetLeafs()[0].Metadata.Attributes["cidr"])

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
	}, root.getAllV4Cidrs(false))
}

func TestSubConflictHighPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})
	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})

	allCidrs := root.getAllV4Cidrs(false)

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
	}, root.getAllV4Cidrs(false))
}
func TestSuperConflictLowPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

	allCidrs := root.getAllV4Cidrs(false)

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

	root.InsertCidr(sub, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(super.String())})

	allCidrs := root.getAllV4Cidrs(false)

	assert.Equal(t, 1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
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

func makeCidrAtrr(cidr string) map[string]string {
	attr := make(map[string]string)
	attr["cidr"] = cidr
	return attr
}
