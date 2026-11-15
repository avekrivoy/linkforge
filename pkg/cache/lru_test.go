package cache

import (
	"errors"
	"reflect"
	"testing"
)

func newTestCache(t *testing.T, capacity int) *LRU[string, int] {
	t.Helper()

	c, err := NewLRU[string, int](capacity)
	if err != nil {
		t.Fatalf("NewLRU() error = %v", err)
	}

	return c
}

func assertKeys(t *testing.T, c *LRU[string, int], want []string) {
	t.Helper()

	got := c.Keys()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
}

func TestNewLRU(t *testing.T) {
	t.Run("valid capacity", func(t *testing.T) {
		c, err := NewLRU[string, int](3)
		if err != nil {
			t.Fatalf("NewLRU() error = %v", err)
		}

		if c == nil {
			t.Fatal("NewLRU() returned nil cache")
		}

		if got := c.Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})

	t.Run("zero capacity", func(t *testing.T) {
		c, err := NewLRU[string, int](0)

		if c != nil {
			t.Errorf("NewLRU() cache = %v, want nil", c)
		}

		if !errors.Is(err, ErrInvalidCapacity) {
			t.Errorf("NewLRU() error = %v, want ErrInvalidCapacity", err)
		}
	})

	t.Run("negative capacity", func(t *testing.T) {
		c, err := NewLRU[string, int](-1)

		if c != nil {
			t.Errorf("NewLRU() cache = %v, want nil", c)
		}

		if !errors.Is(err, ErrInvalidCapacity) {
			t.Errorf("NewLRU() error = %v, want ErrInvalidCapacity", err)
		}
	})
}

func TestLRUGet(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		got, ok := c.Get("b")
		if !ok {
			t.Fatal("Get(b) returned false, want true")
		}

		if got != 2 {
			t.Errorf("Get(b) = %d, want 2", got)
		}

		assertKeys(t, c, []string{"b", "c", "a"})
	})

	t.Run("missing key", func(t *testing.T) {
		c := newTestCache(t, 3)
		c.Put("a", 1)

		got, ok := c.Get("missing")
		if ok {
			t.Fatal("Get(missing) returned true, want false")
		}

		if got != 0 {
			t.Errorf("Get(missing) = %d, want zero value 0", got)
		}

		assertKeys(t, c, []string{"a"})
	})
}

func TestLRUPut(t *testing.T) {
	t.Run("new key", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)

		assertKeys(t, c, []string{"b", "a"})

		if got := c.Len(); got != 2 {
			t.Errorf("Len() = %d, want 2", got)
		}
	})

	t.Run("update existing key", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		c.Put("a", 100)

		got, ok := c.Get("a")
		if !ok {
			t.Fatal("Get(a) returned false, want true")
		}

		if got != 100 {
			t.Errorf("Get(a) = %d, want 100", got)
		}

		assertKeys(t, c, []string{"a", "c", "b"})

		if got := c.Len(); got != 3 {
			t.Errorf("Len() = %d, want 3", got)
		}
	})

	t.Run("evicts least recently used", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		c.Get("a")

		c.Put("d", 4)

		assertKeys(t, c, []string{"d", "a", "c"})

		if _, ok := c.Get("b"); ok {
			t.Error("Get(b) returned true, want false: b should be evicted")
		}

		if got := c.Len(); got != 3 {
			t.Errorf("Len() = %d, want 3", got)
		}
	})

	t.Run("capacity is never exceeded", func(t *testing.T) {
		c := newTestCache(t, 2)

		for i := 0; i < 10; i++ {
			c.Put(string(rune('a'+i)), i)

			if got := c.Len(); got > 2 {
				t.Fatalf("Len() = %d, capacity = 2", got)
			}
		}
	})
}

func TestLRUDelete(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		c.Delete("b")

		assertKeys(t, c, []string{"c", "a"})

		if got := c.Len(); got != 2 {
			t.Errorf("Len() = %d, want 2", got)
		}

		if _, ok := c.Get("b"); ok {
			t.Error("Get(b) returned true after Delete(b)")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		c := newTestCache(t, 3)

		c.Put("a", 1)
		c.Put("b", 2)

		c.Delete("missing")

		assertKeys(t, c, []string{"b", "a"})

		if got := c.Len(); got != 2 {
			t.Errorf("Len() = %d, want 2", got)
		}
	})
}

func TestLRULen(t *testing.T) {
	c := newTestCache(t, 3)

	if got := c.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	c.Put("a", 1)
	c.Put("b", 2)

	if got := c.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}

	c.Delete("a")

	if got := c.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1", got)
	}
}

func TestLRUKeys(t *testing.T) {
	c := newTestCache(t, 4)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	assertKeys(t, c, []string{"c", "b", "a"})

	first := c.Keys()
	second := c.Keys()

	if !reflect.DeepEqual(first, second) {
		t.Errorf("repeated Keys() calls differ: %v and %v", first, second)
	}

	assertKeys(t, c, []string{"c", "b", "a"})
}

func TestLRUWithDifferentTypes(t *testing.T) {
	c, err := NewLRU[int, string](2)
	if err != nil {
		t.Fatal(err)
	}

	c.Put(1, "one")
	c.Put(2, "two")

	got, ok := c.Get(1)
	if !ok {
		t.Fatal("Get(1) returned false")
	}

	if got != "one" {
		t.Errorf("Get(1) = %q, want %q", got, "one")
	}

	if got := c.Keys(); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Errorf("Keys() = %v, want [1 2]", got)
	}
}
