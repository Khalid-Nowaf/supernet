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

func TestSuperConflict(t *testing.T) {
	super := NewSupernet()
	_, cidr2, _ := net.ParseCIDR("192.168.1.1/24")
	_, cidr1, _ := net.ParseCIDR("192.168.0.0/16")

	super.InsertCidr(cidr1, &Metadata{Priority: []uint8{0}, Attributes: makeCidrAtrr(cidr1.String())})
	super.InsertCidr(cidr2, &Metadata{Priority: []uint8{1}, Attributes: makeCidrAtrr(cidr2.String())})
	assert.Equal(t, 8, len(super.ipv4Cidrs.GetLeafsPaths()))
	assert.ElementsMatch(t, []string{}, super.getAllV4Cidrs())
}

func makeCidrAtrr(cidr string) map[string]string {
	attr := make(map[string]string)
	attr["cidr"] = cidr
	return attr
}