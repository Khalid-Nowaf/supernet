package trie

// is an alias for int used to define child positions in a trie node.
type ChildPos = int

// Constants representing possible child positions in the trie.
const ZERO ChildPos = 0
const ONE ChildPos = 1

// Trie is a generic type representing a node in a trie.
type Trie[T any] struct {
	Parent   *Trie[T]    // Pointer to the parent node
	Children [2]*Trie[T] // Array of pointers to child nodes
	Metadata *T          // Generic type to store additional information
	pos      bool        // Represents the potions value at this node's position in its parent (0 or 1)
	depth    int         // The depth of this node in the trie
}

// NewTrieWithMetadata creates a new trie node with the provided metadata and initializes it.
func NewTrieWithMetadata[T any](metadata *T) *Trie[T] {
	return &Trie[T]{
		Metadata: metadata,
		depth:    0,
	}
}

// creates a new trie node with with no metadata
func NewTrie() *Trie[string] {
	s := "" // not sure but to trick the compiler
	return NewTrieWithMetadata(&s)
}

// isRoot checks if the current node is the root of the trie.
func (t *Trie[T]) isRoot() bool {
	return t.Parent == nil
}

// returns 1 or 0.
func (t *Trie[T]) GetPos() int {
	if t.pos {
		return 1
	}
	return 0
}

// GetDepth returns the depth of the node in the trie.
func (t *Trie[T]) GetDepth() int {
	return t.depth
}

// adds a child at the specified position if no child exists there yet.
// return the new added child or the existing one
func (t *Trie[T]) AddChildAtIfNotExist(child *Trie[T], at ChildPos) *Trie[T] {
	if t.Children[at] != nil {
		return t.Children[at]
	}
	return t.AddChildOrReplaceAt(child, at)
}

// adds and return a child at the specified position, replacing any existing child.
// this potently will detach the the subtree of the child has children
func (t *Trie[T]) AddChildOrReplaceAt(child *Trie[T], at ChildPos) *Trie[T] {
	child.Parent = t
	child.pos = (at == ONE)
	child.depth = t.depth + 1
	t.Children[at] = child
	return child
}

// returns the child node Zero or One
//
//	node.GetChildAt(trie.Zero)
func (t *Trie[T]) GetChildAt(at ChildPos) *Trie[T] {
	// TODO: user can get nil
	if t == nil {
		panic("WTF")
	}
	return t.Children[at]
}

// removes both children of the node, effectively making it a leaf node.
func (t *Trie[T]) MakeItALeaf() *Trie[T] {
	t.Children[0] = nil
	t.Children[1] = nil
	return t
}

// checks if the node is a leaf (has no children).
func (t *Trie[T]) IsLeaf() bool {
	return t.Children[0] == nil && t.Children[1] == nil
}

// applies a function to each non-nil child of the node.
// will return the original node t
func (t *Trie[T]) ForEachChild(f func(t *Trie[T])) *Trie[T] {
	if t.Children[0] != nil {
		f(t.Children[0])
	}
	if t.Children[1] != nil {
		f(t.Children[1])
	}
	return t
}

// recursively applies a function (f) to each descendant node as long as a (while) condition holds.
// if no conation is needed you can pass nil as while parameter
// will return the original node t
func (t *Trie[T]) ForEachStepDown(f func(t *Trie[T]), while func(t *Trie[T]) bool) *Trie[T] {
	t.forEachStepDown(f, while)
	return t
}

// is a helper for ForEachStepDown to implement recursive traversal.
// will return the original node t
func (t *Trie[T]) forEachStepDown(f func(t *Trie[T]), while func(t *Trie[T]) bool) *Trie[T] {
	t.ForEachChild(func(child *Trie[T]) {
		if while == nil || while(t) {
			f(child)
			child.forEachStepDown(f, while)
		}
	})
	return t
}

// applies a function to each ancestor of the node, moving from the node to the root.
// will return the original node t
func (t *Trie[T]) ForEachStepUp(f func(t *Trie[T]), while func(t *Trie[T]) bool) *Trie[T] {
	current := t
	for current.Parent != nil && (while == nil || while(current)) {
		f(current)
		current = current.Parent
	}
	return t
}

// return the path from the root node
// the path is an array of 0's and 1's
// reverse it if you need the path form the child to the root
func (t *Trie[T]) GetPath() []int {
	path := []int{}

	t.ForEachStepUp(func(tr *Trie[T]) {
		path = append([]int{tr.GetPos()}, path...)
	}, nil)

	return path

}

// TODO: Doc This
func (t *Trie[T]) GetLeafs() []*Trie[T] {
	leafs := []*Trie[T]{}
	t.ForEachStepDown(func(t *Trie[T]) {
		if t.IsLeaf() {
			leafs = append(leafs, t)
		}
	}, nil)
	return leafs
}

// Generate an array of leafs paths which is uniq by definition
// the path is from the root to leaf
// reverse it if you need the path from leaf to root
func (root *Trie[T]) GetLeafsPaths() [][]int {
	paths := [][]int{}
	root.ForEachStepDown(func(t *Trie[T]) {
		if t.IsLeaf() {
			paths = append(paths, t.GetPath())
		}
	}, nil)
	return paths
}
