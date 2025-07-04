package goinmemcache

import (
	"sync"
	"time"
)

type Config struct {
	Size     *int64
	MaxItems *int64
}

type Cache[K comparable, V any] interface {
	Set(key K, value V)
	SetWithTTL(key K, value V, ttl *time.Duration)
	Get(key K) (V, bool)
	Delete(key K)
	Len() int
	Clear()
	CleanupExpired() int
}

type cache[K comparable, V any] struct {
	mu       sync.RWMutex
	size     *int64
	maxItems *int64

	items []cacheItem[K, V]
	index map[K]int
}

type cacheItem[K comparable, V any] struct {
	Key       K
	Value     V
	TTL       *time.Duration
	CreatedAt time.Time
}

func New[K comparable, V any](config *Config) Cache[K, V] {
	if config == nil {
		config = &Config{}
	}

	return &cache[K, V]{
		size:     config.Size,
		maxItems: config.MaxItems,
		items:    []cacheItem[K, V]{},
		index:    make(map[K]int),
	}
}

func (c *cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setItem(key, value, nil)
}

func (c *cache[K, V]) SetWithTTL(key K, value V, ttl *time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setItem(key, value, ttl)
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if index, exists := c.index[key]; exists {
		item := c.items[index]
		if c.isItemValid(item) {
			return item.Value, true // Item found and valid
		}
	}

	var zeroValue V
	return zeroValue, false // Item not found
}

func (c *cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index, exists := c.index[key]; exists {
		// Remove from index first
		delete(c.index, key)

		// Remove from items slice
		c.items = append(c.items[:index], c.items[index+1:]...)

		// Update all indices that come after the deleted item
		for k, idx := range c.index {
			if idx > index {
				c.index[k] = idx - 1
			}
		}
	}
}

// updateOrAddItem updates an existing item or adds a new one
func (c *cache[K, V]) updateOrAddItem(key K, item cacheItem[K, V]) {
	if index, exists := c.index[key]; exists {
		c.items[index] = item // Update existing item
	} else {
		c.index[key] = len(c.items) // Add new item
		c.items = append(c.items, item)
	}
}

// isItemValid checks if a cache item is valid (not expired)
func (c *cache[K, V]) isItemValid(item cacheItem[K, V]) bool {
	if item.TTL == nil {
		return true // No TTL means never expires
	}
	return time.Since(item.CreatedAt) < *item.TTL
}

// isCacheFull checks if the cache has reached its maximum capacity
func (c *cache[K, V]) isCacheFull() bool {
	return c.maxItems != nil && int64(len(c.items)) >= *c.maxItems
}

// setItem is a helper method that consolidates the logic for setting cache items
func (c *cache[K, V]) setItem(key K, value V, ttl *time.Duration) {
	// If updating existing item, no need to check capacity
	if _, exists := c.index[key]; !exists {
		// Adding new item - check if we need to make space
		if c.isCacheFull() {
			c.removeOldestItem()
		}
	}

	item := cacheItem[K, V]{
		Key:       key,
		Value:     value,
		TTL:       ttl,
		CreatedAt: time.Now(),
	}

	c.updateOrAddItem(key, item)
}

// removeOldestItem removes the oldest (first) item from the cache
func (c *cache[K, V]) removeOldestItem() {
	if len(c.items) == 0 {
		return
	}

	// Get the key of the oldest item (first in the slice)
	oldestKey := c.items[0].Key

	// Remove from index
	delete(c.index, oldestKey)

	// Remove from items slice
	c.items = c.items[1:]

	// Update all indices in the index map since we removed the first element
	for key, index := range c.index {
		c.index[key] = index - 1
	}
}

// Len returns the number of items in the cache
func (c *cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache
func (c *cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = []cacheItem[K, V]{}
	c.index = make(map[K]int)
}

// CleanupExpired removes all expired items from the cache
func (c *cache[K, V]) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var validItems []cacheItem[K, V]
	newIndex := make(map[K]int)
	removedCount := 0

	for _, item := range c.items {
		if c.isItemValid(item) {
			newIndex[item.Key] = len(validItems)
			validItems = append(validItems, item)
		} else {
			removedCount++
		}
	}

	c.items = validItems
	c.index = newIndex

	return removedCount
}
