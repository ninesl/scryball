package scryball_test

import (
	"fmt"
	"log"

	"github.com/ninesl/scryball"
)

func Example_memoryCache() {
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

func Example_persistentCache() {
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
