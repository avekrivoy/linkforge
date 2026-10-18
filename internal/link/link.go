package link

import (
	"net/url"
	"strings"
	"time"

	"github.com/avekrivoy/linkforge/internal/base62"
)

func New(id uint64, target string) (Link, error) {
	if len(target) == 0 {
		return Link{}, ErrEmptyTarget
	}

	if !(strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")) {
		return Link{}, ErrUnsupportedScheme
	}

	_, err := url.ParseRequestURI(target)

	if err != nil {
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

func (p Page) Normalize() Page {
	if p.Limit < 1 {
		p.Limit = DefaultPageLimit
	}

	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}

	return p
}

// AfterID decodes the cursor. An empty cursor returns 0, nil.
// A cursor that base62.Decode rejects returns an error that wraps ErrInvalidCursor.
// func (p Page) AfterID() (uint64, error) {

// }
