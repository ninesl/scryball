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
	// Test: Search for Lightning Bolt specifically
	fmt.Println("=== Testing Query() with Lightning Bolt ===")
	boltCards, err := scryball.Query("name:\"Lightning Bolt\"")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Query returned %d Lightning Bolt cards\n", len(boltCards))
	for i, card := range boltCards {
		fmt.Printf("Card %d: %s - %d printings\n", i+1, card.Name, len(card.Printings))
		if len(card.Printings) <= 1 {
			fmt.Printf("  ❌ WARNING: Only %d printing(s) found!\n", len(card.Printings))
		} else {
			fmt.Printf("  ✅ SUCCESS: Found %d printings\n", len(card.Printings))
		}
	}
	fmt.Println()

	// Search for cards - automatically cached
	cards, err := scryball.Query("color:blue cmc=1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d blue 1-mana cards\n", len(cards))
	// Test first few cards to see if they have multiple printings
	fmt.Println("=== Checking first 3 blue 1-mana cards for printings ===")
	for i := 0; i < 3 && i < len(cards); i++ {
		card := cards[i]
		fmt.Printf("%s: %d printings", card.Name, len(card.Printings))
		if len(card.Printings) <= 1 {
			fmt.Printf(" ❌")
		} else {
			fmt.Printf(" ✅")
		}
		fmt.Println()
	}

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

	// Test printings - Sol Ring should have many printings
	fmt.Printf("Sol Ring has %d printings\n", len(card.Printings))
	if len(card.Printings) > 0 {
		fmt.Printf("First printing: %s (%s)\n", card.Printings[0].SetName, card.Printings[0].SetCode)
		if len(card.Printings) > 1 {
			fmt.Printf("Second printing: %s (%s)\n", card.Printings[1].SetName, card.Printings[1].SetCode)
		}
	}

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

	deckText := `
4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast
`

	deck, _ := scryball.ParseDecklist(deckText)

	fmt.Printf("%d cards\n", deck.NumberOfCards())              // 24
	fmt.Printf("%d sideboard\n", deck.NumberOfSideboardCards()) // 3

	deckText = `Deck
4 Delver of Secrets (MID) 47
3 Mountain (ANA) 32
4 Lightning Bolt (STA) 42
4 Island (ANA) 10
4 Brainstorm (STA) 13
4 Steam Vents (GRN) 257
2 Counterspell (STA) 15
4 Spirebluff Canal (KLR) 286
4 Consider (MID) 44
1 Negate (MOM) 68
4 Mishra's Bauble (BRR) 34
3 Scalding Tarn (MH2) 254
4 Ragavan, Nimble Pilferer (MUL) 86
4 Expressive Iteration (STX) 186
3 Renegade Map (KLR) 265
4 Unholy Heat (MH2) 145
4 Dragon's Rage Channeler (MH2) 121

Sideboard
4 Aether Gust (M20) 42
`

	deck, _ = scryball.ParseDecklist(deckText)

	fmt.Printf("%d cards\n", deck.NumberOfCards())              // 60
	fmt.Printf("%d sideboard\n", deck.NumberOfSideboardCards()) // 4

}
