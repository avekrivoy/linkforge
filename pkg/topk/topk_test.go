package topk

import (
	"errors"
	"reflect"
	"testing"
)

func newTestTopK(t *testing.T, n int) *TopK {
	t.Helper()

	top, err := New(n)
	if err != nil {
		t.Fatalf("New(%d) error = %v", n, err)
	}

	return top
}

func assertResult(t *testing.T, top *TopK, want []Item) {
	t.Helper()

	got := top.Result()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Result() = %+v, want %+v", got, want)
	}
}

func TestNew(t *testing.T) {
	t.Run("valid N", func(t *testing.T) {
		top, err := New(3)

		if err != nil {
			t.Fatalf("New(3) error = %v", err)
		}

		if top == nil {
			t.Fatal("New(3) returned nil")
		}

		if got := top.Result(); len(got) != 0 {
			t.Errorf("Result() = %v, want empty result", got)
		}
	})

	t.Run("zero N", func(t *testing.T) {
		top, err := New(0)

		if top != nil {
			t.Errorf("New(0) = %v, want nil", top)
		}

		if !errors.Is(err, ErrInvalidN) {
			t.Errorf("New(0) error = %v, want ErrInvalidN", err)
		}
	})

	t.Run("negative N", func(t *testing.T) {
		top, err := New(-1)

		if top != nil {
			t.Errorf("New(-1) = %v, want nil", top)
		}

		if !errors.Is(err, ErrInvalidN) {
			t.Errorf("New(-1) error = %v, want ErrInvalidN", err)
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("fewer items than N", func(t *testing.T) {
		top := newTestTopK(t, 3)

		top.Add("apple", 5)
		top.Add("banana", 10)

		want := []Item{
			{Key: "banana", Count: 10},
			{Key: "apple", Count: 5},
		}

		assertResult(t, top, want)
	})

	t.Run("keeps only N best items", func(t *testing.T) {
		top := newTestTopK(t, 3)

		top.Add("k0", 5)
		top.Add("k1", 1)
		top.Add("k2", 9)
		top.Add("k3", 3)
		top.Add("k4", 9)
		top.Add("k5", 7)
		top.Add("k6", 2)

		want := []Item{
			{Key: "k2", Count: 9},
			{Key: "k4", Count: 9},
			{Key: "k5", Count: 7},
		}

		assertResult(t, top, want)
	})

	t.Run("better item replaces worst", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("apple", 5)
		top.Add("banana", 10)

		top.Add("cherry", 7)

		want := []Item{
			{Key: "banana", Count: 10},
			{Key: "cherry", Count: 7},
		}

		assertResult(t, top, want)
	})

	t.Run("worse item is ignored", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("apple", 5)
		top.Add("banana", 10)

		top.Add("cherry", 3)

		want := []Item{
			{Key: "banana", Count: 10},
			{Key: "apple", Count: 5},
		}

		assertResult(t, top, want)
	})

	t.Run("equal count smaller key wins", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("b", 10)
		top.Add("c", 10)

		top.Add("a", 10)

		want := []Item{
			{Key: "a", Count: 10},
			{Key: "b", Count: 10},
		}

		assertResult(t, top, want)
	})

	t.Run("equal count larger key loses", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("a", 10)
		top.Add("b", 10)

		top.Add("c", 10)

		want := []Item{
			{Key: "a", Count: 10},
			{Key: "b", Count: 10},
		}

		assertResult(t, top, want)
	})

	t.Run("equal items are not merged", func(t *testing.T) {
		top := newTestTopK(t, 3)

		top.Add("apple", 5)
		top.Add("apple", 5)
		top.Add("apple", 5)

		want := []Item{
			{Key: "apple", Count: 5},
			{Key: "apple", Count: 5},
			{Key: "apple", Count: 5},
		}

		assertResult(t, top, want)
	})

	t.Run("negative counts", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("a", -10)
		top.Add("b", -5)
		top.Add("c", -20)

		want := []Item{
			{Key: "b", Count: -5},
			{Key: "a", Count: -10},
		}

		assertResult(t, top, want)
	})

	t.Run("N equals one", func(t *testing.T) {
		top := newTestTopK(t, 1)

		top.Add("a", 5)
		top.Add("b", 10)
		top.Add("c", 7)

		want := []Item{
			{Key: "b", Count: 10},
		}

		assertResult(t, top, want)
	})
}

func TestResult(t *testing.T) {
	t.Run("best first", func(t *testing.T) {
		top := newTestTopK(t, 4)

		top.Add("d", 1)
		top.Add("c", 5)
		top.Add("b", 5)
		top.Add("a", 10)

		want := []Item{
			{Key: "a", Count: 10},
			{Key: "b", Count: 5},
			{Key: "c", Count: 5},
			{Key: "d", Count: 1},
		}

		assertResult(t, top, want)
	})

	t.Run("does not modify heap", func(t *testing.T) {
		top := newTestTopK(t, 3)

		top.Add("a", 5)
		top.Add("b", 10)
		top.Add("c", 7)

		first := top.Result()
		second := top.Result()

		if !reflect.DeepEqual(first, second) {
			t.Errorf("repeated Result() calls differ: %v and %v", first, second)
		}

		want := []Item{
			{Key: "b", Count: 10},
			{Key: "c", Count: 7},
			{Key: "a", Count: 5},
		}

		assertResult(t, top, want)

		if top.heap.Len() != 3 {
			t.Errorf("heap.Len() = %d, want 3", top.heap.Len())
		}
	})

	t.Run("modifying result does not modify heap", func(t *testing.T) {
		top := newTestTopK(t, 2)

		top.Add("a", 5)
		top.Add("b", 10)

		result := top.Result()
		result[0].Key = "changed"
		result[0].Count = 999

		want := []Item{
			{Key: "b", Count: 10},
			{Key: "a", Count: 5},
		}

		assertResult(t, top, want)
	})

	t.Run("empty result", func(t *testing.T) {
		top := newTestTopK(t, 3)

		got := top.Result()
		if len(got) != 0 {
			t.Errorf("Result() = %v, want empty result", got)
		}
	})
}

func TestTopKHeapBounded(t *testing.T) {
	top := newTestTopK(t, 5)

	for i := 0; i < 1000; i++ {
		top.Add("key", int64(i))

		if got := top.heap.Len(); got > 5 {
			t.Fatalf("heap.Len() = %d, want at most 5", got)
		}
	}

	if got := top.heap.Len(); got != 5 {
		t.Errorf("heap.Len() = %d, want 5", got)
	}

	result := top.Result()

	if result[0].Count != 999 {
		t.Errorf("best Count = %d, want 999", result[0].Count)
	}

	if result[4].Count != 995 {
		t.Errorf("worst kept Count = %d, want 995", result[4].Count)
	}
}
