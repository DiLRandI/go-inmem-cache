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
		cache.Set(key, i)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, i)
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
		cache.SetWithTTL(key, i, &ttl)
	}
}

func BenchmarkCacheConcurrentRead(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, i)
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
			cache.Set(key, i)
			i++
		}
	})
}

func BenchmarkCacheDelete(b *testing.B) {
	cache := New[string, int](nil)

	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Delete(key)
	}
}
