package trie

import "fmt"

// is an alias for int used to define child positions in a trie node.
type ChildPos = int

// Constants representing possible child positions in the trie.
const (
	ZERO ChildPos = 0
	ONE  ChildPos = 1
)

// BinaryTrie is a generic type representing a node in a trie.
type BinaryTrie[T any] struct {
	parent   *BinaryTrie[T]    // Pointer to the parent node
	children [2]*BinaryTrie[T] // Array of pointers to child nodes
	metadata *T                // Generic type to store additional information
	pos      bool              // Represents the potions value at this node's position in its parent (0 or 1)
	depth    int               // The depth of this node in the trie
}

// creates a new trie node with the provided metadata and initializes it.
func NewTrieWithMetadata[T any](metadata *T) *BinaryTrie[T] {
	return &BinaryTrie[T]{
		metadata: metadata,
		depth:    0,
	}
}

// creates a new trie node with with no metadata
func NewTrie() *BinaryTrie[string] {
	s := "" // not sure but to trick the compiler
	return NewTrieWithMetadata(&s)
}

// checks if the current node is the root of the trie.
func (t *BinaryTrie[T]) IsRoot() bool {
	return t.parent == nil
}

// returns 1 or 0.
func (t *BinaryTrie[T]) Pos() int {
	if t.pos {
		return 1
	}
	return 0
}

// returns the depth of the node in the trie.
func (t *BinaryTrie[T]) Depth() int {
	return t.depth
}

// return the parent node, it will return nil if the node is root node
func (t *BinaryTrie[T]) Parent() *BinaryTrie[T] {
	return t.parent
}

// return the generic metadata
func (t *BinaryTrie[T]) Metadata() *T {
	return t.metadata
}

func (t *BinaryTrie[T]) UpdateMetadata(newNewMetadata *T) {
	t.metadata = newNewMetadata
}

// attached a child at the specified position if no child exists there yet.
// return the new attached child or the existing one
func (t *BinaryTrie[T]) AttachChild(child *BinaryTrie[T], at ChildPos) *BinaryTrie[T] {
	if t.children[at] != nil {
		return t.children[at]
	}
	return t.ReplaceChild(child, at)
}

// replacing any existing child, or simply attach the child
// if child is replace it will potentially will detach the the subtree of child
func (t *BinaryTrie[T]) ReplaceChild(child *BinaryTrie[T], at ChildPos) *BinaryTrie[T] {
	child.parent = t
	child.pos = (at == ONE)
	child.depth = t.depth + 1
	t.children[at] = child
	return child
}

// attach child as sibilating to the current node
func (t *BinaryTrie[T]) AttachSibling(sibling *BinaryTrie[T]) *BinaryTrie[T] {
	return t.parent.AttachChild(sibling, t.Pos()^1)
}

// returns the child node Zero or One
//
//	Example:
//		node.Child(trie.Zero)
func (t *BinaryTrie[T]) Child(at ChildPos) *BinaryTrie[T] {

	if t == nil {
		panic("[BUG] GetChildAt: struct must not be nil nil")
	}

	return t.children[at]
}

// it will return the sibling at the same level, or nil
func (t *BinaryTrie[T]) Sibling() *BinaryTrie[T] {
	return t.parent.Child(t.Pos() ^ 1)
}

func (t *BinaryTrie[T]) IsBranch() bool {
	return t.Sibling() != nil
}

// Detach will discount the node from the tree
// if there is no reference to the node, it will be GC'ed
func (t *BinaryTrie[T]) Detach() {
	if !t.IsRoot() {
		t.parent.children[t.Pos()] = nil
	} else {
		panic("[BUG] Detach: You can not Detach the root")
	}
}

// removes current node, and the whole branch
// and return the parent of last removed node
// this will remove any parent that had only one child until it
// reaches a parent that have 2 children (beginning of the branch)
// node(branch) -->node-->node-->node
//
//	l>node-->node-->node (Dutch)
//	[remove all the branch]
//
// will return the branch node parent
func (t *BinaryTrie[T]) DetachBranch(limit int) *BinaryTrie[T] {
	// if it has children
	if t.IsRoot() {
		panic("[Bug] DetachBranch: You can not detach Root")
	}
	nearestBranchedNode := t
	t.ForEachStepUp(func(next *BinaryTrie[T]) {
		if !next.parent.IsRoot() {
			nearestBranchedNode = next.parent
		}

	}, func(next *BinaryTrie[T]) bool {
		return !next.IsBranch() && next.depth > limit
	})

	nearestBranchedNode.Detach()

	return nearestBranchedNode.parent
}

// checks if the node is a leaf (has no children).
func (t *BinaryTrie[T]) IsLeaf() bool {
	return t.children[ZERO] == nil && t.children[ONE] == nil
}

// applies a function to each non-nil child of the node.
// will return the original node t
func (t *BinaryTrie[T]) ForEachChild(f func(t *BinaryTrie[T])) *BinaryTrie[T] {
	if t.children[ZERO] != nil {
		f(t.children[ZERO])
	}
	if t.children[ONE] != nil {
		f(t.children[ONE])
	}
	return t
}

// recursively applies a function (f) to each descendant node as long as a (while) condition holds.
// if no conation is needed you can pass nil as while parameter
// will return the original node t
func (t *BinaryTrie[T]) ForEachStepDown(f func(t *BinaryTrie[T]), while func(t *BinaryTrie[T]) bool) *BinaryTrie[T] {
	t.ForEachChild(func(child *BinaryTrie[T]) {
		if while == nil || while(t) {
			f(child)
			child.ForEachStepDown(f, while)
		}
	})
	return t
}

// applies a function to each ancestor of the node, moving from the node to the root.
// will return the original node t
func (t *BinaryTrie[T]) ForEachStepUp(f func(*BinaryTrie[T]), while func(*BinaryTrie[T]) bool) *BinaryTrie[T] {
	current := t
	for current.parent != nil && (while == nil || while(current)) {
		f(current)
		current = current.parent
	}
	return t
}

// return the path from the root node
// the path is an array of 0's and 1's
// reverse it if you need the path form the child to the root
func (t *BinaryTrie[T]) Path() []int {
	path := []int{}

	t.ForEachStepUp(func(tr *BinaryTrie[T]) {
		path = append([]int{tr.Pos()}, path...)
	}, nil)

	return path

}

// return all the leafs on the tree
func (t *BinaryTrie[T]) Leafs() []*BinaryTrie[T] {
	leafs := []*BinaryTrie[T]{}
	t.ForEachStepDown(func(t *BinaryTrie[T]) {
		if t.IsLeaf() {
			leafs = append(leafs, t)
		}
	}, nil)
	return leafs
}

// Generate an array of leafs paths which is uniq by definition
// the path is from the root to leaf
// reverse it if you need the path from leaf to root
func (root *BinaryTrie[T]) LeafsPaths() [][]int {
	paths := [][]int{}
	root.ForEachStepDown(func(t *BinaryTrie[T]) {
		if t.IsLeaf() {
			paths = append(paths, t.Path())
		}
	}, nil)
	return paths
}

func (t *BinaryTrie[T]) String(printOnLeaf func(*BinaryTrie[T]) string) {
	t.ForEachStepDown(func(node *BinaryTrie[T]) {
		if node.IsLeaf() {
			extra := ""
			if printOnLeaf != nil {
				extra = printOnLeaf(node)
			}

			fmt.Printf("%v %s\n", node.Path(), extra)
		}
	}, nil)
}
