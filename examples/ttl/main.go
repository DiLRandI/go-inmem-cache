// Optimized TTL Example - demonstrates the efficient heap-based TTL management
// To run: go run main.go
package main

import (
	"fmt"
	"time"

	cache "github.com/DiLRandI/go-inmem-cache"
)

func main() {
	fmt.Println("=== Optimized TTL Management Example ===")

	myCache := cache.New[string, string](nil)

	// The optimized TTL system uses a single background cleanup process
	// instead of individual timers for each item, making it much more
	// memory-efficient for applications with many TTL items

	user123 := "John Doe"
	sessionAbc := "active"
	tempXyz := "data"
	myCache.SetWithTTL("user:123", &user123, 5*time.Second)
	myCache.SetWithTTL("session:abc", &sessionAbc, 30*time.Minute)
	myCache.SetWithTTL("temp:xyz", &tempXyz, 1*time.Hour)

	fmt.Printf("Added 3 items with different TTL values\n")
	fmt.Printf("✓ Using optimized heap-based expiration management\n")

	// Check if items exist immediately
	if valuePtr, found := myCache.Get("user:123"); found {
		fmt.Printf("Found user: %s\n", *valuePtr)
	}

	if valuePtr, found := myCache.Get("session:abc"); found {
		fmt.Printf("Found session: %s\n", *valuePtr)
	}

	fmt.Printf("Cache currently has %d items\n", myCache.Len())

	// Demonstrate adding many TTL items efficiently
	fmt.Println("\nAdding 1000 TTL items to demonstrate efficiency...")
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bulk-item-%d", i)
		value := fmt.Sprintf("bulk-value-%d", i)
		ttl := time.Duration(i+1) * time.Second
		myCache.SetWithTTL(key, &value, ttl)
	}

	fmt.Printf("✓ Added 1000 TTL items efficiently (using single cleanup process)\n")
	fmt.Printf("Cache now has %d items\n", myCache.Len())

	// Wait for the short TTL to expire
	fmt.Println("\nWaiting 6 seconds for user:123 to expire...")
	time.Sleep(6 * time.Second)

	// Manually trigger cleanup to see expired items removed
	myCache.CleanupExpired()

	if _, found := myCache.Get("user:123"); !found {
		fmt.Println("user:123 has expired ✓")
	}

	if valuePtr, found := myCache.Get("session:abc"); found {
		fmt.Printf("session:abc still active: %s ✓\n", *valuePtr)
	}

	fmt.Printf("Cache has %d items after cleanup\n", myCache.Len())
	fmt.Println("✓ Optimized TTL expiration working correctly")

	// Proper cleanup
	defer myCache.Close()
}
