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
	Set(key K, value *V)
	SetWithTTL(key K, value *V, ttl time.Duration)
	Get(key K) (*V, bool)
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

	order []K                    // slice to track order of keys for eviction
	items map[K]*cacheItem[K, V] // map to store actual data for fast access
}

type cacheItem[K comparable, V any] struct {
	Key       K
	Value     *V
	TTL       *time.Duration
	CreatedAt time.Time
	Size      int64
	Index     int // index in the order slice
}

func New[K comparable, V any](config *Config) Cache[K, V] {
	if config == nil {
		config = &Config{}
	}

	return &cache[K, V]{
		size:     config.Size,
		maxItems: config.MaxItems,
		order:    []K{},
		items:    make(map[K]*cacheItem[K, V]),
	}
}

func (c *cache[K, V]) Set(key K, value *V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setItem(key, value, nil)
}

func (c *cache[K, V]) SetWithTTL(key K, value *V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setItem(key, value, &ttl)
}

func (c *cache[K, V]) Get(key K) (*V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, exists := c.items[key]; exists {
		if c.isItemValid(item) {
			return item.Value, true // Item found and valid
		}
	}

	return nil, false // Item not found
}

func (c *cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		// Update current size
		c.currentSize -= item.Size

		// Remove from items map
		delete(c.items, key)

		// Remove from order slice using the stored index
		index := item.Index
		c.order = append(c.order[:index], c.order[index+1:]...)

		// Update indices for all items that come after the deleted item
		for i := index; i < len(c.order); i++ {
			itemKey := c.order[i]
			if existingItem, exists := c.items[itemKey]; exists {
				existingItem.Index = i
				c.items[itemKey] = existingItem
			}
		}
	}
}

// updateOrAddItem updates an existing item or adds a new one
func (c *cache[K, V]) updateOrAddItem(key K, item *cacheItem[K, V]) {
	if _, exists := c.items[key]; exists {
		// Update existing item - keep the same index
		existingItem := c.items[key]
		item.Index = existingItem.Index
		c.items[key] = item
	} else {
		// Add new item to the end
		item.Index = len(c.order)
		c.order = append(c.order, key)
		c.items[key] = item
	}
}

// isItemValid checks if a cache item is valid (not expired)
func (c *cache[K, V]) isItemValid(item *cacheItem[K, V]) bool {
	if item.TTL == nil {
		return true // No TTL means never expires
	}
	return time.Since(item.CreatedAt) < *item.TTL
}

// setItem is a helper method that consolidates the logic for setting cache items
func (c *cache[K, V]) setItem(key K, value *V, ttl *time.Duration) {
	var itemSize int64
	if value != nil {
		itemSize = calculateItemSize(key, *value)
	} else {
		itemSize = calculateItemSize(key, *new(V)) // For nil values, calculate size of zero value
	}

	// If updating existing item, handle size difference
	if existingItem, exists := c.items[key]; exists {
		oldSize := existingItem.Size
		newTotalSize := c.currentSize - oldSize + itemSize

		// Evict items if the update would exceed limits
		for (c.size != nil && newTotalSize > *c.size) ||
			(c.maxItems != nil && int64(len(c.items)) >= *c.maxItems) {
			if len(c.order) <= 1 { // Don't evict the item we're updating
				break
			}
			// Find an item to evict that's not the one we're updating
			if len(c.order) > 0 && c.order[0] == key && len(c.order) > 1 {
				// If the first item is the one we're updating, evict the second
				c.removeItemByKey(c.order[1])
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
			if len(c.order) == 0 {
				break // No items to evict
			}
			c.removeOldestItem()
			newTotalSize = c.currentSize + itemSize
		}

		c.currentSize = newTotalSize
	}

	item := &cacheItem[K, V]{
		Key:       key,
		Value:     value,
		TTL:       ttl,
		CreatedAt: time.Now(),
		Size:      itemSize,
	}

	c.updateOrAddItem(key, item)
}

// removeItemByKey removes an item by its key
func (c *cache[K, V]) removeItemByKey(key K) {
	if item, exists := c.items[key]; exists {
		// Update current size
		c.currentSize -= item.Size

		// Remove from items map
		delete(c.items, key)

		// Remove from order slice using the stored index
		index := item.Index
		c.order = append(c.order[:index], c.order[index+1:]...)

		// Update indices for all items that come after the deleted item
		for i := index; i < len(c.order); i++ {
			itemKey := c.order[i]
			if existingItem, exists := c.items[itemKey]; exists {
				existingItem.Index = i
				c.items[itemKey] = existingItem
			}
		}
	}
}

// removeOldestItem removes the oldest (first) item from the cache
func (c *cache[K, V]) removeOldestItem() {
	if len(c.order) == 0 {
		return
	}

	// Get the oldest item key (first in the order slice)
	oldestKey := c.order[0]
	c.removeItemByKey(oldestKey)
}

// Len returns the number of items in the cache
func (c *cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.order)
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
	c.order = []K{}
	c.items = make(map[K]*cacheItem[K, V])
	c.currentSize = 0
}

// CleanupExpired removes all expired items from the cache
func (c *cache[K, V]) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var validOrder []K
	newItems := make(map[K]*cacheItem[K, V])
	removedCount := 0
	newSize := int64(0)

	for _, key := range c.order {
		if item, exists := c.items[key]; exists {
			if c.isItemValid(item) {
				item.Index = len(validOrder)
				newItems[key] = item
				validOrder = append(validOrder, key)
				newSize += item.Size
			} else {
				removedCount++
			}
		}
	}

	c.order = validOrder
	c.items = newItems
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
