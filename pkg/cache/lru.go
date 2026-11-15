package cache

import (
	"container/list"
	"errors"
)

var ErrInvalidCapacity = errors.New("cache: capacity must be positive")

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRU[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	order    *list.List
}

var _ Cache[string, int] = (*LRU[string, int])(nil)

func NewLRU[K comparable, V any](capacity int) (*LRU[K, V], error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	return &LRU[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, capacity),
		order:    list.New(),
	}, nil
}

func (c *LRU[K, V]) Get(key K) (V, bool) {
	element, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	c.order.MoveToFront(element)

	item := element.Value.(*entry[K, V])
	return item.value, true
}

func (c *LRU[K, V]) Put(key K, value V) {
	if element, ok := c.items[key]; ok {
		item := element.Value.(*entry[K, V])
		item.value = value

		c.order.MoveToFront(element)
		return
	}

	if c.order.Len() >= c.capacity {
		c.removeOldest()
	}

	item := &entry[K, V]{
		key:   key,
		value: value,
	}

	element := c.order.PushFront(item)
	c.items[key] = element
}

func (c *LRU[K, V]) Delete(key K) {
	element, ok := c.items[key]
	if !ok {
		return
	}

	c.order.Remove(element)
	delete(c.items, key)
}

func (c *LRU[K, V]) Len() int {
	return c.order.Len()
}

func (c *LRU[K, V]) Keys() []K {
	keys := make([]K, 0, c.order.Len())

	for element := c.order.Front(); element != nil; element = element.Next() {
		item := element.Value.(*entry[K, V])
		keys = append(keys, item.key)
	}

	return keys
}

func (c *LRU[K, V]) removeOldest() {
	element := c.order.Back()
	if element == nil {
		return
	}

	item := element.Value.(*entry[K, V])

	delete(c.items, item.key)
	c.order.Remove(element)
}
