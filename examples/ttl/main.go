// Simple TTL Example - demonstrates the clean TTL API
// To run: go run simple_ttl.go
package main

import (
	"fmt"
	"time"

	cache "github.com/DiLRandI/go-inmem-cache"
)

func main() {
	fmt.Println("=== Simple TTL API Example ===")
	
	myCache := cache.New[string, string](nil)

	// The new clean API - no pointers needed!
	myCache.SetWithTTL("user:123", "John Doe", 5*time.Second)
	myCache.SetWithTTL("session:abc", "active", 30*time.Minute)
	myCache.SetWithTTL("temp:xyz", "data", 1*time.Hour)

	fmt.Printf("Added 3 items with different TTL values\n")

	// Check if items exist immediately
	if value, found := myCache.Get("user:123"); found {
		fmt.Printf("Found user: %s\n", value)
	}

	if value, found := myCache.Get("session:abc"); found {
		fmt.Printf("Found session: %s\n", value)
	}

	fmt.Printf("Cache has %d items\n", myCache.Len())
	fmt.Printf("Memory usage: %d bytes\n", myCache.CurrentSize())
	
	// Wait for the short TTL to expire
	fmt.Println("\nWaiting 6 seconds for user:123 to expire...")
	time.Sleep(6 * time.Second)
	
	if _, found := myCache.Get("user:123"); !found {
		fmt.Println("user:123 has expired ✓")
	}
	
	if value, found := myCache.Get("session:abc"); found {
		fmt.Printf("session:abc still active: %s ✓\n", value)
	}
	
	fmt.Printf("Cache now has %d items\n", myCache.Len())
}
