package store

import (
	"context"
	"fmt"
	"slices"

	"github.com/avekrivoy/linkforge/internal/link"
)

var _ link.Store = (*Store)(nil)

type Store struct {
	byID   map[uint64]link.Link
	byCode map[string]uint64
}

func New() *Store {
	return &Store{
		byID:   make(map[uint64]link.Link),
		byCode: make(map[string]uint64),
	}
}

func (s *Store) Create(ctx context.Context, l link.Link) (link.Link, error) {
	if err := ctx.Err(); err != nil {
		return link.Link{}, err
	}

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

func (s *Store) Delete(ctx context.Context, id uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	l, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("failed to delete the link from the store by ID: %w", link.ErrNotFound)
	}
	// if l, ok := s.byID[id]; !ok {
	// 	return fmt.Errorf("failed to delete the link from the store: %w", link.ErrNotFound)
	// }

	delete(s.byID, id)

	if _, ok := s.byCode[l.Code]; !ok {
		return fmt.Errorf("failed to delete the link from the store by Code: %w", link.ErrNotFound)
	}
	delete(s.byCode, l.Code)

	return nil
}

func (s *Store) ByCode(ctx context.Context, code string) (link.Link, error) {
	if err := ctx.Err(); err != nil {
		return link.Link{}, err
	}

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
func (s *Store) Get(id uint64) (link.Link, bool) {
	l, ok := s.byID[id]
	return l, ok
}

// All returns a slice of links sorted by ID ascending
func (s *Store) All() []link.Link {
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
func (s *Store) Count() int {
	return len(s.byID)
}

func (s *Store) ListByOwner(ctx context.Context, ownerID string, page link.Page) ([]link.Link, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	page = page.Normalize()

	afterID, err := page.AfterID()
	if err != nil {
		return nil, err
	}

	ids := sortedIDs(s.byID)
	links := make([]link.Link, 0, page.Limit)

	for _, id := range ids {
		if id <= afterID {
			continue
		}

		l := s.byID[id]

		if l.OwnerID != ownerID {
			continue
		}

		links = append(links, l)

		if len(links) == page.Limit {
			break
		}
	}

	return links, nil
}

func sortedIDs(m map[uint64]link.Link) []uint64 {
	ids := make([]uint64, 0, len(m))

	for id := range m {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	return ids
}
