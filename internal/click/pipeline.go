package click

import (
	"context"
	"runtime"
	"sync"
	"time"
)

type Metrics interface {
	ObserveClickDropped()
	SetQueueDepth(n int)
}

type BatchSink interface {
	Add(batch []Click)
}

type Pipeline struct {
	store         Store
	queue         chan Click
	workers       int
	batchSize     int
	flushInterval time.Duration
	metrics       Metrics
	sink          BatchSink

	mu      sync.RWMutex
	started bool
	closed  bool

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

type PipelineOption func(*Pipeline)

type noopPipelineMetrics struct{}

func (noopPipelineMetrics) ObserveClickDropped() {}
func (noopPipelineMetrics) SetQueueDepth(int)    {}

func WithQueueSize(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 0 {
			p.queue = make(chan Click, n)
		}
	}
}

func WithWorkers(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 0 {
			p.workers = n
		}
	}
}

func WithBatchSize(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 0 {
			p.batchSize = n
		}
	}
}

func WithFlushInterval(d time.Duration) PipelineOption {
	return func(p *Pipeline) {
		if d > 0 {
			p.flushInterval = d
		}
	}
}

func WithMetrics(m Metrics) PipelineOption {
	return func(p *Pipeline) {
		if m != nil {
			p.metrics = m
		}
	}
}

func WithSink(s BatchSink) PipelineOption {
	return func(p *Pipeline) {
		p.sink = s
	}
}

func NewPipeline(store Store, opts ...PipelineOption) *Pipeline {
	p := &Pipeline{
		store:         store,
		queue:         make(chan Click, 10000),
		workers:       runtime.GOMAXPROCS(0),
		batchSize:     500,
		flushInterval: time.Second,
		metrics:       noopPipelineMetrics{},
		done:          make(chan struct{}),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

var _ Ingester = (*Pipeline)(nil)

func (p *Pipeline) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started || p.closed {
		return
	}

	p.started = true
	p.ctx, p.cancel = context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(p.workers)

	for i := 0; i < p.workers; i++ {
		go func() {
			defer wg.Done()
			p.worker()
		}()
	}

	go func() {
		wg.Wait()
		close(p.done)
	}()
}

func (p *Pipeline) Submit(c Click) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		p.metrics.ObserveClickDropped()
		return false
	}

	select {
	case p.queue <- c:
		p.metrics.SetQueueDepth(len(p.queue))
		return true
	default:
		p.metrics.ObserveClickDropped()
		return false
	}
}

func (p *Pipeline) worker() {
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	batch := make([]Click, 0, p.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		toAppend := batch
		batch = make([]Click, 0, p.batchSize)

		if err := p.store.Append(p.ctx, toAppend); err != nil {
			for range toAppend {
				p.metrics.ObserveClickDropped()
			}
			return
		}

		if p.sink != nil {
			p.sink.Add(toAppend)
		}
	}

	for {
		select {
		case c, ok := <-p.queue:
			if !ok {
				flush()
				return
			}

			batch = append(batch, c)

			if len(batch) >= p.batchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

func (p *Pipeline) Shutdown(ctx context.Context) error {
	p.mu.Lock()

	if !p.started {
		p.closed = true
		p.mu.Unlock()
		return nil
	}

	if !p.closed {
		p.closed = true
		close(p.queue)
	}

	done := p.done
	cancel := p.cancel

	p.mu.Unlock()

	select {
	case <-done:
		cancel()
		return nil

	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}
