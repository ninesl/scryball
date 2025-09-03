package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ninesl/scryball"
)

func main() {
	scryball.SetConfig(scryball.ScryballConfig{
		DBPath: "demo_cards.db",
	})

	// Test: Search for red instant commons with cmc 1
	fmt.Println("=== Testing Query() with 'c:r t:instant cmc=1 r:common' ===")
	redInstants, err := scryball.Query("c:r t:instant cmc=1 r:common")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Query returned %d red 1-cmc instant commons\n", len(redInstants))

	// Show ALL cards found
	fmt.Println("All cards found:")
	boltFound := false
	for i, card := range redInstants {
		fmt.Printf("  %d. %s: %d printings", i+1, card.Name, len(card.Printings))
		if card.Name == "Lightning Bolt" {
			fmt.Printf(" ✅ LIGHTNING BOLT!")
			boltFound = true
		}
		fmt.Println()
	}

	if !boltFound {
		fmt.Println("❌ Lightning Bolt NOT found in red instant common results!")
	}
	fmt.Println()

	// Get a specific card - also cached
	bolt, err := scryball.QueryCard("Lightning Bolt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s costs %s\n", bolt.Name, *bolt.ManaCost)

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

	// Test new Oracle ID functionality with Black Lotus
	fmt.Println("\n=== Testing QueryCardByOracleID() ===")
	blackLotusOracleID := "5089ec1a-f881-4d55-af14-5d996171203b"

	card, err = scryball.QueryCardByOracleID(blackLotusOracleID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found card by Oracle ID: %s\n", card.Name)
	fmt.Printf("Oracle ID: %s\n", *card.OracleID)
	fmt.Printf("Type: %s\n", card.TypeLine)
	if card.ManaCost != nil {
		fmt.Printf("Mana Cost: %s\n", *card.ManaCost)
	}
	if card.OracleText != nil {
		fmt.Printf("Oracle Text: %s\n", *card.OracleText)
	}
	fmt.Printf("Number of printings: %d\n", len(card.Printings))

	// Test caching - second call should be much faster
	start := time.Now()
	cardByOracle2, _ := scryball.QueryCardByOracleID(blackLotusOracleID)
	duration := time.Since(start)
	fmt.Printf("Second call (cached) took: %v - got %s\n", duration, cardByOracle2.Name)

	deckText := `
4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast
`

	deck, err := scryball.ParseDecklist(deckText)
	if err != nil {
		log.Fatal(err)
	}

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

	deck, err = scryball.ParseDecklist(deckText)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d cards\n", deck.NumberOfCards())              // 60
	fmt.Printf("%d sideboard\n", deck.NumberOfSideboardCards()) // 4

	// Safe to call from multiple goroutines. Be careful of rate-limiting!
	var wg sync.WaitGroup
	for _, color := range []string{"red", "blue", "green"} {
		wg.Add(1)
		fmt.Println("looking for " + color)
		go func(c string) {
			defer wg.Done()
			cards, _ := scryball.Query("set:neo r:rare c:" + c)
			fmt.Printf("%s: %d cards\n", c, len(cards))
		}(color)
	}
	wg.Wait()

}
