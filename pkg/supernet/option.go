package supernet

import (
	"fmt"

	"github.com/khalid_nowaf/supernet/pkg/trie"
)

type Option func(*Supernet) *Supernet
type ComparatorOption func(a *Metadata, b *Metadata) bool
type LoggerOption func(*InsertionResult)

func DefaultOptions() *Supernet {
	return &Supernet{
		ipv4Cidrs:  &trie.BinaryTrie[Metadata]{},
		ipv6Cidrs:  &trie.BinaryTrie[Metadata]{},
		comparator: DefaultComparator,
		logger:     func(ir *InsertionResult) {},
	}
}

func WithComparator(comparator ComparatorOption) Option {
	return func(s *Supernet) *Supernet {
		s.comparator = comparator
		return s
	}
}

func WithCustomLogger(logger LoggerOption) Option {
	return func(s *Supernet) *Supernet {
		s.logger = logger
		return s
	}
}

func WithSimpleLogger() Option {
	return WithCustomLogger(func(ir *InsertionResult) {
		fmt.Println(ir.String())
	})
}
