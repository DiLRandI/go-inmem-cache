package goinmemcache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentAccess(t *testing.T) {
	maxItems := int64(100)
	cache := New[string, int](&Config{MaxItems: &maxItems})

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperationsPerGoroutine := 100

	// Launch multiple goroutines performing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := id*1000 + j

				// Set value
				cache.Set(key, value)

				// Get value
				if val, ok := cache.Get(key); ok && val != value {
					t.Errorf("Expected %d, got %d", value, val)
				}

				// Delete some values
				if j%10 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Final sanity check
	finalLen := cache.Len()
	if finalLen > int(maxItems) {
		t.Errorf("Cache exceeded max items: %d > %d", finalLen, maxItems)
	}
}

func TestConcurrentTTL(t *testing.T) {
	cache := New[string, string](&Config{})
	ttl := 50 * time.Millisecond

	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("ttl-key-%d", id)
			value := fmt.Sprintf("value-%d", id)

			cache.SetWithTTL(key, value, ttl)

			// Should be available immediately
			if _, ok := cache.Get(key); !ok {
				t.Errorf("Key %s should be available immediately", key)
			}

			// Wait for TTL to expire
			time.Sleep(ttl + 10*time.Millisecond)

			// Should be expired now
			if _, ok := cache.Get(key); ok {
				t.Errorf("Key %s should be expired", key)
			}
		}(i)
	}

	wg.Wait()
}

func TestCleanupExpired(t *testing.T) {
	cache := New[string, string](&Config{})
	ttl := 50 * time.Millisecond

	// Add some items with TTL
	cache.SetWithTTL("key1", "value1", ttl)
	cache.SetWithTTL("key2", "value2", ttl)
	cache.Set("key3", "value3") // No TTL

	// Initial length should be 3
	if cache.Len() != 3 {
		t.Errorf("Expected 3 items, got %d", cache.Len())
	}

	// Wait for TTL to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Items should still be in cache (not automatically cleaned)
	if cache.Len() != 3 {
		t.Errorf("Expected 3 items before cleanup, got %d", cache.Len())
	}

	// Clean up expired items
	removedCount := cache.CleanupExpired()

	// Should have removed 2 expired items
	if removedCount != 2 {
		t.Errorf("Expected 2 removed items, got %d", removedCount)
	}

	// Should have 1 item left
	if cache.Len() != 1 {
		t.Errorf("Expected 1 item after cleanup, got %d", cache.Len())
	}

	// The remaining item should be key3
	if val, ok := cache.Get("key3"); !ok || val != "value3" {
		t.Errorf("Expected key3 to still exist with value 'value3'")
	}
}

func TestSizeBasedEviction(t *testing.T) {
	// Create cache with 100 bytes limit
	maxSize := int64(100)
	config := &Config{Size: &maxSize}
	cache := New[string, string](config)

	// Add first item
	cache.Set("item1", "short")
	size1 := cache.CurrentSize()
	fmt.Printf("After item1: size=%d bytes, len=%d\n", size1, cache.Len())

	// Add second item
	cache.Set("item2", "medium length value")
	size2 := cache.CurrentSize()
	fmt.Printf("After item2: size=%d bytes, len=%d\n", size2, cache.Len())

	// Add third item
	cache.Set("item3", "another value")
	size3 := cache.CurrentSize()
	fmt.Printf("After item3: size=%d bytes, len=%d\n", size3, cache.Len())

	// Add a moderately large item that should trigger some eviction
	cache.Set("item4", "this is a longer string that should trigger eviction")

	finalSize := cache.CurrentSize()
	finalLen := cache.Len()

	fmt.Printf("After adding item4: size=%d bytes, len=%d\n", finalSize, finalLen)

	// Size should be within reasonable bounds of the limit
	tolerance := maxSize + (maxSize / 2) // 150% of the limit
	if finalSize > tolerance {
		t.Errorf("Cache size %d significantly exceeds limit %d (tolerance: %d)", finalSize, maxSize, tolerance)
	}

	// Some eviction should have happened
	if finalLen >= 4 {
		t.Errorf("Expected some items to be evicted, but all %d items remain", finalLen)
	}
}

func TestSizeTracking(t *testing.T) {
	cache := New[string, string](nil)

	// Initial size should be 0
	if cache.CurrentSize() != 0 {
		t.Errorf("Initial cache size should be 0, got %d", cache.CurrentSize())
	}

	// Add an item
	cache.Set("key1", "value1")
	size1 := cache.CurrentSize()

	if size1 <= 0 {
		t.Errorf("Cache size should be positive after adding item, got %d", size1)
	}

	// Add another item
	cache.Set("key2", "longer value string")
	size2 := cache.CurrentSize()

	if size2 <= size1 {
		t.Errorf("Cache size should increase after adding larger item: %d <= %d", size2, size1)
	}

	// Delete an item
	cache.Delete("key1")
	size3 := cache.CurrentSize()

	if size3 >= size2 {
		t.Errorf("Cache size should decrease after deleting item: %d >= %d", size3, size2)
	}

	// Clear cache
	cache.Clear()
	if cache.CurrentSize() != 0 {
		t.Errorf("Cache size should be 0 after clear, got %d", cache.CurrentSize())
	}
}

func TestBothSizeAndItemLimits(t *testing.T) {
	// Create cache with both size and item limits
	maxSize := int64(200)
	maxItems := int64(2)
	config := &Config{
		Size:     &maxSize,
		MaxItems: &maxItems,
	}
	cache := New[string, string](config)

	// Add items
	cache.Set("item1", "value1")
	cache.Set("item2", "value2")

	// Should have 2 items
	if cache.Len() != 2 {
		t.Errorf("Expected 2 items, got %d", cache.Len())
	}

	// Add third item - should trigger item-based eviction
	cache.Set("item3", "value3")

	// Should still have 2 items (item1 evicted)
	if cache.Len() != 2 {
		t.Errorf("Expected 2 items after item limit eviction, got %d", cache.Len())
	}

	// item1 should be gone
	if _, found := cache.Get("item1"); found {
		t.Errorf("item1 should have been evicted due to item limit")
	}

	// item2 and item3 should exist
	if _, found := cache.Get("item2"); !found {
		t.Errorf("item2 should still exist")
	}
	if _, found := cache.Get("item3"); !found {
		t.Errorf("item3 should exist")
	}
}
