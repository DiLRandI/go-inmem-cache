package goinmemcache

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkCacheSet(b *testing.B) {
	cache := New[string, int](nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := i
		cache.Set(key, &value)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := i
		cache.Set(key, &value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkCacheSetWithTTL(b *testing.B) {
	cache := New[string, int](nil)
	ttl := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := i
		cache.SetWithTTL(key, &value, ttl)
	}
}

func BenchmarkCacheConcurrentRead(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := i
		cache.Set(key, &value)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkCacheConcurrentWrite(b *testing.B) {
	cache := New[string, int](nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			value := i
			cache.Set(key, &value)
			i++
		}
	})
}

func BenchmarkCacheDelete(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := i
		cache.Set(key, &value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Delete(key)
	}
}

// BenchmarkComplexTypeOperations benchmarks our optimized size calculation with complex types
func BenchmarkComplexTypeOperations(b *testing.B) {
	type ComplexValue struct {
		Data map[string][]int
		Meta []string
		Ptr  *string
	}

	cache := New[string, ComplexValue](nil)
	value := ComplexValue{
		Data: map[string][]int{
			"key1": {1, 2, 3, 4, 5},
			"key2": {6, 7, 8, 9, 10},
		},
		Meta: []string{"meta1", "meta2", "meta3"},
		Ptr:  func() *string { s := "pointer_value"; return &s }(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("complex-key-%d", i)
		cache.Set(key, &value)
	}
}

// BenchmarkOptimizedTTL benchmarks the new heap-based TTL management
func BenchmarkOptimizedTTL(b *testing.B) {
	cache := New[string, string](nil)
	value := "test-value"
	ttl := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("ttl-key-%d", i)
		cache.SetWithTTL(key, &value, ttl)
	}
}

// BenchmarkLRUOperations benchmarks our optimized doubly-linked list LRU
func BenchmarkLRUOperations(b *testing.B) {
	maxItems := int64(1000)
	cache := New[string, string](&Config{MaxItems: &maxItems})

	// Pre-populate to trigger evictions
	for i := 0; i < 999; i++ {
		value := fmt.Sprintf("value-%d", i)
		cache.Set(fmt.Sprintf("key-%d", i), &value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("evict-key-%d", i)
		value := fmt.Sprintf("evict-value-%d", i)
		cache.Set(key, &value) // This will trigger eviction
	}
}

// BenchmarkCacheUpdate benchmarks updating existing items (tests LRU move-to-tail)
func BenchmarkCacheUpdate(b *testing.B) {
	cache := New[string, string](nil)

	// Pre-populate
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		cache.Set(keys[i], &value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%1000]
		value := fmt.Sprintf("updated-value-%d", i)
		cache.Set(key, &value) // Update existing item
	}
}

// BenchmarkTTLCleanup benchmarks the cleanup of expired TTL items
func BenchmarkTTLCleanup(b *testing.B) {
	cache := New[string, string](nil)

	// Add many expired items
	for i := 0; i < 1000; i++ {
		value := fmt.Sprintf("value-%d", i)
		cache.SetWithTTL(fmt.Sprintf("key-%d", i), &value, time.Nanosecond) // Expires immediately
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.CleanupExpired()
	}
}

// BenchmarkMixedOperations benchmarks a realistic mix of operations
func BenchmarkMixedOperations(b *testing.B) {
	maxItems := int64(500)
	cache := New[string, string](&Config{MaxItems: &maxItems})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			value := fmt.Sprintf("value-%d", i)

			switch i % 4 {
			case 0: // Set
				cache.Set(key, &value)
			case 1: // Get
				cache.Get(key)
			case 2: // SetWithTTL
				cache.SetWithTTL(key, &value, time.Hour)
			case 3: // Delete
				cache.Delete(key)
			}
			i++
		}
	})
}
