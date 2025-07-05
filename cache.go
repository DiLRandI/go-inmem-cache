package goinmemcache

import (
	"container/heap"
	"reflect"
	"sync"
	"time"
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
	Clear()
	Close()
	CleanupExpired() // Manually trigger cleanup of expired items
}

// listNode represents a node in the doubly-linked list for LRU ordering
type listNode[K comparable] struct {
	key  K
	prev *listNode[K]
	next *listNode[K]
}

type cache[K comparable, V any] struct {
	mu        sync.RWMutex
	size      *int64
	sizeBytes int64
	maxItems  *int64

	// Doubly-linked list for O(1) LRU operations
	head *listNode[K] // dummy head node
	tail *listNode[K] // dummy tail node

	items map[K]*cacheItem[K, V] // map to store actual data for fast access

	// Optimized TTL expiration management
	expirationQueue []*expirationEntry[K] // min-heap of expiration entries
	expirationMap   map[K]*expirationEntry[K] // fast lookup for expiration entries
	cleanupTicker   *time.Ticker           // single ticker for all TTL cleanup
	stopChan        chan struct{}          // channel to stop background cleanup

	// Size calculation optimization
	keyTypeSize   int64 // cached size for key type
	valueTypeSize int64 // cached size for value type (for fixed-size types)
	isKeyString   bool  // whether key type is string
	isValueString bool  // whether value type is string
}

// expirationEntry represents an item in the expiration queue
type expirationEntry[K comparable] struct {
	key        K
	expireTime time.Time
	index      int // index in the heap
}

type cacheItem[K comparable, V any] struct {
	Value     *V
	TTL       *time.Duration
	CreatedAt time.Time
	Size      int64
	Node      *listNode[K] // reference to the node in the doubly-linked list
}

// expirationHeap implements heap.Interface for expiration entries
type expirationHeap[K comparable] []*expirationEntry[K]

func (h expirationHeap[K]) Len() int           { return len(h) }
func (h expirationHeap[K]) Less(i, j int) bool { return h[i].expireTime.Before(h[j].expireTime) }
func (h expirationHeap[K]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *expirationHeap[K]) Push(x interface{}) {
	entry := x.(*expirationEntry[K])
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *expirationHeap[K]) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	entry.index = -1
	*h = old[0 : n-1]
	return entry
}

