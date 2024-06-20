package supernet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicOnMaskZero(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("1.1.1.1/0")
	assert.Panics(t, func() {
		cidrToBits(cidr)
	})

	_, cidr, _ = net.ParseCIDR("2001:db8::ff00:42:8329/0")

	assert.Panics(t, func() {
		cidrToBits(cidr)
	})

}

func TestCidrToBit(t *testing.T) {
	_, cidr, err := net.ParseCIDR("1.1.1.1/8")
	bits, depth := cidrToBits(cidr)
	assert.NoError(t, err)
	assert.Equal(t, 7, depth)
	assert.Equal(t, []int{0, 0, 0, 0, 0, 0, 0, 1}, bits)

	_, cidr, err = net.ParseCIDR("3.1.1.1/8")
	bits, depth = cidrToBits(cidr)
	assert.NoError(t, err)
	assert.Equal(t, 7, depth)
	assert.Equal(t, []int{0, 0, 0, 0, 0, 0, 1, 1}, bits)

	_, cidr, err = net.ParseCIDR("2001:db8::ff00:42:8329/16")

	bits, depth = cidrToBits(cidr)
	assert.NoError(t, err)
	assert.Equal(t, 15, depth)
	ipv6Path := []int{
		0, 0, 1, 0, // 2001
		0, 0, 0, 0, // 2001
		0, 0, 0, 0, // db8
		0, 0, 0, 1} // db8
	assert.Equal(t, ipv6Path, bits)
}

func TestBitsToCidr(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("1.1.1.1/8")
	bits, _ := cidrToBits(cidr)
	assert.Equal(t, cidr.String(), bitsToCidr(bits, false).String())

	_, cidr, _ = net.ParseCIDR("192.168.1.0/24")
	bits, _ = cidrToBits(cidr)
	assert.Equal(t, cidr.String(), bitsToCidr(bits, false).String())

	_, cidr, _ = net.ParseCIDR("192.168.2.0/23")
	bits, _ = cidrToBits(cidr)
	assert.Equal(t, cidr.String(), bitsToCidr(bits, false).String())

	_, cidr, _ = net.ParseCIDR("2001:db8::ff00:42:8329/16")
	bits, _ = cidrToBits(cidr)
	assert.Equal(t, cidr.String(), bitsToCidr(bits, true).String())

}

func TestCompactor(t *testing.T) {
	a := newPathTrie()
	b := newPathTrie()

	a.Metadata = &Metadata{Priority: []uint8{1, 1, 1}}
	b.Metadata = &Metadata{Priority: []uint8{1, 1, 0}}
	assert.True(t, comparator(a, b))

	a.Metadata = &Metadata{Priority: []uint8{0, 1, 1}}
	b.Metadata = &Metadata{Priority: []uint8{1, 0, 0}}
	assert.False(t, comparator(a, b))

	a.Metadata = &Metadata{Priority: []uint8{1, 1, 1}}
	b.Metadata = &Metadata{Priority: []uint8{1, 1, 1}}
	assert.True(t, comparator(a, b))

	a.Metadata = &Metadata{Priority: []uint8{0, 0, 1}}
	b.Metadata = &Metadata{Priority: []uint8{0, 1, 0}}
	assert.False(t, comparator(a, b))
}

func TestInsertSimple(t *testing.T) {
	super := NewSupernet()
	_, cidr, _ := net.ParseCIDR("1.1.1.1/8")
	_, cidr2, _ := net.ParseCIDR("2.1.1.1/8")
	_, cidr3, _ := net.ParseCIDR("3.1.1.1/8")

	super.InsertCidr(cidr, nil)
	super.InsertCidr(cidr2, nil)
	super.InsertCidr(cidr3, nil)

	assert.ElementsMatch(t, [][]int{
		{0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 1, 1},
	}, super.ipv4Cidrs.GetLeafsPaths())

	// TODO: IPV6
	assert.ElementsMatch(t, []string{
		"1.0.0.0/8",
		"2.0.0.0/8",
		"3.0.0.0/8",
	}, super.getAllV4Cidrs())

	_, cidr, _ = net.ParseCIDR("2001:db8::ff00:42:8329/16")

	super.InsertCidr(cidr, nil)
	ipv6Path := []int{
		0, 0, 1, 0, // 2001
		0, 0, 0, 0, // 2001
		0, 0, 0, 0, // db8
		0, 0, 0, 1} // db8
	assert.ElementsMatch(t, ipv6Path, super.ipv6Cidrs.GetLeafsPaths()[0])
	// assert.ElementsMatch(t, []string{}, super.getAllV4Cidrs())
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
	}, root.getAllV4Cidrs())
}

func TestSplitSuperAroundSub(t *testing.T) {
	//TODO: more testing is needed
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("1.0.0.0/8")

	root.InsertCidr(super, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(super.String())})
	// 0000 0001 /8
	subNode := root.ipv4Cidrs.GetLeafs()[0]
	// 0000 0 /5
	superNode := subNode.Parent.Parent.Parent

	newPath := [][]int{}
	for _, newNode := range splitSuperAroundSub(superNode, subNode, &Metadata{}) {
		newPath = append(newPath, newNode.GetPath())
	}

	assert.Equal(t, len(newPath), subNode.GetDepth()-superNode.GetDepth())
	assert.ElementsMatch(t, newPath, [][]int{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 1},
	})

}

func TestSuperConflictLowPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})
	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})

	allCidrs := root.getAllV4Cidrs()

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

	allCidrs := root.getAllV4Cidrs()

	assert.Equal(t, 1, len(allCidrs))

	assert.ElementsMatch(t, []string{
		"192.168.0.0/16",
	}, allCidrs)
}

func TestSubConflictHighPriority(t *testing.T) {
	root := NewSupernet()
	_, super, _ := net.ParseCIDR("192.168.0.0/16")
	_, sub, _ := net.ParseCIDR("192.168.1.1/24")

	root.InsertCidr(super, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(super.String())})
	root.InsertCidr(sub, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(sub.String())})

	allCidrs := root.getAllV4Cidrs()

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
	}, root.getAllV4Cidrs())
}

func makeCidrAtrr(cidr string) map[string]string {
	attr := make(map[string]string)
	attr["cidr"] = cidr
	return attr
}
