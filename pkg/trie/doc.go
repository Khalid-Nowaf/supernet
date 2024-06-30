// ## Overview
// Package trie implements a generic trie (prefix tree) data structure.
// The trie supports generic types and provides functions to create new nodes,
// check node properties (like being a leaf or the root), and manipulate the trie's structure
// through node addition or removal. Additional utility functions are provided to traverse
// the trie and perform actions on nodes based on various conditions.
//
// ## Example usage:
//
//		    // Add child nodes
//		    child1 := trie.NewTrieWithMetadata[string]("child1")
//	     root := trie.NewTrieWithMetadata[string]("root")
//		    root.AddChild(child1, trie.ZERO)
//
//		    child2 := trie.NewTrieWithMetadata[string]("child2")
//		    root.AddChild(child2, trie.ONE)
//
//		    // Check if a node is the root
//		    fmt.Println("Is root:", root.isRoot()) // Output: Is root: true
//
//		    // Get the depth of a node
//		    fmt.Println("Depth of child1:", child1.Depth()) // Output: Depth of child1: 1
//
//		    // Check if a node is a leaf
//		    fmt.Println("Is child1 a leaf:", child1.IsLeaf()) // Output: Is child1 a leaf: false
//
//		    // Traverse the trie and print each node's metadata
//		    root.ForEachStepDown(func(t *trie.Trie[string]) {
//		        fmt.Println(t.Metadata)
//		    }, nil)
//		}
//
// This package uses generics to allow the trie to store any type of metadata with each node.
package trie