func New[K comparable, V any](config *Config) Cache[K, V] {
	if config == nil {
		config = &Config{}
	}

	// Create dummy head and tail nodes for the doubly-linked list
	head := &listNode[K]{}
	tail := &listNode[K]{}
	head.next = tail
	tail.prev = head

	// Pre-calculate type information for size calculations
	var zeroK K
	var zeroV V
	keyType := reflect.TypeOf(zeroK)
	valueType := reflect.TypeOf(zeroV)

	isKeyString := keyType.Kind() == reflect.String
	isValueString := valueType.Kind() == reflect.String

	var keyTypeSize, valueTypeSize int64
	if !isKeyString {
		keyTypeSize = int64(keyType.Size())
	}
	if !isValueString && valueType.Kind() != reflect.Slice && valueType.Kind() != reflect.Map && valueType.Kind() != reflect.Ptr {
		valueTypeSize = int64(valueType.Size())
	}

	c := &cache[K, V]{
		size:             config.Size,
		maxItems:         config.MaxItems,
		head:             head,
		tail:             tail,
		items:            make(map[K]*cacheItem[K, V]),
		expirationQueue:  make([]*expirationEntry[K], 0),
		expirationMap:    make(map[K]*expirationEntry[K]),
		stopChan:         make(chan struct{}),
		keyTypeSize:      keyTypeSize,
		valueTypeSize:    valueTypeSize,
		isKeyString:      isKeyString,
		isValueString:    isValueString,
	}

	// Start the cleanup ticker for periodic expiration check
	c.cleanupTicker = time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanupExpiredItems()
			case <-c.stopChan:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()

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

	// Remove any existing expiration entry for this key
	c.removeExpirationEntry(key)

	// Add new expiration entry if TTL is positive
	if ttl > 0 {
		expireTime := time.Now().Add(ttl)
		c.addExpirationEntry(key, expireTime)
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

	// Single lookup for item
	item, itemExists := c.items[key]

	if itemExists {
		// Update current size
		c.sizeBytes -= item.Size

		// Remove from items map
		delete(c.items, key)

		// Remove from doubly-linked list
		c.removeNode(item.Node)

		// Remove from expiration queue
		c.removeExpirationEntry(key)
	}
}

// updateOrAddItem updates an existing item or adds a new one
func (c *cache[K, V]) updateOrAddItem(key K, item *cacheItem[K, V]) {
	if existingItem, exists := c.items[key]; exists {
		// Update existing item - reuse the same node and move to tail
		existingItem.Value = item.Value
		existingItem.TTL = item.TTL
		existingItem.CreatedAt = item.CreatedAt
		existingItem.Size = item.Size
		c.moveToTail(existingItem.Node)
		c.items[key] = existingItem
	} else {
		// Add new item - create new node and add to tail
		node := &listNode[K]{key: key}
		item.Node = node
		c.addToTail(node)
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
		itemSize = c.fastCalculateItemSize(key, *value)
	} else {
		itemSize = c.fastCalculateItemSize(key, *new(V)) // For nil values, calculate size of zero value
	}

	// If updating existing item, handle size difference
	if existingItem, exists := c.items[key]; exists {
		oldSize := existingItem.Size
		newTotalSize := c.sizeBytes - oldSize + itemSize

		// Evict items if the update would exceed limits
		for (c.size != nil && newTotalSize > *c.size) ||
			(c.maxItems != nil && int64(len(c.items)) >= *c.maxItems) {
			if c.listSize() <= 1 { // Don't evict the item we're updating
				break
			}
			// Find an item to evict that's not the one we're updating
			if !c.isEmpty() && c.head.next.key == key && c.listSize() > 1 {
				// If the first item is the one we're updating, evict the second
				secondNode := c.head.next.next
				if secondNode != c.tail {
					c.removeItemByKey(secondNode.key)
				}
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
			if c.isEmpty() {
				break // No items to evict
			}
			c.removeOldestItem()
			newTotalSize = c.sizeBytes + itemSize
		}

		c.sizeBytes = newTotalSize
	}

	item := &cacheItem[K, V]{
		Value:     value,
		TTL:       ttl,
		CreatedAt: time.Now(),
		Size:      itemSize,
	}

	c.updateOrAddItem(key, item)

	// Manage expiration queue for TTL
	if ttl != nil {
		if *ttl > 0 {
			expireTime := time.Now().Add(*ttl)
			c.addExpirationEntry(key, expireTime)
		} else {
			// Zero or negative TTL removes the item immediately
			c.removeItemByKey(key)
		}
	}
}

// removeItemByKey removes an item by its key
func (c *cache[K, V]) removeItemByKey(key K) {
	// Single lookup for item
	item, itemExists := c.items[key]

	if itemExists {
		// Update current size
		c.sizeBytes -= item.Size

		// Remove from items map
		delete(c.items, key)

		// Remove from doubly-linked list
		c.removeNode(item.Node)

		// Remove from expiration queue
		c.removeExpirationEntry(key)
	}
}

// removeOldestItem removes the oldest (first) item from the cache
func (c *cache[K, V]) removeOldestItem() {
	if c.isEmpty() {
		return
	}

	// Get the oldest item key (first in the doubly-linked list)
	oldestNode := c.removeHead()
	if oldestNode != nil {
		c.removeItemByKey(oldestNode.key)
	}
}

// expireKey removes an expired key (called from background cleanup)
func (c *cache[K, V]) expireKey(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Single lookup for item
	item, itemExists := c.items[key]

	// Remove the item if it still exists and is expired
	if itemExists && !c.isItemValid(item) {
		c.removeItemByKey(key)
	}
}

// cleanupExpiredItems removes expired items from the cache
func (c *cache[K, V]) cleanupExpiredItems() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	h := (*expirationHeap[K])(&c.expirationQueue)

	// Remove expired items from the front of the heap
	for h.Len() > 0 {
		entry := (*h)[0]
		if entry.expireTime.After(now) {
			break // No more expired items
		}

		// Remove expired entry from heap
		heap.Pop(h)
		delete(c.expirationMap, entry.key)

		// Check if item still exists and is expired
		if item, exists := c.items[entry.key]; exists {
			if !c.isItemValid(item) {
				c.removeItemByKey(entry.key)
			}
		}
	}
}

// addExpirationEntry adds an entry to the expiration queue
func (c *cache[K, V]) addExpirationEntry(key K, expireTime time.Time) {
	entry := &expirationEntry[K]{
		key:        key,
		expireTime: expireTime,
	}
	h := (*expirationHeap[K])(&c.expirationQueue)
	heap.Push(h, entry)
	c.expirationMap[key] = entry
}

// removeExpirationEntry removes an entry from the expiration queue
func (c *cache[K, V]) removeExpirationEntry(key K) {
	if entry, exists := c.expirationMap[key]; exists {
		h := (*expirationHeap[K])(&c.expirationQueue)
		if entry.index >= 0 && entry.index < len(*h) {
			heap.Remove(h, entry.index)
		}
		delete(c.expirationMap, key)
	}
}

// fastCalculateItemSize is an optimized version of calculateItemSize that uses cached type information
func (c *cache[K, V]) fastCalculateItemSize(key K, value V) int64 {
	var size int64

	// Calculate key size using cached information
	if c.isKeyString {
		// For strings, we need to calculate the actual length
		size += int64(len(any(key).(string)))
	} else {
		// For other types, use the cached size
		size += c.keyTypeSize
	}

	// Calculate value size using cached information
	if c.isValueString {
		// For strings, calculate actual length
		size += int64(len(any(value).(string)))
	} else if c.valueTypeSize > 0 {
		// For fixed-size types, use cached size
		size += c.valueTypeSize
	} else {
		// For complex types (slices, maps, pointers), fall back to reflection
		size += c.calculateComplexValueSize(value)
	}

	// Add overhead for the cache item struct itself (pre-calculated constants)
	size += 32 // time.Time (24) + int64 Size field (8)
	size += 8  // TTL pointer
	size += 8  // Node pointer

	return size
}

// calculateComplexValueSize handles complex types that need reflection
func (c *cache[K, V]) calculateComplexValueSize(value V) int64 {
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)

	switch valueType.Kind() {
	case reflect.Slice:
		if valueVal.Len() == 0 {
			return 0
		}
		return int64(valueVal.Len()) * int64(valueType.Elem().Size())
	case reflect.Map:
		if valueVal.Len() == 0 {
			return 0
		}
		return int64(valueVal.Len()) * (int64(valueType.Key().Size()) + int64(valueType.Elem().Size()))
	case reflect.Ptr:
		if valueVal.IsNil() {
			return 8 // pointer size
		}
		return int64(valueType.Elem().Size())
	default:
		return int64(valueType.Size())
	}
}

