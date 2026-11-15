package topk

import (
	"container/heap"
	"errors"
	"slices"
)

var ErrInvalidN = errors.New("topk: n must be positive")

type Item struct {
	Key   string
	Count int64
}

type TopK struct {
	n    int
	heap itemHeap
}

type itemHeap []Item

func (h itemHeap) Len() int {
	return len(h)
}

func (h itemHeap) Less(i, j int) bool {
	if h[i].Count != h[j].Count {
		return h[i].Count < h[j].Count
	}

	return h[i].Key > h[j].Key
}

func (h itemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *itemHeap) Push(value any) {
	*h = append(*h, value.(Item))
}

func (h *itemHeap) Pop() any {
	old := *h
	last := len(old) - 1

	item := old[last]
	*h = old[:last]

	return item
}

func New(n int) (*TopK, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}

	return &TopK{
		n:    n,
		heap: make(itemHeap, 0, n),
	}, nil
}

func (t *TopK) Add(key string, count int64) {
	item := Item{
		Key:   key,
		Count: count,
	}

	if t.heap.Len() < t.n {
		heap.Push(&t.heap, item)
		return
	}

	worst := t.heap[0]

	if !better(item, worst) {
		return
	}

	heap.Pop(&t.heap)
	heap.Push(&t.heap, item)
}

func (t *TopK) Result() []Item {
	result := make([]Item, len(t.heap))
	copy(result, t.heap)

	slices.SortFunc(result, func(a, b Item) int {
		if a.Count > b.Count {
			return -1
		}
		if a.Count < b.Count {
			return 1
		}

		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}

		return 0
	})

	return result
}

func better(a, b Item) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	}

	return a.Key < b.Key
}
