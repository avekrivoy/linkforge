package link

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyTarget       = errors.New("link: empty target")
	ErrUnsupportedScheme = errors.New("link: unsupported scheme")
	ErrNotFound          = errors.New("link: not found")
	ErrAliasTaken        = errors.New("link: alias already taken")
	ErrInvalidAlias      = errors.New("link: invalid alias")
	ErrReservedAlias     = errors.New("link: reserved alias")
	ErrForbidden         = errors.New("link: forbidden")
	ErrExpired           = errors.New("link: expired")
	ErrInvalidExpiry     = errors.New("link: invalid expiry")
	ErrEmptyOwner        = errors.New("link: owner is required")
	ErrInvalidCursor     = errors.New("link: invalid cursor")
)

// AliasError holds the alias of the original error
type AliasError struct {
	Alias string
	Err   error
}

func (e *AliasError) Error() string {
	return fmt.Sprintf("alias %q: %v", e.Alias, e.Err)
}

func (e *AliasError) Unwrap() error {
	return e.Err
}
