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
	myCache.Set("name", "John Doe")
	myCache.Set("city", "New York")
	myCache.Set("country", "USA")

	// Get values
	if name, found := myCache.Get("name"); found {
		fmt.Printf("Name: %s\n", name)
	}

	if city, found := myCache.Get("city"); found {
		fmt.Printf("City: %s\n", city)
	}

	fmt.Printf("Cache size: %d\n", myCache.Len())

	// Delete a value
	myCache.Delete("city")
	fmt.Printf("Cache size after deletion: %d\n", myCache.Len())
}

func ttlExample() {
	myCache := cache.New[string, string](nil)

	// Set value with 2-second TTL
	ttl := 2 * time.Second
	myCache.SetWithTTL("session", "user123", &ttl)

	// Check immediately
	if value, found := myCache.Get("session"); found {
		fmt.Printf("Session found: %s\n", value)
	}

	// Wait for 1 second
	time.Sleep(1 * time.Second)
	if value, found := myCache.Get("session"); found {
		fmt.Printf("Session still valid: %s\n", value)
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
	myCache.Set("item1", 1)
	myCache.Set("item2", 2)
	myCache.Set("item3", 3)
	fmt.Printf("Cache size: %d\n", myCache.Len())

	// Add fourth item - should evict oldest (item1)
	myCache.Set("item4", 4)
	fmt.Printf("Cache size after adding 4th item: %d\n", myCache.Len())

	// item1 should be gone
	if _, found := myCache.Get("item1"); !found {
		fmt.Println("item1 was evicted (LRU)")
	}

	// Other items should still exist
	if value, found := myCache.Get("item2"); found {
		fmt.Printf("item2 still exists: %d\n", value)
	}
}

func sizeExample() {
	// Create cache with 100 bytes memory limit
	maxSize := int64(100)
	config := &cache.Config{Size: &maxSize}
	myCache := cache.New[string, string](config)

	// Add items and show memory usage
	myCache.Set("small1", "x")
	fmt.Printf("After adding small1: %d bytes, %d items\n", myCache.CurrentSize(), myCache.Len())

	myCache.Set("small2", "y")
	fmt.Printf("After adding small2: %d bytes, %d items\n", myCache.CurrentSize(), myCache.Len())

	myCache.Set("small3", "z")
	fmt.Printf("After adding small3: %d bytes, %d items\n", myCache.CurrentSize(), myCache.Len())

	// Add a larger item that should trigger size-based eviction
	myCache.Set("large", "this is a much longer string that will trigger eviction")
	fmt.Printf("After adding large item: %d bytes, %d items\n", myCache.CurrentSize(), myCache.Len())

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

	userCache.Set("user:1", user1)
	userCache.Set("user:2", user2)

	if user, found := userCache.Get("user:1"); found {
		fmt.Printf("User found: %+v\n", user)
	}

	// Cache with int keys and float values
	priceCache := cache.New[int, float64](nil)
	priceCache.Set(12345, 29.99)
	priceCache.Set(67890, 49.99)

	if price, found := priceCache.Get(12345); found {
		fmt.Printf("Product 12345 price: $%.2f\n", price)
	}
}
