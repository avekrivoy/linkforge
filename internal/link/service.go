package link

import (
	"time"
)

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
