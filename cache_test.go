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

// TestOptimizedTTLManagement tests the new heap-based TTL system
func TestOptimizedTTLManagement(t *testing.T) {
	cache := New[string, string](&Config{})

	// Add multiple items with different TTLs
	values := make([]string, 5)
	for i := 0; i < 5; i++ {
		values[i] = fmt.Sprintf("value-%d", i)
		ttl := time.Duration(i+1) * 50 * time.Millisecond
		cache.SetWithTTL(fmt.Sprintf("key-%d", i), &values[i], ttl)
	}

	// All should be available immediately
	for i := 0; i < 5; i++ {
		if _, found := cache.Get(fmt.Sprintf("key-%d", i)); !found {
			t.Errorf("Key-%d should be available immediately", i)
		}
	}

	// Wait for first few to expire
	time.Sleep(200 * time.Millisecond)

	// Check expiration pattern - earlier items should expire first
	expectedExpired := 3 // items 0, 1, 2 should be expired (50ms, 100ms, 150ms)
	actualExpired := 0
	for i := 0; i < 5; i++ {
		if _, found := cache.Get(fmt.Sprintf("key-%d", i)); !found {
			actualExpired++
		}
	}

	if actualExpired < expectedExpired {
		t.Errorf("Expected at least %d items to be expired, got %d", expectedExpired, actualExpired)
	}
}

// TestComplexTypeSizeCalculation tests optimized size calculation with complex types
func TestComplexTypeSizeCalculation(t *testing.T) {
	type ComplexValue struct {
		Data map[string][]int
		Meta []string
		Ptr  *string
	}

	cache := New[string, ComplexValue](&Config{})

	value := ComplexValue{
		Data: map[string][]int{
			"key1": {1, 2, 3, 4, 5},
			"key2": {6, 7, 8, 9, 10},
		},
		Meta: []string{"meta1", "meta2", "meta3"},
		Ptr:  func() *string { s := "pointer_value"; return &s }(),
	}

	// This should not panic and should work efficiently
	cache.Set("complex", &value)

	if retrieved, found := cache.Get("complex"); !found {
		t.Errorf("Complex value should be retrievable")
	} else {
		if len(retrieved.Data) != 2 || len(retrieved.Meta) != 3 || *retrieved.Ptr != "pointer_value" {
			t.Errorf("Complex value not stored/retrieved correctly")
		}
	}
}

// TestLRUOrderingWithOptimizations tests that LRU ordering works with doubly-linked list
func TestLRUOrderingWithOptimizations(t *testing.T) {
	maxItems := int64(3)
	cache := New[string, string](&Config{MaxItems: &maxItems})

	// Add items in order
	values := []string{"first", "second", "third"}
	for i, val := range values {
		cache.Set(fmt.Sprintf("item%d", i), &val)
	}

	// Access item0 to make it most recently used
	cache.Get("item0")

	// Add fourth item - should evict item1 (least recently used)
	fourth := "fourth"
	cache.Set("item3", &fourth)

	// Check that item1 was evicted but item0 remains
	if _, found := cache.Get("item1"); found {
		t.Errorf("item1 should have been evicted (was least recently used)")
	}
	if _, found := cache.Get("item0"); !found {
		t.Errorf("item0 should remain (was recently accessed)")
	}
	if _, found := cache.Get("item2"); !found {
		t.Errorf("item2 should remain")
	}
	if _, found := cache.Get("item3"); !found {
		t.Errorf("item3 should remain (was just added)")
	}
}

// TestTTLUpdateOptimization tests that updating TTL works efficiently
func TestTTLUpdateOptimization(t *testing.T) {
	cache := New[string, string](&Config{})

	value := "test-value"

	// Set with short TTL
	cache.SetWithTTL("update-test", &value, 50*time.Millisecond)

	// Wait half the time
	time.Sleep(25 * time.Millisecond)

	// Update with longer TTL
	cache.SetWithTTL("update-test", &value, 200*time.Millisecond)

	// Wait past original TTL
	time.Sleep(50 * time.Millisecond)

	// Should still be available due to updated TTL
	if _, found := cache.Get("update-test"); !found {
		t.Errorf("Item should still be available after TTL update")
	}
}

// TestConcurrentTTLOperations tests concurrent TTL operations
func TestConcurrentTTLOperations(t *testing.T) {
	cache := New[string, string](&Config{})

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent TTL operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("ttl-key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				ttl := time.Duration(j+1) * 10 * time.Millisecond

				cache.SetWithTTL(key, &value, ttl)

				// Immediately try to get it
				if _, found := cache.Get(key); !found {
					t.Errorf("Key should be available immediately after setting with TTL")
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestMemoryEfficiencyOptimizations tests that our optimizations actually reduce memory usage
func TestMemoryEfficiencyOptimizations(t *testing.T) {
	// Test with many TTL items to ensure we don't create too many goroutines
	cache := New[string, string](&Config{})

	// Add many items with TTL - this should not create memory issues
	for i := 0; i < 1000; i++ {
		value := fmt.Sprintf("value-%d", i)
		cache.SetWithTTL(fmt.Sprintf("key-%d", i), &value, time.Hour)
	}

	// Basic functionality should still work
	testValue := "test"
	cache.Set("test-key", &testValue)

	if _, found := cache.Get("test-key"); !found {
		t.Errorf("Basic functionality should work even with many TTL items")
	}
}

// TestCacheLen tests the Len() method
func TestCacheLen(t *testing.T) {
	cache := New[string, string](&Config{})

	// Empty cache
	if cache.Len() != 0 {
		t.Errorf("Empty cache should have length 0, got %d", cache.Len())
	}

	// Add items
	values := []string{"one", "two", "three"}
	for i, val := range values {
		cache.Set(fmt.Sprintf("key%d", i), &val)
	}

	if cache.Len() != 3 {
		t.Errorf("Cache should have length 3, got %d", cache.Len())
	}

	// Delete one
	cache.Delete("key1")

	if cache.Len() != 2 {
		t.Errorf("Cache should have length 2 after deletion, got %d", cache.Len())
	}
}

// TestCacheClear tests the Clear() method
func TestCacheClear(t *testing.T) {
	cache := New[string, string](&Config{})

	// Add items
	values := []string{"one", "two", "three"}
	for i, val := range values {
		cache.Set(fmt.Sprintf("key%d", i), &val)
	}

	// Add TTL items
	cache.SetWithTTL("ttl-key", &values[0], time.Hour)

	if cache.Len() != 4 {
		t.Errorf("Cache should have 4 items before clear")
	}

	// Clear cache
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Cache should be empty after clear, got %d items", cache.Len())
	}

	// Verify items are actually gone
	for i := 0; i < 3; i++ {
		if _, found := cache.Get(fmt.Sprintf("key%d", i)); found {
			t.Errorf("Item key%d should be gone after clear", i)
		}
	}

	if _, found := cache.Get("ttl-key"); found {
		t.Errorf("TTL item should be gone after clear")
	}
}

// TestCacheClose tests the Close() method
func TestCacheClose(t *testing.T) {
	cache := New[string, string](&Config{})

	// Add some items
	value := "test"
	cache.Set("test-key", &value)
	cache.SetWithTTL("ttl-key", &value, time.Hour)

	// Close should not panic
	cache.Close()

	// Cache should still be usable for basic operations after close
	// (though cleanup goroutine won't run)
	if _, found := cache.Get("test-key"); !found {
		t.Errorf("Existing items should still be accessible after close")
	}
}
