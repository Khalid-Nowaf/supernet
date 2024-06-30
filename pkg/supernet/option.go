package supernet

import "github.com/khalid_nowaf/supernet/pkg/trie"

type Option func(*Supernet) *Supernet
type ComparatorOption func(a *Metadata, b *Metadata) bool

func DefaultOptions() *Supernet {
	return &Supernet{
		ipv4Cidrs:  &trie.BinaryTrie[Metadata]{},
		ipv6Cidrs:  &trie.BinaryTrie[Metadata]{},
		comparator: DefaultComparator,
	}
}

func WithComparator(comparator ComparatorOption) Option {
	return func(s *Supernet) *Supernet {
		s.comparator = comparator
		return s
	}
}
