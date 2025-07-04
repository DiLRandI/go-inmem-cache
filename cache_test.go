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

			cache.SetWithTTL(key, value, &ttl)

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
	cache.SetWithTTL("key1", "value1", &ttl)
	cache.SetWithTTL("key2", "value2", &ttl)
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
