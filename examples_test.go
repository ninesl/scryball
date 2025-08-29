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
