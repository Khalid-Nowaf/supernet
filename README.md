
# Supernet (NOT STABLE, UNDER DEVELOPMENT)
`Supernet` is a conflict-free CIDR Database (network store).

## Features
- **In-Place Conflict Resolution**: Automatically resolves conflicts during CIDR insertions, based on the CIDR priorities.
- **Generic Metadata**: Supports genetic metadata for CIDRs, making it easy to add custom data for CIDRs.
- **Small Memory Footprint**: designed to minimize memory usage while maintaining fast access and modification speeds.
- **IP Lookups**: Once Supernet loads all CIDRs, it's ready to lookup IP and return the associated CIDR and its Metadata.
- **Fixable CLI**: Shipped with simple and configurable CLI that can resolve conflicts in files (JSON, CSV and TSV).



## Installation

To install `Supernet`, ensure you have Go installed on your machine (version 1.13 or later is recommended). Install the package by executing:

```sh
go get github.com/khalid-nowaf/supernet
```

## Installation

To install `supernet`, you need to have Go installed on your machine (version 1.13 or later recommended). Install the package by running:

```sh
go get github.com/khalid-nowaf/supernet
```

## Usage
Below are some examples of how you can use the supernet package to manage network CIDRs:

## CLI
```shell
go run cmd/supernet/main.go resolve

Resolve CIDR conflicts

Arguments:
  <files> ...    Input file containing CIDRs in CSV or JSON format

Flags:
  -h, --help                   Show context-sensitive help.
      --log                    Print the details about the inserted CIDR and the conflicts if any

      --cidr-key="cidr"        Key/Colum of the CIDRs in the file
      --priority-keys=,...     Keys/Columns to be used as CIDRs priorities
      --fill-empty-priority    Replace empty/null priority with zero value
      --flip-rank-priority     Make low value priority mean higher priority
      --report                 Report only conflicted CIDRs
      --output-format="csv"    Output file format
      --drop-keys=,...         Keys/Columns to be dropped
      --split-ip-versions      Split the results in to separate files based on the CIDR IP version
```
### Initializing a Supernet
```go
package main

import (
    "github.com/khalid-nowaf/supernet"
)

func main() {
    sn := supernet.NewSupernet()
    // Now you can use `sn` to insert and manage CIDRs.
}
```

### Simple Inserting CIDRs and Lookup an IP

```go
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
```

**Output**

```shell
## Insertion Results
 Action Taken: Insert New CIDR, Added CIDRs: [123.123.123.0/24], Removed CIDRs: []

 Detect Super CIDR conflict |New CIDR 123.123.122.0/23 conflicted with [123.123.123.0/24 ]
 Action Taken: Remove Existing CIDR, Added CIDRs: [], Removed CIDRs: [123.123.123.0/24]
 Action Taken: Insert New CIDR, Added CIDRs: [123.123.122.0/23], Removed CIDRs: []

 Detect Sub CIDR conflict |New CIDR 123.123.123.0/30 conflicted with [123.123.122.0/23 ]
 Action Taken: Insert New CIDR, Added CIDRs: [123.123.123.0/30], Removed CIDRs: []
 Action Taken: Split Existing CIDR, Added CIDRs: [123.123.123.4/30 123.123.123.8/29 123.123.123.16/28 123.123.123.32/27 123.123.123.64/26 123.123.123.128/25 123.123.122.0/24], Removed CIDRs: []
 Action Taken: Remove Existing CIDR, Added CIDRs: [], Removed CIDRs: [123.123.122.0/23]

## Internal CIDRs State
 CIDR: 123.123.122.0/24, name: could be my home network
 CIDR: 123.123.123.0/30, name: it is my home network
 CIDR: 123.123.123.4/30, name: could be my home network
 CIDR: 123.123.123.8/29, name: could be my home network
 CIDR: 123.123.123.16/28, name: could be my home network
 CIDR: 123.123.123.32/27, name: could be my home network
 CIDR: 123.123.123.64/26, name: could be my home network
 CIDR: 123.123.123.128/25, name: could be my home network

## IP Lookup
 found: CIDR: 123.123.123.16/28, name: could be my home network
```

### Running Tests
To run tests for the supernet package, use the Go tool:

```sh
go test github.com/khalid-nowaf/supernet
```

### Contributing
Contributions to improve supernet are welcome. Please feel free to fork the repository, make changes, and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.





