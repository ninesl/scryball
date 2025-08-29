package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ninesl/scryball"
)

func main() {
	// Search for cards - automatically cached
	cards, err := scryball.Query("color:blue cmc=1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d blue 1-mana cards\n", len(cards))

	// Get a specific card - also cached
	bolt, err := scryball.QueryCard("Lightning Bolt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s costs %s\n", bolt.Name, *bolt.ManaCost)

	_ = scryball.SetConfig(scryball.ScryballConfig{
		DBPath: "./cards.db",
	})
	sb, _ := scryball.NewWithConfig(scryball.ScryballConfig{
		DBPath: "./cards.db",
	})
	now := time.Now()
	cards, _ = scryball.Query("set:neo rarity:mythic")
	fmt.Printf("%v found %d cards, fetched from scryfall API\n", time.Since(now), len(cards))
	now = time.Now()
	cards, _ = sb.Query("set:neo rarity:mythic")
	fmt.Printf("%v found %d cards, found from DB\n", time.Since(now), len(cards))

	card, _ := scryball.QueryCard("Sol Ring")

	// Full Scryfall API data: https://scryfall.com/docs/api/cards
	fmt.Println(card.Name)        // Sol Ring
	fmt.Println(card.TypeLine)    // Artifact
	fmt.Println(*card.ManaCost)   // {1}
	fmt.Println(*card.OracleText) // {T}: Add {C}{C}.
	fmt.Println(*card.OracleID)   // Unique across all printings

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	card, _ = scryball.QueryCardWithContext(ctx, "Black Lotus")
	fmt.Println(card.Name)
	fmt.Println(card.TypeLine)
	fmt.Println(*card.ManaCost)
	fmt.Println(*card.OracleText)
	fmt.Println(*card.OracleID)

	// Safe to call from multiple goroutines
	var wg sync.WaitGroup
	for _, color := range []string{"red", "blue", "green"} {
		wg.Add(1)
		fmt.Println("looking for " + color)
		go func(c string) {
			defer wg.Done()
			cards, _ := scryball.Query("set:neo c:" + c)
			fmt.Printf("%s: %d cards\n", c, len(cards))
		}(color)
	}
	wg.Wait()
}
