package main

import (
	"fmt"
	"time"

	cache "github.com/DiLRandI/go-inmem-cache"
)

func main() {
	// Example 1: Basic Usage
	fmt.Println("=== Basic Usage ===")
	basicExample()

	// Example 2: TTL Usage
	fmt.Println("\n=== TTL Usage ===")
	ttlExample()

	// Example 3: LRU Eviction
	fmt.Println("\n=== LRU Eviction ===")
	lruExample()

	// Example 4: Size-Based Eviction
	fmt.Println("\n=== Size-Based Eviction ===")
	sizeExample()

	// Example 5: Different Types
	fmt.Println("\n=== Different Types ===")
	typeExample()
}

func basicExample() {
	// Create a cache with no size limit
	myCache := cache.New[string, string](nil)

	// Set some values
	name := "John Doe"
	city := "New York"
	country := "USA"
	myCache.Set("name", &name)
	myCache.Set("city", &city)
	myCache.Set("country", &country)

	// Get values
	if namePtr, found := myCache.Get("name"); found {
		fmt.Printf("Name: %s\n", *namePtr)
	}

	if cityPtr, found := myCache.Get("city"); found {
		fmt.Printf("City: %s\n", *cityPtr)
	}

	// Delete a value
	myCache.Delete("city")

	// Check if the deleted value is gone
	if _, found := myCache.Get("city"); !found {
		fmt.Println("City was successfully deleted")
	}
}

func ttlExample() {
	myCache := cache.New[string, string](nil)

	// Set value with 2-second TTL
	ttl := 2 * time.Second
	session := "user123"
	myCache.SetWithTTL("session", &session, ttl)

	// Check immediately
	if valuePtr, found := myCache.Get("session"); found {
		fmt.Printf("Session found: %s\n", *valuePtr)
	}

	// Wait for 1 second
	time.Sleep(1 * time.Second)
	if valuePtr, found := myCache.Get("session"); found {
		fmt.Printf("Session still valid: %s\n", *valuePtr)
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)
	if _, found := myCache.Get("session"); !found {
		fmt.Println("Session expired")
	}
}

func lruExample() {
	// Create cache with max 3 items
	maxItems := int64(3)
	config := &cache.Config{MaxItems: &maxItems}
	myCache := cache.New[string, int](config)

	// Add items
	item1, item2, item3 := 1, 2, 3
	myCache.Set("item1", &item1)
	myCache.Set("item2", &item2)
	myCache.Set("item3", &item3)
	fmt.Println("Added 3 items to cache")

	// Add fourth item - should evict oldest (item1)
	item4 := 4
	myCache.Set("item4", &item4)
	fmt.Println("Added 4th item - oldest should be evicted")

	// item1 should be gone
	if _, found := myCache.Get("item1"); !found {
		fmt.Println("item1 was evicted (LRU)")
	}

	// Other items should still exist
	if valuePtr, found := myCache.Get("item2"); found {
		fmt.Printf("item2 still exists: %d\n", *valuePtr)
	}
}

func sizeExample() {
	// Create cache with 100 bytes memory limit
	maxSize := int64(100)
	config := &cache.Config{Size: &maxSize}
	myCache := cache.New[string, string](config)

	// Add items and show memory usage
	small1, small2, small3 := "x", "y", "z"
	myCache.Set("small1", &small1)
	fmt.Println("Added small1")

	myCache.Set("small2", &small2)
	fmt.Println("Added small2")

	myCache.Set("small3", &small3)
	fmt.Println("Added small3")

	// Add a larger item that should trigger size-based eviction
	large := "this is a much longer string that will trigger eviction"
	myCache.Set("large", &large)
	fmt.Println("Added large item - may trigger eviction")

	// Check which items remain
	for _, key := range []string{"small1", "small2", "small3", "large"} {
		if _, found := myCache.Get(key); found {
			fmt.Printf("Key '%s' still exists\n", key)
		} else {
			fmt.Printf("Key '%s' was evicted\n", key)
		}
	}
}

func typeExample() {
	// Example with different types
	type User struct {
		ID   int
		Name string
		Age  int
	}

	// Cache with string keys and User struct values
	userCache := cache.New[string, User](nil)

	user1 := User{ID: 1, Name: "Alice", Age: 30}
	user2 := User{ID: 2, Name: "Bob", Age: 25}

	userCache.Set("user:1", &user1)
	userCache.Set("user:2", &user2)

	if userPtr, found := userCache.Get("user:1"); found {
		fmt.Printf("User found: %+v\n", *userPtr)
	}

	// Cache with int keys and float values
	priceCache := cache.New[int, float64](nil)
	price1, price2 := 29.99, 49.99
	priceCache.Set(12345, &price1)
	priceCache.Set(67890, &price2)

	if pricePtr, found := priceCache.Get(12345); found {
		fmt.Printf("Product 12345 price: $%.2f\n", *pricePtr)
	}
}
