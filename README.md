# Scryball

Thread-safe Go client for the Scryfall Magic: The Gathering API with SQLite caching.

```bash
go get github.com/ninesl/scryball
```

## Why?

- **Cache lookups:** Cards are automatically cached to avoid redundant API calls
- **Simple syntax:** Use Scryfall's search syntax directly: `"color:red power>=3"`
- **Simplified configuration:** Works out of the box, persist to disk optionally

## Quick Start

```go
import "github.com/ninesl/scryball"

cards, err := scryball.Query("color:red power>=3")
card, err := scryball.QueryCard("Lightning Bolt")
card, err := scryball.QueryCardByOracleID("4457ed35-7c10-48c8-9776-456485fdf070")
```

## Persistent Cache

By default, cards are cached in memory. Set a database path to persist across restarts:

```go
scryball.SetConfig(scryball.ScryballConfig{
    DBPath: "./cards.db",
})

cards, _ := scryball.Query("set:neo rarity:mythic")
```

Multiple instances need separate database files:

```go
sb1, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./app1.db"})
sb2, _ := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: "./app2.db"})
```

## Card Data

All Scryfall API fields are available. Nullable fields are pointers:

Based on: [https://scryfall.com/docs/api/cards](https://scryfall.com/docs/api/cards)

```go
card, _ := scryball.QueryCard("Sol Ring")

fmt.Println(card.Name)        // "Sol Ring"
fmt.Println(*card.ManaCost)   // "{1}"
fmt.Println(*card.OracleText) // "Add {C}{C}."

for _, printing := range card.Printings {
    fmt.Printf("%s (%s)\n", printing.SetName, printing.SetCode)
}
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
fmt.Printf("%d cards, %d sideboard\n", deck.NumberOfCards(), deck.NumberOfSideboardCards())
```

## Context Support

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cards, err := scryball.QueryWithContext(ctx, "set:one")
```

## Thread Safety

All operations are thread-safe:

```go
var wg sync.WaitGroup
for _, color := range []string{"red", "blue", "green"} {
    wg.Add(1)
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