// moveToTail moves a node to the tail (most recently used position)
func (c *cache[K, V]) moveToTail(node *listNode[K]) {
	c.removeNode(node)
	c.addToTail(node)
}

// addToTail adds a node right before the tail (most recently used position)
func (c *cache[K, V]) addToTail(node *listNode[K]) {
	prev := c.tail.prev
	prev.next = node
	node.prev = prev
	node.next = c.tail
	c.tail.prev = node
}

// removeNode removes a node from the doubly-linked list
func (c *cache[K, V]) removeNode(node *listNode[K]) {
	prev := node.prev
	next := node.next
	prev.next = next
	next.prev = prev
}

// removeTail removes and returns the least recently used node (right before tail)
func (c *cache[K, V]) removeHead() *listNode[K] {
	head := c.head.next
	if head == c.tail {
		return nil // list is empty
	}
	c.removeNode(head)
	return head
}

// isEmpty checks if the list is empty
func (c *cache[K, V]) isEmpty() bool {
	return c.head.next == c.tail
}

// listSize returns the number of nodes in the list
func (c *cache[K, V]) listSize() int {
	count := 0
	current := c.head.next
	for current != c.tail {
		count++
		current = current.next
	}
	return count
}

// Len returns the number of items currently in the cache
func (c *cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache
func (c *cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Clear all maps and reset size
	c.items = make(map[K]*cacheItem[K, V])
	c.expirationMap = make(map[K]*expirationEntry[K])
	c.expirationQueue = make([]*expirationEntry[K], 0)
	c.sizeBytes = 0
	
	// Reset doubly-linked list
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Close stops the background cleanup goroutine and releases resources
func (c *cache[K, V]) Close() {
	close(c.stopChan)
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
}

// CleanupExpired manually triggers cleanup of expired items
func (c *cache[K, V]) CleanupExpired() {
	c.cleanupExpiredItems()
}
