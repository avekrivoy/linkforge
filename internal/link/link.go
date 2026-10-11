package link

import (
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

	link := Link{
		ID:        id,
		Code:      base62.EncodeWidth(id, 6),
		Target:    target,
		CreatedAt: time.Now().UTC(),
	}

	return link, nil
}
