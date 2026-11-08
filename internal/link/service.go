package link

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/avekrivoy/linkforge/internal/base62"
)

var _ Shortener = (*Service)(nil)

var reservedWords = []string{
	"api",
	"ws",
	"healthz",
	"readyz",
	"metrics",
	"admin",
}

func validateAlias(a string) error {

	if a != "" {
		if len(a) < 3 || len(a) > 32 {
			return &AliasError{
				Alias: a,
				Err:   ErrInvalidAlias,
			}
		}

		for _, r := range a {
			if !(r >= 'a' && r <= 'z' ||
				r >= 'A' && r <= 'Z' ||
				r >= '0' && r <= '9' ||
				r == '_' ||
				r == '-') {
				return &AliasError{
					Alias: a,
					Err:   ErrInvalidAlias,
				}
			}
		}

		if slices.Contains(reservedWords, strings.ToLower(a)) {
			return &AliasError{
				Alias: a,
				Err:   ErrReservedAlias,
			}
		}
	}

	return nil
}

func (s *Service) Shorten(ctx context.Context, req ShortenRequest) (Link, error) {
	if len(req.OwnerID) == 0 {
		return Link{}, ErrEmptyOwner
	}

	l, err := New(s.ID, req.Target)

	if err != nil {
		return Link{}, fmt.Errorf("failed to create a link: %w", err)
	}

	if !req.ExpiresAt.IsZero() && !req.ExpiresAt.After(s.now()) {
		return Link{}, ErrInvalidExpiry
	}

	err = validateAlias(req.Alias)

	if err != nil {
		return Link{}, fmt.Errorf("failed to validate an alias: %w", err)
	}

	l.OwnerID = req.OwnerID
	l.ExpiresAt = req.ExpiresAt
	l.CreatedAt = s.now().UTC()

	if len(req.Alias) != 0 {
		l.Code = req.Alias
		link, err := s.Store.Create(ctx, l)

		if err != nil {
			if errors.Is(err, ErrAliasTaken) {
				return Link{}, &AliasError{
					Alias: req.Alias,
					Err:   err,
				}
			}

			return Link{}, fmt.Errorf("failed to create a link: %w", err)
		}

		s.ID++

		return link, nil
	} else {
		maxRetries := 3

		for range maxRetries {
			l.Code = base62.EncodeWidth(s.ID, 6)
			link, err := s.Store.Create(ctx, l)

			if err != nil {

				if errors.Is(err, ErrAliasTaken) {
					s.ID++
					l.ID = s.ID
					continue
				}

				return Link{}, fmt.Errorf("an error occured: %w", err)
			}

			s.ID++
			l.ID = s.ID

			return link, nil
		}

		return Link{}, fmt.Errorf("an error occured: %w", ErrAliasTaken)
	}
}

func (s *Service) Resolve(ctx context.Context, code string) (Link, error) {
	now := s.now()
	l, err := s.Store.ByCode(ctx, code)

	if err != nil {
		return Link{}, fmt.Errorf("failed to resolve the code %q: %w", code, err)
	}

	if !l.ExpiresAt.IsZero() && !l.ExpiresAt.After(now) {
		return Link{}, ErrExpired
	}

	return l, nil
}

func (s *Service) List(ctx context.Context, ownerID string, page Page) ([]Link, error) {
	if ownerID == "" {
		return nil, ErrEmptyOwner
	}

	return s.Store.ListByOwner(ctx, ownerID, page.Normalize())
}

func (s *Service) Delete(ctx context.Context, ownerID string, code string) error {
	if len(ownerID) == 0 {
		return ErrEmptyOwner
	}

	l, err := s.Store.ByCode(ctx, code)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("failed to delete the link: %w", err)
		} else {
			return fmt.Errorf("failed to delete the link - unexpected error occured: %w", err)
		}
	}

	if l.OwnerID != ownerID {
		return fmt.Errorf("failed to delete the link: %w", ErrForbidden)
	}

	if err = s.Store.Delete(ctx, l.ID); err != nil {
		return fmt.Errorf("failed to delete the link - unexpected error occured: %w", err)
	}
	return nil
}

// WithClock replaces a time source
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

// NewService constructs a Service object
func NewService(store Store, opts ...Option) *Service {
	svc := &Service{
		Store: store,
		ID:    1,
		now:   time.Now,
	}

	for _, o := range opts {
		o(svc)
	}

	return svc
}
