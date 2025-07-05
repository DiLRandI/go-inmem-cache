package goinmemcache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBasicSetGet(t *testing.T) {
	cache := New[string, string](&Config{})

	value := "test-value"
	cache.Set("test-key", &value)

	if result, found := cache.Get("test-key"); !found || *result != "test-value" {
		t.Errorf("Expected to find 'test-value', got %v, found: %v", result, found)
	}

	// Test non-existent key
	if _, found := cache.Get("non-existent"); found {
		t.Errorf("Expected not to find non-existent key")
	}
}

func TestBasicDelete(t *testing.T) {
	cache := New[string, string](&Config{})

	value := "test-value"
	cache.Set("test-key", &value)

	// Verify it exists
	if _, found := cache.Get("test-key"); !found {
		t.Errorf("Expected to find test-key before deletion")
	}

	// Delete it
	cache.Delete("test-key")

	// Verify it's gone
	if _, found := cache.Get("test-key"); found {
		t.Errorf("Expected test-key to be deleted")
	}
}

func TestTTLExpiration(t *testing.T) {
	cache := New[string, string](&Config{})
	ttl := 50 * time.Millisecond

	// Set value with TTL
	value := "expires-soon"
	cache.SetWithTTL("ttl-key", &value, ttl)

	// Should be available immediately
	if _, found := cache.Get("ttl-key"); !found {
		t.Errorf("Key should be available immediately")
	}

	// Wait for TTL to expire + buffer for timer execution
	time.Sleep(ttl + 100*time.Millisecond)

	// Should be expired now
	if _, found := cache.Get("ttl-key"); found {
		t.Errorf("Key should be expired")
	}
}

func TestTTLUpdate(t *testing.T) {
	cache := New[string, string](&Config{})

	// Set with long TTL
	value := "long-lived"
	cache.SetWithTTL("update-key", &value, 1*time.Hour)

	// Verify it exists
	if _, found := cache.Get("update-key"); !found {
		t.Errorf("Key should exist after setting with long TTL")
	}

	// Update with short TTL
	cache.SetWithTTL("update-key", &value, 10*time.Millisecond)

	// Wait for new TTL to expire
	time.Sleep(50 * time.Millisecond)

	// Should be expired now
	if _, found := cache.Get("update-key"); found {
		t.Errorf("Key should be expired after TTL update")
	}
}

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
				cache.Set(key, &value)

				// Get value
				if val, ok := cache.Get(key); ok && *val != value {
					t.Errorf("Expected %d, got %d", value, *val)
				}

				// Delete some values
				if j%10 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Basic functionality test after concurrent access
	testKey := "test-key"
	testValue := 42
	cache.Set(testKey, &testValue)
	if val, found := cache.Get(testKey); !found || *val != testValue {
		t.Errorf("Basic cache functionality should work after concurrent access")
	}
}

func TestItemLimitEviction(t *testing.T) {
	// Create cache with max 2 items
	maxItems := int64(2)
	config := &Config{MaxItems: &maxItems}
	cache := New[string, int](config)

	// Add items
	item1, item2, item3 := 1, 2, 3
	cache.Set("item1", &item1)
	cache.Set("item2", &item2)

	// Both items should exist
	if _, found := cache.Get("item1"); !found {
		t.Errorf("item1 should exist")
	}
	if _, found := cache.Get("item2"); !found {
		t.Errorf("item2 should exist")
	}

	// Add third item - should evict oldest (item1)
	cache.Set("item3", &item3)

	// item1 should be gone, item2 and item3 should exist
	if _, found := cache.Get("item1"); found {
		t.Errorf("item1 should have been evicted")
	}
	if _, found := cache.Get("item2"); !found {
		t.Errorf("item2 should still exist")
	}
	if _, found := cache.Get("item3"); !found {
		t.Errorf("item3 should exist")
	}
}

func TestSizeBasedEviction(t *testing.T) {
	// Create cache with small size limit
	maxSize := int64(100)
	config := &Config{Size: &maxSize}
	cache := New[string, string](config)

	// Add items
	item1 := "short"
	cache.Set("item1", &item1)

	item2 := "medium length value"
	cache.Set("item2", &item2)

	item3 := "another value"
	cache.Set("item3", &item3)

	// Add a larger item that should trigger eviction
	item4 := "this is a much longer string that should trigger eviction of some items"
	cache.Set("item4", &item4)

	// The last item should definitely exist
	if val, found := cache.Get("item4"); !found || *val != item4 {
		t.Errorf("Expected to be able to retrieve the last added item")
	}

	// Test that eviction happened by checking if some earlier items are gone
	item1Found := false
	if _, found := cache.Get("item1"); found {
		item1Found = true
	}
	item2Found := false
	if _, found := cache.Get("item2"); found {
		item2Found = true
	}
	item3Found := false
	if _, found := cache.Get("item3"); found {
		item3Found = true
	}

	// Not all earlier items should still exist (some should be evicted)
	totalEarlierFound := 0
	if item1Found {
		totalEarlierFound++
	}
	if item2Found {
		totalEarlierFound++
	}
	if item3Found {
		totalEarlierFound++
	}

	if totalEarlierFound == 3 {
		t.Errorf("Expected some items to be evicted due to size limit, but all remain")
	}
}

func TestDifferentTypes(t *testing.T) {
	// Test with integer keys and float values
	intCache := New[int, float64](&Config{})

	value1 := 3.14
	value2 := 2.71
	intCache.Set(1, &value1)
	intCache.Set(2, &value2)

	if val, found := intCache.Get(1); !found || *val != 3.14 {
		t.Errorf("Expected 3.14, got %v", val)
	}

	if val, found := intCache.Get(2); !found || *val != 2.71 {
		t.Errorf("Expected 2.71, got %v", val)
	}

	// Test with struct values
	type Person struct {
		Name string
		Age  int
	}

	structCache := New[string, Person](&Config{})

	person1 := Person{Name: "Alice", Age: 30}
	person2 := Person{Name: "Bob", Age: 25}

	structCache.Set("person1", &person1)
	structCache.Set("person2", &person2)

	if val, found := structCache.Get("person1"); !found || val.Name != "Alice" || val.Age != 30 {
		t.Errorf("Expected Alice, 30, got %v", val)
	}

	if val, found := structCache.Get("person2"); !found || val.Name != "Bob" || val.Age != 25 {
		t.Errorf("Expected Bob, 25, got %v", val)
	}
}

func TestNilValues(t *testing.T) {
	cache := New[string, string](&Config{})

	// Test setting nil value
	var nilValue *string
	cache.Set("nil-key", nilValue)

	if val, found := cache.Get("nil-key"); !found || val != nil {
		t.Errorf("Expected nil value to be stored and retrieved")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	cache := New[string, string](&Config{})

	// Deleting non-existent key should not panic
	cache.Delete("non-existent")

	// Add an item and delete it twice
	value := "test"
	cache.Set("test-key", &value)
	cache.Delete("test-key")
	cache.Delete("test-key") // Second delete should be safe
}

func TestZeroTTL(t *testing.T) {
	cache := New[string, string](&Config{})

	// Test zero TTL (should expire immediately)
	value := "expires-now"
	cache.SetWithTTL("zero-ttl", &value, 0)

	// Give the timer a moment to fire
	time.Sleep(10 * time.Millisecond)

	// Should be expired immediately
	if _, found := cache.Get("zero-ttl"); found {
		t.Errorf("Item with zero TTL should be expired immediately")
	}
}
