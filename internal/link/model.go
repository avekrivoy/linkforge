// Package link holds the domain model and the contracts its consumers require.
// It may not import a transport or a storage package.
package link

import (
	"time"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// Link is a shortened URL. A zero expiry means it never expires.
type Link struct {
	ID        uint64
	Code      string
	Target    string
	OwnerID   string
	CreatedAt time.Time
	ExpiresAt time.Time // the zero value means: never expires
}

// ShortenRequest carries everything needed to create a link. An empty alias
// asks for a generated code.
type ShortenRequest struct {
	Target    string
	Alias     string // empty means: generate the code
	OwnerID   string
	ExpiresAt time.Time
}

// Page describes one slice of a cursor-paginated listing.
type Page struct {
	Cursor string
	Limit  int
}

type Service struct {
	ID    uint64
	now   func() time.Time
	Store Store
}

type Option func(*Service)
