package store

import (
	"context"
	"slices"

	"github.com/avekrivoy/linkforge/internal/link"
)

type Store struct {
	byID   map[uint64]link.Link
	byCode map[string]uint64
}

// New creates a new Store
func New() *Store {
	return &Store{
		byID: make(map[uint64]link.Link),
	}
}

func (s *Store) Create(ctx context.Context, l link.Link) (link.Link, error) {

	if _, ok := s.byID[l.ID]; ok {
		return link.Link{}, ErrDuplicateID
	}

	if _, err := s.ByCode(ctx, l.Code); err == nil {
		return link.Link{}, link.ErrAliasTaken
	}

	s.byID[l.ID] = l
	s.byCode[l.Code] = l.ID

	return l, nil
}

func (s *Store) ByCode(ctx context.Context, code string) (link.Link, error) {
	id, ok := s.byCode[code]

	if !ok {
		return link.Link{}, link.ErrNotFound
	}

	l, ok := s.byID[id]

	if !ok {
		return link.Link{}, link.ErrNotFound
	}

	return l, nil
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
func Count(s *Store) int {
	return len(s.byID)
}
