# go-inmem-cache

A high-performance, thread-safe, generic in-memory cache implementation for Go with TTL (Time-To-Live) support and dual eviction strategies.

## Features

- 🚀 **Generic Type Support**: Works with any comparable key type and any value type
- 🔒 **Thread-Safe**: Built with `sync.RWMutex` for concurrent access
- ⏰ **Clean TTL API**: Simple `SetWithTTL(key, value, duration)` - no pointers needed!
- 📦 **Dual Eviction**: FIFO eviction based on item count OR memory size limits
- 💾 **Memory Tracking**: Accurate size calculation and `CurrentSize()` method
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
    // Create a cache with both size and item limits
    config := &cache.Config{
        Size:     &[]int64{1024 * 1024}[0], // 1MB memory limit
        MaxItems: &[]int64{100}[0],         // 100 item limit
    }
    
    // Create cache for string keys and string values
    myCache := cache.New[string, string](config)
    
    // Set a value
    myCache.Set("user:123", "John Doe")
    
    // Set a value with TTL (clean API - no pointers!)
    myCache.SetWithTTL("session:abc", "active", 30*time.Minute)
    
    // Get a value
    if value, found := myCache.Get("user:123"); found {
        fmt.Printf("Found value: %s\n", value)
    }
    
    // Check cache stats
    fmt.Printf("Items: %d, Memory: %d bytes\n", 
        myCache.Len(), myCache.CurrentSize())
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
    SetWithTTL(key K, value V, ttl time.Duration)
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
// Create cache with default config (no limits)
myCache := cache.New[string, string](nil)

// Create cache with item limit only
maxItems := int64(1000)
config := &cache.Config{
    MaxItems: &maxItems,
}
myCache := cache.New[string, string](config)

// Create cache with memory size limit only (FIFO eviction)
maxSize := int64(1024 * 1024) // 1MB
config := &cache.Config{
    Size: &maxSize,
}
myCache := cache.New[string, string](config)

// Create cache with both limits
config := &cache.Config{
    Size:     &maxSize,
    MaxItems: &maxItems,
}
myCache := cache.New[string, string](config)
```

### Basic Operations

#### Set

```go
// Set a value without expiration
myCache.Set("key1", "value1")

// Set a value with TTL (clean API - no pointers!)
myCache.SetWithTTL("key2", "value2", 30*time.Second)
```

#### Get

```go
if value, found := myCache.Get("key1"); found {
    fmt.Printf("Value: %s\n", value)
} else {
    fmt.Println("Key not found or expired")
}
```

#### Delete

```go
myCache.Delete("key1")
```

### Cache Management

#### Get Cache Size

```go
size := myCache.Len()
fmt.Printf("Current cache size: %d\n", size)
```

#### Get Memory Usage

```go
memoryUsage := myCache.CurrentSize()
fmt.Printf("Current memory usage: %d bytes\n", memoryUsage)
```

#### Clear All Items

```go
myCache.Clear()
```

#### Cleanup Expired Items

```go
removedCount := myCache.CleanupExpired()
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
// Different TTL durations - clean API without pointers!
myCache.SetWithTTL("short", "value", 5*time.Second)
myCache.SetWithTTL("medium", "value", 5*time.Minute)
myCache.SetWithTTL("long", "value", 1*time.Hour)

// Items without TTL never expire (unless evicted by FIFO)
myCache.Set("permanent", "value")
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

- **Read Operations**: O(1) average case with concurrent read support via `sync.RWMutex`
- **Write Operations**: O(1) average case with exclusive write access
- **Memory Usage**: Accurate size tracking using reflection for all data types
- **FIFO Eviction**: O(n) when removing oldest items due to size/count limits
- **Cleanup**: O(n) when cleaning expired items
- **Thread Safety**: Uses `sync.RWMutex` for optimal concurrent performance

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

The `examples/` directory contains comprehensive demonstrations:

```bash
# Run the comprehensive example (all features)
cd examples/basic
go run main.go

# Run the TTL-focused example  
cd examples/ttl
go run main.go
```

Each example demonstrates:

- Basic cache operations (set, get, delete)
- TTL functionality with the clean API
- FIFO eviction with size and item limits
- Memory usage tracking
- Using different key/value types

### Running Benchmarks

```bash
go test -bench=. -benchmem
```

Example benchmark results:

```text
BenchmarkCacheSet-16                 2452876       505.1 ns/op     448 B/op      2 allocs/op
BenchmarkCacheGet-16                13508820        80.41 ns/op     13 B/op      1 allocs/op
BenchmarkCacheSetWithTTL-16          2265871       535.3 ns/op     491 B/op      3 allocs/op
BenchmarkCacheConcurrentRead-16     28890189        77.49 ns/op     13 B/op      1 allocs/op
BenchmarkCacheConcurrentWrite-16     3420054       343.2 ns/op      57 B/op      1 allocs/op
```

Check out the `cache_test.go` file for more detailed examples and usage patterns.