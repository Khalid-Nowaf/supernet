package trie

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewTrie verifies that a new Trie node is correctly initialized with default values.
func TestNewTrie(t *testing.T) {
	root := NewTrie()
	assert.NotNil(t, root, "Trie should not be nil upon creation")
	assert.Empty(t, *root.metadata, "Metadata should be initialized to nil for a new boolean Trie")
	assert.Equal(t, 0, root.Depth(), "Depth should be initialized to 0 for a new Trie")
}

// TestNewTrieWithMetadata verifies the initialization of a Trie node with specific metadata.
func TestNewTrieWithMetadata(t *testing.T) {
	root := NewTrie()
	assert.NotNil(t, root, "Trie should not be nil upon creation")
	assert.Equal(t, "", *root.metadata, "Metadata should match the initialization value")
	assert.Equal(t, 0, root.Depth(), "Depth should be initialized to 0 for a new Trie")
}

// TestAddChildAtIfNotExist verifies the behavior of adding a child node if it does not already exist.
func TestAddChildAtIfNotExist(t *testing.T) {
	root := NewTrie()
	child := NewTrie()
	addedChild := root.AttachChild(child, ONE)

	assert.Equal(t, child, addedChild, "Should return the added child")
	assert.Equal(t, root, child.parent, "Child's parent should be set correctly")
	assert.Equal(t, 1, child.Pos(), "Child's binary value should be true for position ONE")
	assert.Equal(t, 1, child.Depth(), "Child's depth should increment by 1 from the parent")
}

// TestGetChildAt verifies retrieving children from specific positions.
func TestGetChildAt(t *testing.T) {
	root := NewTrie()
	child := NewTrie()
	root.AttachChild(child, ZERO)

	assert.Equal(t, child, root.Child(ZERO), "Should retrieve the child at position ZERO")
	assert.Nil(t, root.Child(ONE), "Should return nil for an empty child position")
}

// TestForEachChild checks that ForEachChild iterates over all children correctly.
func TestForEachChild(t *testing.T) {
	root := NewTrie()
	root.AttachChild(NewTrie(), ZERO)
	root.AttachChild(NewTrie(), ONE)

	var count int
	root.ForEachChild(func(t *BinaryTrie[string]) {
		count++
	})

	assert.Equal(t, 2, count, "ForEachChild should iterate over both children")
}

// TestForEachStepDown verifies that each node in the trie can be visited and modified.
func TestForEachStepDown(t *testing.T) {
	visitedPaths := ""
	var traverseAndVerify func(tr *BinaryTrie[string])

	paths := []string{"001", "0010", "1010", "101010101010", "111111"}

	root := NewTrie()
	generateTrieAs(paths, root)

	// Mark each visited node with "visited" in its metadata.
	root.ForEachStepDown(func(tr *BinaryTrie[string]) {
		tr.metadata = strPtr("visited")
	}, nil)

	traverseAndVerify = func(tr *BinaryTrie[string]) {
		if tr == nil {
			return
		}
		tr.ForEachChild(func(c *BinaryTrie[string]) {
			visitedPaths += strconv.Itoa(c.Pos())
			assert.Contains(t, *c.metadata, "visited", "Metadata should contain 'visited'")
			traverseAndVerify(c)
		})
	}
	traverseAndVerify(root)
	assert.Equal(t, "001010101010101011111", visitedPaths, "Visited paths should match expected sequence")
}

// TestGetPath verifies that the path from the root to a specific node is correctly identified.
func TestGetPath(t *testing.T) {
	root := NewTrie()
	child := root.AttachChild(NewTrie(), ONE)
	grandchild := child.AttachChild(NewTrie(), ZERO)

	path := grandchild.Path()
	expectedPath := []int{1, 0}
	assert.Equal(t, expectedPath, path, "Path should correctly represent the bits from root to grandchild")
}

// TestGetUniquePaths verifies that unique paths in a trie are correctly identified and returned.
func TestGetUniquePaths(t *testing.T) {
	paths := []string{"001", "0010", "1010", "101010", "1111"}
	root := NewTrie()

	generateTrieAs(paths, root)

	expectedPaths := [][]int{
		{0, 0, 1, 0},
		{1, 0, 1, 0, 1, 0},
		{1, 1, 1, 1},
	}
	actualPaths := root.LeafsPaths()
	assert.ElementsMatch(t, expectedPaths, actualPaths, "Unique paths should match the expected paths")
}

func TestGetSibling(t *testing.T) {

	paths := []string{"0010", "0011", "00111"}
	root := NewTrie()
	generateTrieAs(paths, root)

	leafs := root.Leafs()
	assert.NotNil(t, leafs[0].Sibling())
	assert.Equal(t, 1, leafs[0].Sibling().Pos())
	assert.Nil(t, leafs[1].Sibling())
}

func TestAddSiblingIfNotExist(t *testing.T) {
	paths := []string{"0010", "0011", "00111"}
	root := NewTrie()
	generateTrieAs(paths, root)

	leafs := root.Leafs()
	assert.NotNil(t, leafs[0].Sibling())
	assert.Nil(t, leafs[1].Sibling())
	sibling := NewTrie()
	leafs[1].AttachSibling(sibling)
	assert.Equal(t, sibling, leafs[1].Sibling())

}

