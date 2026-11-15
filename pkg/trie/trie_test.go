package trie

import (
	"reflect"
	"testing"
)

func newTestTrie() *Trie {
	return New()
}

func assertWords(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNew(t *testing.T) {
	trie := New()

	if trie == nil {
		t.Fatal("New() returned nil")
	}

	if trie.Contains("anything") {
		t.Error("new trie should be empty")
	}

	if got := trie.WithPrefix("", 0); len(got) != 0 {
		t.Errorf("WithPrefix() = %v, want empty result", got)
	}
}

func TestInsert(t *testing.T) {
	t.Run("insert new words", func(t *testing.T) {
		trie := newTestTrie()

		if !trie.Insert("docs") {
			t.Error("Insert(docs) = false, want true")
		}

		if !trie.Insert("api") {
			t.Error("Insert(api) = false, want true")
		}

		if !trie.Contains("docs") || !trie.Contains("api") {
			t.Error("inserted words should exist")
		}
	})

	t.Run("empty word", func(t *testing.T) {
		trie := newTestTrie()

		if trie.Insert("") {
			t.Error("Insert(empty) = true, want false")
		}

		if trie.Contains("") {
			t.Error("Contains(empty) = true, want false")
		}
	})

	t.Run("duplicate word", func(t *testing.T) {
		trie := newTestTrie()

		if !trie.Insert("docs") {
			t.Fatal("first Insert(docs) = false, want true")
		}

		if trie.Insert("docs") {
			t.Error("second Insert(docs) = true, want false")
		}

		assertWords(t, trie.WithPrefix("", 0), []string{"docs"})
	})

	t.Run("word and its prefix", func(t *testing.T) {
		trie := newTestTrie()

		if !trie.Insert("doc") {
			t.Fatal("Insert(doc) = false, want true")
		}

		if !trie.Insert("docs") {
			t.Fatal("Insert(docs) = false, want true")
		}

		if !trie.Contains("doc") || !trie.Contains("docs") {
			t.Error("both prefix and longer word should exist")
		}
	})
}

func TestContains(t *testing.T) {
	trie := newTestTrie()

	trie.Insert("doc")
	trie.Insert("docs")

	tests := []struct {
		name string
		word string
		want bool
	}{
		{name: "existing word", word: "doc", want: true},
		{name: "longer word", word: "docs", want: true},
		{name: "only a prefix", word: "do", want: false},
		{name: "missing word", word: "dog", want: false},
		{name: "empty word", word: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trie.Contains(tt.word); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Run("delete existing word", func(t *testing.T) {
		trie := newTestTrie()
		trie.Insert("docs")

		if !trie.Delete("docs") {
			t.Fatal("Delete(docs) = false, want true")
		}

		if trie.Contains("docs") {
			t.Error("docs should not exist after deletion")
		}

		assertWords(t, trie.WithPrefix("", 0), []string{})
	})

	t.Run("delete missing word", func(t *testing.T) {
		trie := newTestTrie()
		trie.Insert("docs")

		if trie.Delete("missing") {
			t.Error("Delete(missing) = true, want false")
		}

		if !trie.Contains("docs") {
			t.Error("deleting missing word should not affect docs")
		}
	})

	t.Run("delete empty word", func(t *testing.T) {
		trie := newTestTrie()

		if trie.Delete("") {
			t.Error("Delete(empty) = true, want false")
		}
	})

	t.Run("delete prefix keeps longer word", func(t *testing.T) {
		trie := newTestTrie()
		trie.Insert("doc")
		trie.Insert("docs")

		if !trie.Delete("doc") {
			t.Fatal("Delete(doc) = false, want true")
		}

		if trie.Contains("doc") {
			t.Error("doc should not exist after deletion")
		}

		if !trie.Contains("docs") {
			t.Error("docs should remain after deleting doc")
		}

		assertWords(t, trie.WithPrefix("do", 0), []string{"docs"})
	})

	t.Run("delete longer word keeps prefix", func(t *testing.T) {
		trie := newTestTrie()
		trie.Insert("doc")
		trie.Insert("docs")

		if !trie.Delete("docs") {
			t.Fatal("Delete(docs) = false, want true")
		}

		if !trie.Contains("doc") {
			t.Error("doc should remain after deleting docs")
		}

		if trie.Contains("docs") {
			t.Error("docs should not exist after deletion")
		}
	})

	t.Run("delete one branch keeps another", func(t *testing.T) {
		trie := newTestTrie()
		trie.Insert("doc")
		trie.Insert("dog")
		trie.Insert("day")

		if !trie.Delete("dog") {
			t.Fatal("Delete(dog) = false, want true")
		}

		assertWords(t, trie.WithPrefix("", 0), []string{"day", "doc"})
	})
}

func TestWithPrefix(t *testing.T) {
	trie := newTestTrie()

	words := []string{
		"docs",
		"doc",
		"dog",
		"do-it",
		"day",
		"api",
		"dock",
	}

	for _, word := range words {
		trie.Insert(word)
	}

	t.Run("example without limit", func(t *testing.T) {
		got := trie.WithPrefix("do", 0)
		want := []string{"do-it", "doc", "dock", "docs", "dog"}

		assertWords(t, got, want)
	})

	t.Run("example with limit", func(t *testing.T) {
		got := trie.WithPrefix("do", 2)
		want := []string{"do-it", "doc"}

		assertWords(t, got, want)
	})

	t.Run("prefix itself is included", func(t *testing.T) {
		got := trie.WithPrefix("doc", 0)
		want := []string{"doc", "dock", "docs"}

		assertWords(t, got, want)
	})

	t.Run("empty prefix returns all words", func(t *testing.T) {
		got := trie.WithPrefix("", 0)
		want := []string{
			"api",
			"day",
			"do-it",
			"doc",
			"dock",
			"docs",
			"dog",
		}

		assertWords(t, got, want)
	})

	t.Run("negative limit means no limit", func(t *testing.T) {
		got := trie.WithPrefix("do", -1)
		want := []string{"do-it", "doc", "dock", "docs", "dog"}

		assertWords(t, got, want)
	})

	t.Run("unknown prefix", func(t *testing.T) {
		got := trie.WithPrefix("xyz", 0)

		assertWords(t, got, []string{})
	})

	t.Run("limit greater than result count", func(t *testing.T) {
		got := trie.WithPrefix("api", 10)
		want := []string{"api"}

		assertWords(t, got, want)
	})

	t.Run("zero limit means unlimited", func(t *testing.T) {
		got := trie.WithPrefix("do", 0)
		want := []string{"do-it", "doc", "dock", "docs", "dog"}

		assertWords(t, got, want)
	})
}

func TestWithPrefixByteOrder(t *testing.T) {
	trie := newTestTrie()

	trie.Insert("a")
	trie.Insert("A")
	trie.Insert("0")
	trie.Insert("b")

	got := trie.WithPrefix("", 0)
	want := []string{"0", "A", "a", "b"}

	assertWords(t, got, want)
}

func TestTrieAfterDeleteAndInsert(t *testing.T) {
	trie := newTestTrie()

	trie.Insert("docs")
	trie.Insert("docker")

	if !trie.Delete("docs") {
		t.Fatal("Delete(docs) = false, want true")
	}

	if !trie.Insert("docs") {
		t.Fatal("Insert(docs) after deletion = false, want true")
	}

	assertWords(t, trie.WithPrefix("do", 0), []string{
		"docker",
		"docs",
	})
}
