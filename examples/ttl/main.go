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

	// The pointer-based API - requires taking addresses of values
	user123 := "John Doe"
	sessionAbc := "active"
	tempXyz := "data"
	myCache.SetWithTTL("user:123", &user123, 5*time.Second)
	myCache.SetWithTTL("session:abc", &sessionAbc, 30*time.Minute)
	myCache.SetWithTTL("temp:xyz", &tempXyz, 1*time.Hour)

	fmt.Printf("Added 3 items with different TTL values\n")

	// Check if items exist immediately
	if valuePtr, found := myCache.Get("user:123"); found {
		fmt.Printf("Found user: %s\n", *valuePtr)
	}

	if valuePtr, found := myCache.Get("session:abc"); found {
		fmt.Printf("Found session: %s\n", *valuePtr)
	}

	fmt.Println("All items stored successfully")

	// Wait for the short TTL to expire
	fmt.Println("\nWaiting 6 seconds for user:123 to expire...")
	time.Sleep(6 * time.Second)

	if _, found := myCache.Get("user:123"); !found {
		fmt.Println("user:123 has expired ✓")
	}

	if valuePtr, found := myCache.Get("session:abc"); found {
		fmt.Printf("session:abc still active: %s ✓\n", *valuePtr)
	}

	fmt.Println("TTL expiration working correctly")
}
