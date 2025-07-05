# go-inmem-cache

A **high-performance**, **thread-safe**, **generic** in-memory cache implementation for Go with **optimized TTL management** and **dual eviction strategies**.

## ⚡ Performance Optimizations

This cache includes several **cutting-edge optimizations** for maximum performance:

1. **🔗 Doubly-Linked List LRU**: O(1) insertions, deletions, and LRU updates (vs O(n) with slice-based approaches)
2. **⏰ Heap-Based TTL Management**: Single background cleanup process instead of individual timers per item
3. **🧮 Optimized Size Calculation**: Cached type information eliminates repeated reflection calls
4. **🔍 Reduced Map Lookups**: Single lookups instead of redundant hash computations
5. **💾 Memory Efficiency**: Eliminated duplicate key storage and optimized struct layouts

## Features

- 🚀 **Generic Type Support**: Works with any comparable key type and any value type
- 🔒 **Thread-Safe**: Built with `sync.RWMutex` for concurrent access
- ⏰ **Optimized TTL Support**: Efficient heap-based expiration with `SetWithTTL(key, &value, duration)`
- 📦 **Dual Eviction**: LRU eviction based on item count OR memory size limits
- 💾 **Memory Tracking**: Accurate size calculation with optimized reflection usage
- 🧹 **Automatic Cleanup**: Background cleanup of expired items with manual trigger option
- 📊 **Cache Management**: Get cache size, item count, clear cache, and proper cleanup
- 🎯 **Zero Dependencies**: Uses only Go standard library
- ⚡ **High Performance**: Optimized for speed and memory efficiency

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
    name := "John Doe"
    myCache.Set("user:123", &name)
    
    // Set a value with TTL
    session := "active"
    myCache.SetWithTTL("session:abc", &session, 30*time.Minute)
    
    // Get a value
    if valuePtr, found := myCache.Get("user:123"); found {
        fmt.Printf("Found value: %s\n", *valuePtr)
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
    Set(key K, value *V)
    SetWithTTL(key K, value *V, ttl time.Duration)
    Get(key K) (*V, bool)
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
name := "value1"
myCache.Set("key1", &name)

// Set a value with TTL
session := "value2"
myCache.SetWithTTL("key2", &session, 30*time.Second)
```

#### Get

```go
if valuePtr, found := myCache.Get("key1"); found {
    fmt.Printf("Value: %s\n", *valuePtr)
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
count := myCache.Len()
fmt.Printf("Current item count: %d\n", count)
```

#### Clear All Items

```go
myCache.Clear()
```

#### Cleanup Expired Items (Manual)

```go
myCache.CleanupExpired()
fmt.Println("Expired items cleaned up")
```

#### Proper Cleanup

```go
// Always close the cache when done to stop background cleanup
defer myCache.Close()
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
user := User{ID: 123, Name: "John"}
userCache.Set("user:123", &user)

// Integer keys, string values
intCache := cache.New[int, string](nil)
answer := "answer"
intCache.Set(42, &answer)

// Custom types as keys (must be comparable)
type ProductID string
productCache := cache.New[ProductID, float64](nil)
price := 29.99
productCache.Set(ProductID("prod-123"), &price)
```

### TTL Examples

```go
// Different TTL durations
short := "value"
medium := "value"
long := "value"
myCache.SetWithTTL("short", &short, 5*time.Second)
myCache.SetWithTTL("medium", &medium, 5*time.Minute)
myCache.SetWithTTL("long", &long, 1*time.Hour)

// Items without TTL never expire (unless evicted by FIFO)
permanent := "value"
myCache.Set("permanent", &permanent)
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
            value := id * 10
            
            // Safe concurrent writes
            myCache.Set(key, &value)
            
            // Safe concurrent reads
            if valuePtr, found := myCache.Get(key); found {
                fmt.Printf("Goroutine %d: %d\n", id, *valuePtr)
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
BenchmarkCacheSet-16                  1871722        609.4 ns/op
BenchmarkCacheGet-16                 12779833        117.6 ns/op
BenchmarkCacheSetWithTTL-16           1000000       1463 ns/op
BenchmarkCacheConcurrentRead-16       4000948        318.0 ns/op
BenchmarkCacheConcurrentWrite-16      3619467        397.2 ns/op
BenchmarkCacheDelete-16               4033398        347.6 ns/op
BenchmarkComplexTypeOperations-16     2327635        646.9 ns/op
BenchmarkOptimizedTTL-16              1000000       1132 ns/op
BenchmarkLRUOperations-16             2490841        505.5 ns/op
BenchmarkCacheUpdate-16               4869848        264.2 ns/op
BenchmarkTTLCleanup-16               24032960         53.32 ns/op
BenchmarkMixedOperations-16           2623839        455.0 ns/op
```

Check out the `cache_test.go` file for more detailed examples and usage patterns.
