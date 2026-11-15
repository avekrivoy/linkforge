package click

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/avekrivoy/linkforge/pkg/topk"
)

var (
	ErrInvalidGranularity = errors.New("click: invalid granularity")
	ErrInvalidN           = errors.New("click: n must be positive")
	ErrClosed             = errors.New("click: counters are closed")
)

type Counters struct {
	requests chan counterRequest
	done     chan struct{}

	mu     sync.RWMutex
	closed bool
}

type counterRequest struct {
	kind        counterRequestKind
	batch       []Click
	code        string
	window      Window
	granularity Granularity
	n           int
	reply       chan counterResponse
}

type counterRequestKind int

const (
	counterAdd counterRequestKind = iota
	counterSeries
	counterTop
)

type counterResponse struct {
	buckets []Bucket
	totals  []Total
	err     error
}

var _ Aggregator = (*Counters)(nil)
var _ BatchSink = (*Counters)(nil)

func NewCounters() *Counters {
	c := &Counters{
		requests: make(chan counterRequest),
		done:     make(chan struct{}),
	}

	go c.run()

	return c
}

func (c *Counters) run() {
	// Дані належать тільки цій goroutine.
	// Структура: code -> UTC hour -> clicks.
	counts := make(map[string]map[time.Time]int64)

	for {
		select {
		case <-c.done:
			return

		case req := <-c.requests:
			switch req.kind {
			case counterAdd:
				for _, click := range req.batch {
					if click.Code == "" {
						continue
					}

					hour := click.At.UTC().Truncate(time.Hour)

					if counts[click.Code] == nil {
						counts[click.Code] = make(map[time.Time]int64)
					}

					counts[click.Code][hour]++
				}

				req.reply <- counterResponse{}

			case counterSeries:
				buckets, err := buildSeries(
					counts,
					req.code,
					req.window,
					req.granularity,
				)

				req.reply <- counterResponse{
					buckets: buckets,
					err:     err,
				}

			case counterTop:
				totals, err := buildTop(counts, req.window, req.n)

				req.reply <- counterResponse{
					totals: totals,
					err:    err,
				}
			}
		}
	}
}

// Add передає batch owner goroutine.
// Метод повертається після того, як owner обробив batch.
func (c *Counters) Add(batch []Click) {
	if len(batch) == 0 {
		return
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	reply := make(chan counterResponse, 1)

	req := counterRequest{
		kind:  counterAdd,
		batch: append([]Click(nil), batch...),
		reply: reply,
	}

	select {
	case c.requests <- req:
	case <-c.done:
		return
	}

	select {
	case <-reply:
	case <-c.done:
	}
}

func (c *Counters) Series(
	ctx context.Context,
	code string,
	w Window,
	g Granularity,
) ([]Bucket, error) {
	// Порядок перевірок заданий умовою.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if g != GranularityHour && g != GranularityDay {
		return nil, ErrInvalidGranularity
	}

	if c.isClosed() {
		return nil, ErrClosed
	}

	reply := make(chan counterResponse, 1)

	req := counterRequest{
		kind:        counterSeries,
		code:        code,
		window:      w,
		granularity: g,
		reply:       reply,
	}

	select {
	case c.requests <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClosed
	}

	select {
	case result := <-reply:
		return result.buckets, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClosed
	}
}

func (c *Counters) Top(
	ctx context.Context,
	w Window,
	n int,
) ([]Total, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if n < 1 {
		return nil, ErrInvalidN
	}

	if c.isClosed() {
		return nil, ErrClosed
	}

	reply := make(chan counterResponse, 1)

	req := counterRequest{
		kind:   counterTop,
		window: w,
		n:      n,
		reply:  reply,
	}

	select {
	case c.requests <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClosed
	}

	select {
	case result := <-reply:
		return result.totals, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClosed
	}
}

func (c *Counters) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

func (c *Counters) Close() {
	c.mu.Lock()

	if !c.closed {
		c.closed = true
		close(c.done)
	}

	c.mu.Unlock()

	// Чекаємо завершення owner goroutine.
	<-c.done
}

func buildSeries(
	counts map[string]map[time.Time]int64,
	code string,
	w Window,
	g Granularity,
) ([]Bucket, error) {
	hours, ok := counts[code]
	if !ok {
		return []Bucket{}, nil
	}

	result := make([]Bucket, 0)

	if g == GranularityHour {
		for start, clicks := range hours {
			if start.Before(w.From) || !start.Before(w.To) {
				continue
			}

			result = append(result, Bucket{
				Start:  start,
				Clicks: clicks,
			})
		}
	} else {
		daily := make(map[time.Time]int64)

		for hour, clicks := range hours {
			if hour.Before(w.From) || !hour.Before(w.To) {
				continue
			}

			y, m, d := hour.UTC().Date()
			day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

			daily[day] += clicks
		}

		for start, clicks := range daily {
			result = append(result, Bucket{
				Start:  start,
				Clicks: clicks,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Start.Before(result[j].Start)
	})

	return result, nil
}

func buildTop(
	counts map[string]map[time.Time]int64,
	w Window,
	n int,
) ([]Total, error) {
	// Використовуємо topk із домашки 4.
	ranking, err := topk.New(n)
	if err != nil {
		return nil, ErrInvalidN
	}

	for code, hours := range counts {
		var total int64

		for hour, clicks := range hours {
			if hour.Before(w.From) || !hour.Before(w.To) {
				continue
			}

			total += clicks
		}

		if total > 0 {
			ranking.Add(code, total)
		}
	}

	items := ranking.Result()

	result := make([]Total, 0, len(items))
	for _, item := range items {
		result = append(result, Total{
			Code:   item.Key,
			Clicks: item.Count,
		})
	}

	return result, nil
}
