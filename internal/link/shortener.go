package link

import (
	"context"
	"strings"
	"time"

	"github.com/avekrivoy/linkforge/internal/base62"
)

// Shortener is the domain service every transport sits in front of. No
// transport may reach a Store directly.
type Shortener interface {
	Shorten(ctx context.Context, req ShortenRequest) (Link, error)
	Resolve(ctx context.Context, code string) (Link, error)
	List(ctx context.Context, ownerID string, page Page) ([]Link, error)
	Delete(ctx context.Context, ownerID, code string) error
}

// NoopShortener satisfies Shortener and does nothing. Implemented in M2.
type NoopShortener struct{}

var _ Shortener = NoopShortener{}

func (NoopShortener) Shorten(context.Context, ShortenRequest) (Link, error) { return Link{}, nil }

func (NoopShortener) Resolve(context.Context, string) (Link, error) { return Link{}, nil }

func (NoopShortener) List(context.Context, string, Page) ([]Link, error) { return nil, nil }

func (NoopShortener) Delete(context.Context, string, string) error { return nil }

func New(id uint64, target string) (Link, error) {
	if len(target) == 0 {
		return Link{}, ErrEmptyTarget
	}

	if !(strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")) {
		return Link{}, ErrUnsupportedScheme
	}

	link := Link{
		ID:        id,
		Code:      base62.EncodeWidth(id, 6),
		Target:    target,
		CreatedAt: time.Now().UTC(),
	}

	return link, nil
}
