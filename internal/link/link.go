package link

import (
	"fmt"
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

// AfterID decodes the cursor.
func (p Page) AfterID() (uint64, error) {
	if len(p.Cursor) == 0 {
		return 0, nil
	}

	decoded, err := base62.Decode(p.Cursor)

	if err != nil {
		return 0, fmt.Errorf("%v: %w", ErrInvalidCursor, err)
	}

	return decoded, nil
}
