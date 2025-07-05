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
}

type cache[K comparable, V any] struct {
	mu        sync.RWMutex
	size      *int64
	sizeBytes int64
	maxItems  *int64

	order []K                    // slice to track order of keys for eviction
	items map[K]*cacheItem[K, V] // map to store actual data for fast access

	// TTL expiration management
	expirationTimers map[K]*time.Timer // map to track expiration timers for each key
	stopChan         chan struct{}     // channel to stop background cleanup
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

	c := &cache[K, V]{
		size:             config.Size,
		maxItems:         config.MaxItems,
		order:            []K{},
		items:            make(map[K]*cacheItem[K, V]),
		expirationTimers: make(map[K]*time.Timer),
		stopChan:         make(chan struct{}),
	}

	return c
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

	// Cancel existing timer for this key if it exists
	if timer, exists := c.expirationTimers[key]; exists {
		timer.Stop()
	}

	// Set up automatic expiration timer
	if ttl > 0 {
		timer := time.AfterFunc(ttl, func() {
			c.expireKey(key)
		})
		c.expirationTimers[key] = timer
	}
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
		c.sizeBytes -= item.Size

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

	// Cancel and remove expiration timer if it exists
	if timer, exists := c.expirationTimers[key]; exists {
		timer.Stop()
		delete(c.expirationTimers, key)
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
		newTotalSize := c.sizeBytes - oldSize + itemSize

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
			newTotalSize = c.sizeBytes - oldSize + itemSize
		}

		c.sizeBytes = newTotalSize
	} else {
		// Adding new item - evict items if necessary before adding
		newTotalSize := c.sizeBytes + itemSize

		for (c.size != nil && newTotalSize > *c.size) ||
			(c.maxItems != nil && int64(len(c.items)) >= *c.maxItems) {
			if len(c.order) == 0 {
				break // No items to evict
			}
			c.removeOldestItem()
			newTotalSize = c.sizeBytes + itemSize
		}

		c.sizeBytes = newTotalSize
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
		c.sizeBytes -= item.Size

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

	// Cancel and remove expiration timer if it exists
	if timer, exists := c.expirationTimers[key]; exists {
		timer.Stop()
		delete(c.expirationTimers, key)
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

// expireKey removes an expired key asynchronously
func (c *cache[K, V]) expireKey(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove the item if it still exists and is expired
	if item, exists := c.items[key]; exists {
		if !c.isItemValid(item) {
			c.removeItemByKey(key)
		}
	}

	// Clean up the timer
	if timer, exists := c.expirationTimers[key]; exists {
		timer.Stop()
		delete(c.expirationTimers, key)
	}
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
