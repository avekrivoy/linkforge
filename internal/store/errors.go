package store

import "errors"

var (
	ErrDuplicateID = errors.New("store: id is already taken")
)
