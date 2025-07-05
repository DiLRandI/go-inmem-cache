# Examples

This directory contains working examples demonstrating the go-inmem-cache library features.

## Available Examples

### 1. Basic Example (`basic/`)

**Run:** `cd basic && go run main.go`

Comprehensive demonstration of all features:

- Basic cache operations (set, get, delete) using pointer-based API
- TTL functionality with `SetWithTTL(key, &value, duration)`
- FIFO eviction when item count limit is reached
- Size-based FIFO eviction with memory limits
- Using different key/value types (strings, structs, integers)
- Memory usage tracking

### 2. TTL Example (`ttl/`)

**Run:** `cd ttl && go run main.go`

Focused demonstration of TTL (Time-To-Live) functionality:

- Pointer-based `SetWithTTL(key, &value, duration)` API
- Different expiration times (seconds, minutes, hours)
- Real-time expiration demonstration
- Memory usage tracking

## How to Run Examples

Each example is in its own directory with a complete Go module setup:

```bash
# Run the comprehensive basic example
cd examples/basic
go run main.go

# Run the TTL-focused example  
cd examples/ttl
go run main.go
```

## Module Setup

Both examples are already configured with proper `go.mod` files that reference the parent cache library:

- Each has its own Go module (`example` or `ttl-example`)
- Uses `replace` directives to point to the local cache implementation
- Ready to run without additional setup

This setup allows you to test the examples against your local development version of the cache library.
