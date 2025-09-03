package scryball_test

import (
	"context"
	"fmt"
	"log"

	"github.com/ninesl/scryball"
)

// Example demonstrating default in-memory database
func Example_defaultInMemory() {
	// No configuration needed - uses in-memory database by default
	cards, err := scryball.Query("t:creature cmc=1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d one-mana creatures\n", len(cards))

	// Output varies based on actual API results
}

// Example demonstrating memory-only cache behavior
func Example_memoryCaching() {
	// Default behavior: cache exists only during program execution
	// No DBPath specified = memory-only cache

	// First query fetches from API and caches in memory
	cards1, err := scryball.Query("name:lightning")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("First query found %d cards\n", len(cards1))

	// Second identical query uses the memory cache (fast!)
	cards2, err := scryball.Query("name:lightning")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Second query found %d cards (from cache)\n", len(cards2))

	// When this program exits, the cache is lost
	// Running the program again would fetch from API again

	// Output varies based on API results
}

// Example showing explicit in-memory configuration
func Example_explicitInMemory() {
	// Explicitly configure in-memory database
	err := scryball.SetConfig(scryball.ScryballConfig{
		// DBPath is empty, so uses in-memory database
		AppUserAgent: "MyMTGApp/1.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	cards, err := scryball.Query("t:instant color:red")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d red instants\n", len(cards))

	// Output varies based on actual API results
}

// Example showing persistent cache to disk
func Example_diskCache() {
	// To save cache to disk (survives program restarts)
	err := scryball.SetConfig(scryball.ScryballConfig{
		DBPath: "/tmp/my-mtg-cache.db", // Cache saved to this file
	})
	if err != nil {
		log.Fatal(err)
	}

	// Now queries are cached to disk
	cards, err := scryball.Query("set:neo")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d cards (saved to disk cache)\n", len(cards))

	// Even after program exits and restarts, the cache at
	// /tmp/my-mtg-cache.db will still have these cards

	// Output varies based on API results
}

// Example demonstrating persistent file database
func Example_persistentDatabase() {
	// Configure persistent database that survives restarts
	err := scryball.SetConfig(scryball.ScryballConfig{
		DBPath:       "/tmp/mtg-cache.db", // File path for persistent storage
		AppUserAgent: "MyMTGApp/1.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	// First query hits API and caches result
	cards1, err := scryball.Query("set:neo rarity:mythic")
	if err != nil {
		log.Fatal(err)
	}

	// Second identical query uses cache (even after restart)
	cards2, err := scryball.Query("set:neo rarity:mythic")
	if err != nil {
		log.Fatal(err)
	}

	// Both results are identical
	if len(cards1) == len(cards2) {
		fmt.Println("Cache working correctly")
	}

	// Output:
	// Cache working correctly
}

// Example creating independent instances
func Example_multipleInstances() {
	// Create in-memory instance (default)
	memInstance, err := scryball.NewWithConfig(scryball.ScryballConfig{})
	if err != nil {
		log.Fatal(err)
	}

	// Create file-based instance
	fileInstance, err := scryball.NewWithConfig(scryball.ScryballConfig{
		DBPath: "/tmp/cache1.db",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Each instance has its own cache
	ctx := context.Background()

	// Query with in-memory instance
	memCards, err := memInstance.QueryWithContext(ctx, "t:artifact")
	if err != nil {
		log.Fatal(err)
	}

	// Query with file instance
	fileCards, err := fileInstance.QueryWithContext(ctx, "t:artifact")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Memory instance: %d artifacts\n", len(memCards))
	fmt.Printf("File instance: %d artifacts\n", len(fileCards))

	// Output varies based on actual API results
}
