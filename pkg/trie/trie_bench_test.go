package trie

import (
	"fmt"
	"testing"
)

func BenchmarkWithPrefix(b *testing.B) {
	tr := New()

	// Створюємо приблизно 10 000 слів.
	for i := 0; i < 10_000; i++ {
		word := fmt.Sprintf("word-%05d", i)
		tr.Insert(word)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = tr.WithPrefix("word-09", 100)
	}
}
