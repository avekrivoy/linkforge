package store

import (
	"errors"
	"slices"

	"github.com/avekrivoy/linkforge/internal/link"
)

type Store struct {
	byID map[uint64]link.Link
}

var (
	ErrDuplicateID = errors.New("store: id is already taken")
)

// New creates a new Store
func New() Store {
	return Store{
		byID: make(map[uint64]link.Link),
	}
}

// Add adds a link to the store
func Add(s Store, l link.Link) error {
	if _, ok := s.byID[l.ID]; ok {
		return ErrDuplicateID
	}
	s.byID[l.ID] = l

	return nil
}

// Get returns a link from the store
func Get(s Store, id uint64) (link.Link, bool) {
	l, ok := s.byID[id]
	return l, ok
}

// All returns a slice of links sorted by ID ascending
func All(s Store) []link.Link {
	var ids = make([]uint64, 0, len(s.byID))
	var links = make([]link.Link, 0, len(s.byID))

	for id := range s.byID {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	for _, id := range ids {
		links = append(links, s.byID[id])
	}

	return links
}

// Count returns a number of elements in the store
func Count(s Store) int {
	return len(s.byID)
}
