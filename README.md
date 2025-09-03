# Scryball

Fast, thread-safe Go client for the Scryfall Magic: The Gathering API with efficient SQLite caching.

Completely supports Scryfall's syntax, see: https://scryfall.com/docs/syntax

```bash
go get github.com/ninesl/scryball
```

## Quick Start

If no DBPath is set, scryball defaults to storing the fetched cards in memory.

```go
import "github.com/ninesl/scryball"

// Search for cards
cards, err := scryball.Query("color:red power>=3")

// Get specific card  
card, err := scryball.QueryCard("Lightning Bolt")
```

That's it. No configuration needed.

## Persistent Cache

```go
// Save cache to disk (survives restarts)
scryball.SetConfig(scryball.ScryballConfig{
    DBPath: "./cards.db",
})

cards, _ := scryball.Query("set:neo rarity:mythic")
// Next time this runs, returns from cache with zero API calls
```


```go
// Independent cache per instance
sb, err := scryball.NewWithConfig(scryball.ScryballConfig{
    DBPath: "./tournament.db",
})

cards, err := sb.Query("legal:standard")
```

**Important:** Each scryball instance should use its own database file to avoid race conditions:

```go
// Recommended - separate databases
scryball.SetConfig(scryball.ScryballConfig{DBPath: "./global.db"})
sb1, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./app1.db"})
sb2, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./app2.db"})

// Not Recommended - shared database (race conditions possible)  
sb1, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./shared.db"})
sb2, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./shared.db"})
```


## Card Data

```go
card, _ := scryball.QueryCard("Sol Ring")

// Full Scryfall API data: https://scryfall.com/docs/api/cards  
// Note: Fields with * are pointers because they're nullable in the Scryfall API
// (some cards don't have mana costs, oracle text, power/toughness, etc.)

fmt.Println(card.Name)           // "Sol Ring"
fmt.Println(card.TypeLine)       // "Artifact"
fmt.Println(*card.ManaCost)      // "{1}"
fmt.Println(*card.OracleText)    // "Add {C}{C}."
fmt.Println(*card.OracleID)      // Unique across all printings

// All printings of this card
for _, printing := range card.Printings {
    fmt.Printf("%s (%s) - %s\n", 
        printing.SetName, printing.SetCode, printing.Rarity)
}
// "Revised Edition (3ed) - uncommon"
// "Commander 2014 (c14) - common"
// etc...
```

## Decklist Parsing

```go
deckText := `
4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast
`

deck, err := scryball.ParseDecklist(deckText)

fmt.Printf("%d cards\n", deck.NumberOfCards())        // 24
fmt.Printf("%d sideboard\n", deck.NumberOfSideboardCards()) // 3

// Also supports set symbol format (like when exporting from arena)
// All cards returned include their complete printing history
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

fmt.Printf("%d cards\n", deck.NumberOfCards())        // 60
fmt.Printf("%d sideboard\n", deck.NumberOfSideboardCards()) // 4
```

## Context Support

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cards, err := scryball.QueryWithContext(ctx, "set:one")
card, err := scryball.QueryCardWithContext(ctx, "Black Lotus")
```

## Thread Safe

```go
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
```

---

**Full query syntax:** https://scryfall.com/docs/syntax  
**Complete API reference:** [docs/API_REFERENCE.md](docs/API_REFERENCE.md)