func TestAddSiblingIfExist(t *testing.T) {
	paths := []string{"0010", "0011", "00111"}
	root := NewTrie()
	generateTrieAs(paths, root)

	leafs := root.Leafs()
	assert.NotNil(t, leafs[0].Sibling())
	sibling := NewTrie()
	leafs[0].AttachSibling(sibling)
	assert.NotEqual(t, sibling, leafs[0].Sibling())
}

func TestDetach(t *testing.T) {
	paths := []string{"0010", "0011"}
	root := NewTrie()
	generateTrieAs(paths, root)
	assert.ElementsMatch(t, [][]int{
		{0, 0, 1, 0},
		{0, 0, 1, 1},
	}, root.LeafsPaths())
	leafs := root.Leafs()

	leafs[0].Detach()

	newLeafs := root.Leafs()
	assert.Equal(t, 1, len(newLeafs))
	assert.Equal(t, []int{0, 0, 1, 1}, newLeafs[0].Path())

	leafs[1].Detach()

	newLeafs = root.Leafs()
	assert.Equal(t, 1, len(newLeafs))
	assert.Equal(t, []int{0, 0, 1}, newLeafs[0].Path())

}
func TestDetachBranch(t *testing.T) {
	//(0)-> 0|0|1[0]
	//      (1)->|1|0|0|0|0[0] if we detach at last bit of the branch, the whole branch should be deleted
	paths := []string{
		"0010",
		"001100000"}
	root := NewTrie()

	generateTrieAs(paths, root)

	expectedPaths := [][]int{
		{0, 0, 1, 0},
	}
	lastLeaf := root.Leafs()
	lastLeaf[1].DetachBranch(0)
	actualPaths := root.LeafsPaths()
	assert.ElementsMatch(t, expectedPaths, actualPaths, "Unique paths should match the expected paths")

	// case where the bench is the first
	paths = []string{
		"01",
		"111010110101"}
	root = NewTrie()

	generateTrieAs(paths, root)

	expectedPaths = [][]int{
		{0, 1},
	}
	lastLeaf = root.Leafs()
	lastLeaf[1].DetachBranch(0)
	actualPaths = root.LeafsPaths()
	assert.ElementsMatch(t, expectedPaths, actualPaths, "Unique paths should match the expected paths")
}

// goos: linux
// goarch: amd64
// pkg: github.com/khalid_nowaf/supernet/pkg/trie
// cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
// BenchmarkWrites32BitPaths-12    	30400182	        40.23 ns/op	      48 B/op	       1 allocs/op
// PASS
// ok  	github.com/khalid_nowaf/supernet/pkg/trie	2.990s
func BenchmarkWrites32BitPaths(b *testing.B) {
	paths := generateRandomPaths(b.N, 19, 32)
	root := NewTrie()
	b.ResetTimer()

	for _, path := range paths {
		for _, pos := range path {
			root.AttachChild(NewTrie(), pos)
		}
	}

}

// goos: linux
// goarch: amd64
// pkg: github.com/khalid_nowaf/supernet/pkg/trie
// cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
// BenchmarkRead32BitPaths-12    	100000000	        12.47 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	github.com/khalid_nowaf/supernet/pkg/trie	10.062s// 12.47 ns/op
func BenchmarkRead32BitPaths(b *testing.B) {

	paths := generateRandomPaths(b.N, 19, 32)
	root := NewTrie()
	for _, path := range paths {
		for _, pos := range path {
			root.AttachChild(NewTrie(), pos)
		}
	}

	paths = root.LeafsPaths()
	maxPaths := len(paths)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := root
		pr := root
		randomPath := paths[rand.Intn(maxPaths)]
		for _, pos := range randomPath {
			if node == nil {
				fmt.Printf("Node is nil \npr node path: %v\n random path is:%v\n", pr.Path(), randomPath)
				panic("node is nil")
			}
			pr = node
			node = node.Child(pos)
		}
	}

}

func BenchmarkGetLeafPaths(b *testing.B) {
	paths := generateRandomPaths(b.N, 19, 32)
	root := NewTrie()
	for _, path := range paths {
		for _, pos := range path {
			root.AttachChild(NewTrie(), pos)
		}
	}

	// b.ResetTimer() this is to fast, so it redo the benchmark forever!
	root.LeafsPaths()

}

// generateTrieAs constructs a trie based on provided paths and updates it to contain metadata indicating its creation path.
func generateTrieAs(paths []string, trie *BinaryTrie[string]) {
	for _, path := range paths {
		current := trie
		for _, bit := range path {
			metadata := strPtr(*current.metadata + string(bit) + " -> ")
			if bit == '0' {
				current = current.AttachChild(NewTrieWithMetadata(metadata), ZERO)
			} else {
				current = current.AttachChild(NewTrieWithMetadata(metadata), ONE)
			}
		}
	}
}

func generateRandomPaths(totalNode int, minDepth int, maxDepth int) [][]int {
	paths := [][]int{}
	n := 0
	for i := 0; n < totalNode; i++ {
		paths = append(paths, []int{})
		for j := 0; j < rand.Intn(maxDepth-minDepth+1)+minDepth; j++ {
			paths[i] = append(paths[i], rand.Intn(2))
			n = n + 1
		}
	}
	return paths
}

func strPtr(s string) *string {
	return &s
}
