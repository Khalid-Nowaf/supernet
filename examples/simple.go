package main

import (
	"fmt"
	"net"

	"github.com/khalid-nowaf/supernet/pkg/supernet"
)

type Network struct {
	cidr       string
	name       string
	priorities []uint8
}

func main() {

	// create random Cidrs
	networks := []*Network{
		{cidr: "123.123.123.0/24", name: "maybe my home network", priorities: []uint8{0, 0, 1}},
		{cidr: "123.123.123.0/23", name: "could be my home network", priorities: []uint8{0, 0, 2}},
		{cidr: "123.123.123.0/30", name: "it is my home network", priorities: []uint8{0, 0, 3}},
	}

	super := supernet.NewSupernet()
	for _, network := range networks {
		if _, ipnet, err := net.ParseCIDR(network.cidr); err == nil {
			//
			metadata := supernet.NewMetadata(ipnet)
			// attributes is used to store any additional data about the network
			metadata.Attributes["name"] = network.name
			// optional, it will be used to resolve conflict
			// if no priority add. the size of the network e.g /32, will be used as priority
			// so it is grunted smaller network will be not be over taken by larger network
			metadata.Priority = network.priorities
			// result has information about the conflict and how it solve it
			insertResult := super.InsertCidr(ipnet, metadata)
			fmt.Println(insertResult.String()) // see what happened
		}
	}
	// get all networks (it will return conflict free networks)
	nodes := super.AllCIDRS(false)

	for _, node := range nodes {
		fmt.Printf("CIDR: %s, name: %s\n", supernet.NodeToCidr(node), node.Metadata().Attributes["name"])
	}

	// if you want to to lookup a an IP
	ipnet, node, _ := super.LookupIP("123.123.123.16")
	if ipnet != nil {
		fmt.Printf("found: CIDR: %s, name: %s\n", ipnet.String(), node.Metadata().Attributes["name"])
	}

}

// OUTPUT:
// Action Taken: Insert New CIDR, Added CIDRs: [123.123.123.0/24], Removed CIDRs: []

// Detect Super CIDR conflict |New CIDR 123.123.122.0/23 conflicted with [123.123.123.0/24 ]
// Action Taken: Remove Existing CIDR, Added CIDRs: [], Removed CIDRs: [123.123.123.0/24]
// Action Taken: Insert New CIDR, Added CIDRs: [123.123.122.0/23], Removed CIDRs: []

// Detect Sub CIDR conflict |New CIDR 123.123.123.0/30 conflicted with [123.123.122.0/23 ]
// Action Taken: Insert New CIDR, Added CIDRs: [123.123.123.0/30], Removed CIDRs: []
// Action Taken: Split Existing CIDR, Added CIDRs: [123.123.123.4/30 123.123.123.8/29 123.123.123.16/28 123.123.123.32/27 123.123.123.64/26 123.123.123.128/25 123.123.122.0/24], Removed CIDRs: []
// Action Taken: Remove Existing CIDR, Added CIDRs: [], Removed CIDRs: [123.123.122.0/23]

// CIDR: 123.123.122.0/24, name: could be my home network
// CIDR: 123.123.123.0/30, name: it is my home network
// CIDR: 123.123.123.4/30, name: could be my home network
// CIDR: 123.123.123.8/29, name: could be my home network
// CIDR: 123.123.123.16/28, name: could be my home network
// CIDR: 123.123.123.32/27, name: could be my home network
// CIDR: 123.123.123.64/26, name: could be my home network
// CIDR: 123.123.123.128/25, name: could be my home network

// found: CIDR: 123.123.123.16/28, name: could be my home network
