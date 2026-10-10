// Package link holds the domain model and the contracts its consumers require.
// It may not import a transport or a storage package.
package link

// Link is a shortened URL. A zero expiry means it never expires.
type Link struct{}

// ShortenRequest carries everything needed to create a link. An empty alias
// asks for a generated code.
type ShortenRequest struct{}

// Page describes one slice of a cursor-paginated listing.
type Page struct{}
