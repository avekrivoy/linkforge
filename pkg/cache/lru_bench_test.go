package cache

import "testing"

func BenchmarkLRUGetHit(b *testing.B) {
	c, err := NewLRU[string, int](100)
	if err != nil {
		b.Fatal(err)
	}

	c.Put("existing-key", 42)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, ok := c.Get("existing-key")
		if !ok {
			b.Fatal("expected cache hit")
		}
	}
}
