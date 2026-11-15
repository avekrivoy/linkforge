package trie

type trieNode struct {
	children [256]*trieNode
	isWord   bool
}

type Trie struct {
	root *trieNode
}

func New() *Trie {
	return &Trie{
		root: &trieNode{},
	}
}

func (t *Trie) Insert(word string) bool {
	if word == "" {
		return false
	}

	node := t.root

	for i := 0; i < len(word); i++ {
		b := word[i]

		if node.children[b] == nil {
			node.children[b] = &trieNode{}
		}

		node = node.children[b]
	}

	if node.isWord {
		return false
	}

	node.isWord = true
	return true
}

func (t *Trie) Contains(word string) bool {
	if word == "" {
		return false
	}

	node := t.findNode(word)
	return node != nil && node.isWord
}

func (t *Trie) Delete(word string) bool {
	if word == "" {
		return false
	}

	return deleteWord(t.root, word, 0)
}

func deleteWord(node *trieNode, word string, depth int) bool {
	if depth == len(word) {
		if !node.isWord {
			return false
		}

		node.isWord = false
		return true
	}

	b := word[depth]
	child := node.children[b]

	if child == nil {
		return false
	}

	deleted := deleteWord(child, word, depth+1)
	if !deleted {
		return false
	}

	if !child.isWord && !hasChildren(child) {
		node.children[b] = nil
	}

	return true
}

func (t *Trie) WithPrefix(prefix string, limit int) []string {
	node := t.findNode(prefix)
	if node == nil {
		return []string{}
	}

	result := make([]string, 0)

	var walk func(*trieNode, []byte)

	walk = func(current *trieNode, path []byte) {
		if limit > 0 && len(result) >= limit {
			return
		}

		if current.isWord {
			result = append(result, string(path))
		}

		for b := 0; b < 256; b++ {
			if limit > 0 && len(result) >= limit {
				return
			}

			child := current.children[b]
			if child == nil {
				continue
			}

			nextPath := append(path, byte(b))
			walk(child, nextPath)
		}
	}

	walk(node, []byte(prefix))

	return result
}

func (t *Trie) findNode(s string) *trieNode {
	node := t.root

	for i := 0; i < len(s); i++ {
		node = node.children[s[i]]
		if node == nil {
			return nil
		}
	}

	return node
}

func hasChildren(node *trieNode) bool {
	for _, child := range node.children {
		if child != nil {
			return true
		}
	}

	return false
}
