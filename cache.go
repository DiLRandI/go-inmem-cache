package goinmemcache

import (
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type Config struct {
	Size     *int64
	MaxItems *int64
}

type Cache[K comparable, V any] interface {
	Set(key K, value V)
	SetWithTTL(key K, value V, ttl time.Duration)
	Get(key K) (V, bool)
	Delete(key K)
	Len() int
	CurrentSize() int64
	Clear()
	CleanupExpired() int
}

type cache[K comparable, V any] struct {
	mu          sync.RWMutex
	size        *int64
	currentSize int64
	maxItems    *int64

	items []cacheItem[K, V]
	index map[K]int
}

type cacheItem[K comparable, V any] struct {
	Key       K
	Value     V
	TTL       *time.Duration
	CreatedAt time.Time
	Size      int64
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

func (c *cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setItem(key, value, &ttl)
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
		// Update current size
		c.currentSize -= c.items[index].Size

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

// setItem is a helper method that consolidates the logic for setting cache items
func (c *cache[K, V]) setItem(key K, value V, ttl *time.Duration) {
	itemSize := calculateItemSize(key, value)

	// If updating existing item, handle size difference
	if index, exists := c.index[key]; exists {
		oldSize := c.items[index].Size
		newTotalSize := c.currentSize - oldSize + itemSize

		// Evict items if the update would exceed limits
		for (c.size != nil && newTotalSize > *c.size) ||
			(c.maxItems != nil && int64(len(c.items)) >= *c.maxItems) {
			if len(c.items) <= 1 { // Don't evict the item we're updating
				break
			}
			// Find an item to evict that's not the one we're updating
			if c.items[0].Key == key && len(c.items) > 1 {
				// If the first item is the one we're updating, evict the second
				oldestItem := c.items[1]
				c.currentSize -= oldestItem.Size
				delete(c.index, oldestItem.Key)
				c.items = append(c.items[:1], c.items[2:]...)
				// Update indices
				for k, idx := range c.index {
					if idx > 1 {
						c.index[k] = idx - 1
					}
				}
			} else {
				c.removeOldestItem()
			}
			newTotalSize = c.currentSize - oldSize + itemSize
		}

		c.currentSize = newTotalSize
	} else {
		// Adding new item - evict items if necessary before adding
		newTotalSize := c.currentSize + itemSize

		for (c.size != nil && newTotalSize > *c.size) ||
			(c.maxItems != nil && int64(len(c.items)) >= *c.maxItems) {
			if len(c.items) == 0 {
				break // No items to evict
			}
			c.removeOldestItem()
			newTotalSize = c.currentSize + itemSize
		}

		c.currentSize = newTotalSize
	}

	item := cacheItem[K, V]{
		Key:       key,
		Value:     value,
		TTL:       ttl,
		CreatedAt: time.Now(),
		Size:      itemSize,
	}

	c.updateOrAddItem(key, item)
}

// removeOldestItem removes the oldest (first) item from the cache
func (c *cache[K, V]) removeOldestItem() {
	if len(c.items) == 0 {
		return
	}

	// Get the oldest item (first in the slice)
	oldestItem := c.items[0]

	// Update current size
	c.currentSize -= oldestItem.Size

	// Remove from index
	delete(c.index, oldestItem.Key)

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

// CurrentSize returns the current memory usage in bytes
func (c *cache[K, V]) CurrentSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// Clear removes all items from the cache
func (c *cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = []cacheItem[K, V]{}
	c.index = make(map[K]int)
	c.currentSize = 0
}

// CleanupExpired removes all expired items from the cache
func (c *cache[K, V]) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var validItems []cacheItem[K, V]
	newIndex := make(map[K]int)
	removedCount := 0
	newSize := int64(0)

	for _, item := range c.items {
		if c.isItemValid(item) {
			newIndex[item.Key] = len(validItems)
			validItems = append(validItems, item)
			newSize += item.Size
		} else {
			removedCount++
		}
	}

	c.items = validItems
	c.index = newIndex
	c.currentSize = newSize

	return removedCount
}

// calculateItemSize estimates the memory size of a cache item
func calculateItemSize[K comparable, V any](key K, value V) int64 {
	var size int64

	// Calculate key size
	keyType := reflect.TypeOf(key)
	if keyType.Kind() == reflect.String {
		size += int64(len(reflect.ValueOf(key).String()))
	} else {
		size += int64(keyType.Size())
	}

	// Calculate value size
	valueType := reflect.TypeOf(value)
	switch valueType.Kind() {
	case reflect.String:
		size += int64(len(reflect.ValueOf(value).String()))
	case reflect.Slice:
		valueVal := reflect.ValueOf(value)
		size += int64(valueVal.Len()) * int64(valueType.Elem().Size())
	case reflect.Map:
		valueVal := reflect.ValueOf(value)
		size += int64(valueVal.Len()) * (int64(valueType.Key().Size()) + int64(valueType.Elem().Size()))
	case reflect.Ptr:
		if !reflect.ValueOf(value).IsNil() {
			size += int64(valueType.Elem().Size())
		}
	default:
		size += int64(valueType.Size())
	}

	// Add overhead for the cache item struct itself
	size += int64(unsafe.Sizeof(time.Time{})) // CreatedAt
	size += 8                                 // Size field
	if reflect.TypeOf((*time.Duration)(nil)).Elem().Size() > 0 {
		size += 8 // TTL pointer
	}

	return size
}
