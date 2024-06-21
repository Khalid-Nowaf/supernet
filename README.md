
# Supernet (NOT STABLE, UNDER DEVELOPMENT)

`supernet` is CIDR Store that can handle conflict resolution. It utilizes trie data structures to efficiently handle IPv4 and IPv6 addresses, ensuring fast conflict detection and resolution upon CIDR insertions. Designed to optimize memory usage, `supernet` can be used for different network management applications.

## Features

- **Dynamic Conflict Resolution**: Automatically resolves conflicts during CIDR insertions, based on the CIDR priorities.
- **Generic Metadata**: Supports genetic metadata for CIDRs, making it adaptable for various applications.
- **Optimized Memory Footprint**: designed to minimize memory usage while maintaining fast access and modification speeds.
- **Fast Lookups and Conflict Detection**: Uses trie data structures to ensure rapid conflict resolution and CIDR lookups, crucial for high-performance network environments.

## Installation

To install `supernet`, ensure you have Go installed on your machine (version 1.13 or later is recommended). Install the package by executing:

```sh
go get github.com/khalid_nowaf/supernet


## Installation

To install `supernet`, you need to have Go installed on your machine (version 1.13 or later recommended). Install the package by running:

```sh
go get github.com/khalid_nowaf/supernet
```

## Usage
Below are some examples of how you can use the supernet package to manage network CIDRs:

### Initializing a Supernet
```go
package main

import (
    "github.com/khalid_nowaf/supernet"
)

func main() {
    sn := supernet.NewSupernet()
    // Now you can use `sn` to insert and manage CIDRs.
}
```

### Inserting a CIDR
```go
import (
    "net"
    "github.com/khalid_nowaf/supernet"
)

func main() {
    sn := supernet.NewSupernet()
    _, ipnet, _ := net.ParseCIDR("192.168.100.14/24")
    metadata := supernet.NewDefaultMetadata()
    sn.InsertCidr(ipnet, metadata)
}

```
### Searching for a CIDR
```go
import (
    "fmt"
    "github.com/khalid_nowaf/supernet"
)

func main() {
    sn := supernet.NewSupernet()
    result, err := sn.LookupIP("192.168.100.14")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    if result != nil {
        fmt.Printf("Found CIDR: %s\n", result)
    } else {
        fmt.Println("No matching CIDR found.")
    }
}
```

### Running Tests
To run tests for the supernet package, use the Go tool:

```sh
go test github.com/khalid_nowaf/supernet
```

### Contributing
Contributions to improve supernet are welcome. Please feel free to fork the repository, make changes, and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.





