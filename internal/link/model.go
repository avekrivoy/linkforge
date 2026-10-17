// Package link holds the domain model and the contracts its consumers require.
// It may not import a transport or a storage package.
package link

import (
	"errors"
	"time"
)

// Link is a shortened URL. A zero expiry means it never expires.
type Link struct {
	ID        uint64
	Code      string
	Target    string
	CreatedAt time.Time
}

// ShortenRequest carries everything needed to create a link. An empty alias
// asks for a generated code.
type ShortenRequest struct{}

// Page describes one slice of a cursor-paginated listing.
type Page struct{}

var (
	ErrEmptyTarget       = errors.New("link: empty target")
	ErrUnsupportedScheme = errors.New("link: unsupported scheme")
)
