// Package goinmemcache provides a high-performance, thread-safe, generic in-memory cache
// implementation with TTL (Time-To-Live) support and dual eviction strategies.
//
// The cache supports:
//   - Generic types for both keys (comparable) and values (any type)
//   - Thread-safe concurrent access using sync.RWMutex
//   - TTL support with automatic expiration
//   - FIFO eviction based on item count or memory size limits
//   - Memory usage tracking and reporting
//   - Manual cleanup of expired items
//
// Basic usage:
//
//	// Create a cache
//	myCache := goinmemcache.New[string, string](nil)
//
//	// Set a value
//	value := "hello world"
//	myCache.Set("key1", &value)
//
//	// Set with TTL
//	myCache.SetWithTTL("key2", &value, 5*time.Minute)
//
//	// Get a value
//	if valuePtr, found := myCache.Get("key1"); found {
//		fmt.Printf("Value: %s\n", *valuePtr)
//	}
//
// With size limits:
//
//	maxSize := int64(1024 * 1024) // 1MB limit
//	maxItems := int64(100)        // 100 item limit
//	config := &goinmemcache.Config{
//		Size:     &maxSize,
//		MaxItems: &maxItems,
//	}
//	cache := goinmemcache.New[string, string](config)
package goinmemcache
