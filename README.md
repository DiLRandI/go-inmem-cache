# go-inmem-cache

A high-performance, thread-safe, generic in-memory cache implementation for Go with TTL (Time-To-Live) support and LRU (Least Recently Used) eviction policy.

## Features

- 🚀 **Generic Type Support**: Works with any comparable key type and any value type
- 🔒 **Thread-Safe**: Built with `sync.RWMutex` for concurrent access
- ⏰ **TTL Support**: Set expiration times for cache entries
- 📦 **LRU Eviction**: Automatically removes oldest items when cache reaches capacity
- 💾 **Size-Based Eviction**: FIFO eviction when memory usage exceeds size limit
- 🧹 **Manual Cleanup**: Remove expired items on-demand
- 📊 **Cache Statistics**: Get cache size, item count, and memory usage
- 🎯 **Zero Dependencies**: Uses only Go standard library

## Installation

```bash
go get github.com/DiLRandI/go-inmem-cache
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    cache "github.com/DiLRandI/go-inmem-cache"
)

func main() {
    // Create a new cache with maximum 100 items and 1MB memory limit
    maxItems := int64(100)
    maxSize := int64(1024 * 1024) // 1MB
    config := &cache.Config{
        MaxItems: &maxItems,
        Size:     &maxSize,
    }
    
    // Create cache for string keys and int values
    myCache := cache.New[string, int](config)
    
    // Set a value
    myCache.Set("user:123", 42)
    
    // Get a value
    if value, found := myCache.Get("user:123"); found {
        fmt.Printf("Found value: %d\n", value)
    }
    
    // Set a value with TTL (expires in 5 seconds)
    ttl := 5 * time.Second
    myCache.SetWithTTL("session:abc", 999, &ttl)
    
    // Delete a value
    myCache.Delete("user:123")
    
    // Get cache size
    fmt.Printf("Cache size: %d\n", myCache.Len())
}
```

## API Reference

### Types

```go
type Config struct {
    Size     *int64  // Maximum memory usage in bytes (triggers FIFO eviction)
    MaxItems *int64  // Maximum number of items in cache (triggers FIFO eviction)
}

type Cache[K comparable, V any] interface {
    Set(key K, value V)
    SetWithTTL(key K, value V, ttl *time.Duration)
    Get(key K) (V, bool)
    Delete(key K)
    Len() int
    CurrentSize() int64
    Clear()
    CleanupExpired() int
}
```

### Creating a Cache

```go
// Create cache with default config (no size limit)
cache := cache.New[string, int](nil)

// Create cache with item limit
maxItems := int64(1000)
config := &cache.Config{
    MaxItems: &maxItems,
}
cache := cache.New[string, int](config)

// Create cache with memory size limit (FIFO eviction)
maxSize := int64(1024 * 1024) // 1MB
config := &cache.Config{
    Size: &maxSize,
}
cache := cache.New[string, int](config)

// Create cache with both limits
config := &cache.Config{
    Size:     &maxSize,
    MaxItems: &maxItems,
}
cache := cache.New[string, int](config)
```

### Basic Operations

#### Set
```go
// Set a value without expiration
cache.Set("key1", "value1")

// Set a value with TTL
ttl := 30 * time.Second
cache.SetWithTTL("key2", "value2", &ttl)
```

#### Get
```go
if value, found := cache.Get("key1"); found {
    fmt.Printf("Value: %s\n", value)
} else {
    fmt.Println("Key not found or expired")
}
```

#### Delete
```go
cache.Delete("key1")
```

### Cache Management

#### Get Cache Size
```go
size := cache.Len()
fmt.Printf("Current cache size: %d\n", size)
```

#### Get Memory Usage
```go
memoryUsage := cache.CurrentSize()
fmt.Printf("Current memory usage: %d bytes\n", memoryUsage)
```

#### Clear All Items
```go
cache.Clear()
```

#### Cleanup Expired Items
```go
removedCount := cache.CleanupExpired()
fmt.Printf("Removed %d expired items\n", removedCount)
```

## Advanced Usage

### Different Key/Value Types

```go
// String keys, struct values
type User struct {
    ID   int
    Name string
}

userCache := cache.New[string, User](nil)
userCache.Set("user:123", User{ID: 123, Name: "John"})

// Integer keys, string values
intCache := cache.New[int, string](nil)
intCache.Set(42, "answer")

// Custom types as keys (must be comparable)
type ProductID string
productCache := cache.New[ProductID, float64](nil)
productCache.Set(ProductID("prod-123"), 29.99)
```

### TTL Examples

```go
// Different TTL durations
cache.SetWithTTL("short", "value", &[]time.Duration{5 * time.Second}[0])
cache.SetWithTTL("medium", "value", &[]time.Duration{5 * time.Minute}[0])
cache.SetWithTTL("long", "value", &[]time.Duration{1 * time.Hour}[0])

// Items without TTL never expire (unless evicted by LRU)
cache.Set("permanent", "value")
```

### Concurrent Usage

```go
package main

import (
    "sync"
    "fmt"
    
    cache "github.com/DiLRandI/go-inmem-cache"
)

func main() {
    myCache := cache.New[string, int](nil)
    var wg sync.WaitGroup
    
    // Multiple goroutines can safely access the cache
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            key := fmt.Sprintf("key-%d", id)
            
            // Safe concurrent writes
            myCache.Set(key, id*10)
            
            // Safe concurrent reads
            if value, found := myCache.Get(key); found {
                fmt.Printf("Goroutine %d: %d\n", id, value)
            }
        }(i)
    }
    
    wg.Wait()
}
```

## Performance Characteristics

- **Read Operations**: O(1) average case with concurrent read support
- **Write Operations**: O(1) average case with exclusive write access
- **Memory Usage**: Stores both slice and map for efficient access and ordering
- **Item-Based Eviction**: O(n) when updating indices after LRU eviction
- **Size-Based Eviction**: O(n) when evicting multiple items to fit memory limit
- **Cleanup**: O(n) when cleaning expired items
- **Size Calculation**: Uses reflection for accurate memory estimation

## Thread Safety

The cache is fully thread-safe and uses `sync.RWMutex` for optimal performance:

- **Multiple readers**: Can access the cache simultaneously
- **Single writer**: Exclusive access during write operations
- **No data races**: Passes Go's race detector tests

## Use Cases

- **Session Storage**: Store user sessions with automatic expiration
- **API Response Caching**: Cache API responses with TTL
- **Rate Limiting**: Track request counts with time-based expiration
- **Configuration Caching**: Cache configuration data with periodic refresh
- **Temporary Data Storage**: Store computed results with automatic cleanup

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Examples

### Running the Examples

```bash
cd examples
go run main.go
```

The example file demonstrates:
- Basic cache operations (set, get, delete)
- TTL functionality with expiration
- LRU eviction when cache reaches capacity
- Using different key/value types

### Running Benchmarks

```bash
go test -bench=. -benchmem
```

Example benchmark results:
```
BenchmarkCacheSet-16                	 2432089	       501.6 ns/op	     431 B/op	       2 allocs/op
BenchmarkCacheGet-16                	14751292	        84.10 ns/op	      13 B/op	       1 allocs/op
BenchmarkCacheSetWithTTL-16         	 2541315	       500.1 ns/op	     414 B/op	       2 allocs/op
BenchmarkCacheConcurrentRead-16     	19445493	        70.28 ns/op	      13 B/op	       1 allocs/op
BenchmarkCacheConcurrentWrite-16    	 3446289	       367.5 ns/op	      55 B/op	       1 allocs/op
```

Check out the `cache_test.go` file for more detailed examples and usage patterns.