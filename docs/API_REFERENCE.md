# Scryball API Reference

Complete reference for all public functions, types, and methods in the Scryball library.

## Package Functions

These functions operate on a global Scryball instance that's automatically initialized (or whatever's configured with `scryball.SetConfig()`).

### Query Functions

#### `Query(query string) ([]*MagicCard, error)`

Searches for Magic cards using Scryfall query syntax.

**Parameters:**
- `query`: Scryfall search query string

**Returns:**
- `[]*MagicCard`: Array of matching cards (empty if no matches)
- `error`: Network, API, or database errors

**Behavior:**
- Cache hits return results with zero API calls
- Cache misses make single API call per unique card 
- Each card insertion fetches all printings across all sets
- All results cached for future queries

**Example:**
```go
cards, err := scryball.Query("color:blue cmc=1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d blue 1-mana cards\n", len(cards))
```

---

#### `QueryWithContext(ctx context.Context, query string) ([]*MagicCard, error)`

Same as `Query()` but supports context cancellation and timeouts.

**Parameters:**
- `ctx`: Context for cancellation/timeout
- `query`: Scryfall search query string

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
cards, err := scryball.QueryWithContext(ctx, "set:neo")
```

---

#### `QueryCard(cardQuery string) (*MagicCard, error)`

Fetches a single Magic card by exact name match.

**Parameters:**
- `cardQuery`: Exact card name (case-insensitive)

**Returns:**
- `*MagicCard`: The matching card with all printings populated
- `error`: Error if card not found or other issues

**Behavior:**
- Cache hits return complete card data with zero API calls
- Cache misses make single API call that fetches all printings
- Consider using `QueryCardByOracleID()` if you have the Oracle ID

**Example:**
```go
card, err := scryball.QueryCard("Lightning Bolt")
if err != nil {
    log.Fatal(err)
}
```

---

#### `QueryCardWithContext(ctx context.Context, cardQuery string) (*MagicCard, error)`

Same as `QueryCard()` but supports context cancellation and timeouts.

---

#### `QueryCardByOracleID(oracleID string) (*MagicCard, error)`

Fetches a single Magic card by Oracle ID.

**Parameters:**
- `oracleID`: Oracle ID string (e.g. "4457ed35-7c10-48c8-9776-456485fdf070")

**Returns:**
- `*MagicCard`: The matching card with all printings populated
- `error`: Error if card not found or other issues

**Behavior:**
- Cache hits return complete card data without API calls
- Cache misses make single API call that fetches all printings
- Oracle ID matching is case-insensitive and exact
- Uses Scryfall's `/cards/search?q=oracleid:` endpoint internally
- All card data cached for future requests

**Example:**
```go
// Direct Oracle ID lookup
card, err := scryball.QueryCardByOracleID("4457ed35-7c10-48c8-9776-456485fdf070")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found: %s\n", card.Name) // "Lightning Bolt"

// Common workflow: name → Oracle ID → cached future lookups
card, _ := scryball.QueryCard("Lightning Bolt")
oracleID := *card.OracleID  // Store this for later
// Later... cached lookup avoids API call
sameCard, _ := scryball.QueryCardByOracleID(oracleID)
```

---

#### `QueryCardByOracleIDWithContext(ctx context.Context, oracleID string) (*MagicCard, error)`

Same as `QueryCardByOracleID()` but supports context cancellation and timeouts.

**Parameters:**
- `ctx`: Context for cancellation/timeout
- `oracleID`: Oracle ID string (e.g. "4457ed35-7c10-48c8-9776-456485fdf070")

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
card, err := scryball.QueryCardByOracleIDWithContext(ctx, "4457ed35-7c10-48c8-9776-456485fdf070")
```

---

### Configuration Functions

#### `SetConfig(config ScryballConfig) error`

Sets global configuration for the package-level query functions.

**Parameters:**
- `config`: Configuration options

**Example:**
```go
err := scryball.SetConfig(scryball.ScryballConfig{
    DBPath: "./cards.db",
    AppUserAgent: "MyApp/1.0",
})
```

---

#### `WithConfig(config ScryballConfig) (*Scryball, error)`

Creates a new independent Scryball instance with custom configuration.
See the ScryballConfig section for more

**Returns:**
- `*Scryball`: New instance with custom config
- `error`: Database or configuration errors

**Example:**
```go
sb, err := scryball.WithConfig(scryball.ScryballConfig{
    DBPath: "./tournament_cards.db",
})
cards, err := sb.Query("format:standard")
```

---

### Decklist Functions

#### `ParseDecklist(decklist string) (*Decklist, error)`

Parses an Arena-format decklist string.

**Parameters:**
- `decklist`: Arena export format string

**Returns:**
- `*Decklist`: Parsed decklist with card objects
- `error`: Parse errors or card lookup failures

**Supported Format:**
```
4 Lightning Bolt
2 Counterspell
20 Mountain

Sideboard  
3 Pyroblast
```

**Example:**
```go
deck, err := scryball.ParseDecklist(decklistText)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deck has %d cards\n", deck.NumberOfCards())
```

---

#### `ParseDecklistWithContext(ctx context.Context, arenaExport string) (*Decklist, error)`

Same as `ParseDecklist()` but supports context cancellation.

---

### Utility Functions

#### `NewSchema(dbPath string) (*ScryballDB, error)`

Creates a new SQLite database with Scryball schema.

**Parameters:**
- `dbPath`: File path for database (empty string for in-memory)

**Returns:**
- `*ScryballDB`: Initialized database wrapper
- `error`: Database creation errors

---

## Types

### ScryballConfig

Configuration options for Scryball instances.

```go
type ScryballConfig struct {
    // Database file path (empty = in-memory)
    DBPath string
    
    // HTTP client for API requests  
    Client *http.Client
    
    // User-Agent header for API calls
    AppUserAgent string
}
```

**Field Details:**

- **`DBPath`**: File path for SQLite database. Empty string creates in-memory database that's lost when program exits. File path creates persistent cache that survives restarts.

- **`Client`**: Custom HTTP client for Scryfall API requests. Useful for proxies, timeouts, or rate limiting. Defaults to `&http.Client{}`.

- **`AppUserAgent`**: User-Agent header sent with API requests. Scryfall appreciates descriptive user agents to identify your app. Defaults to `"MTGScryball/1.0"`.

---

### MagicCard

Represents a Magic: The Gathering card with all its printings.

Based on: [https://scryfall.com/docs/api/cards](https://scryfall.com/docs/api/cards)

```go
type MagicCard struct {
    *client.Card      // Embedded Scryfall card data
    Printings []Printing  // All set printings of this card
}
```

**Usage:**
```go
// Access any Scryfall field directly
fmt.Println(card.Name)           // "Lightning Bolt"  
fmt.Println(card.TypeLine)       // "Instant"
fmt.Println(*card.ManaCost)      // "{R}"
fmt.Println(*card.OracleText)    // Full rules text
fmt.Println(*card.OracleID)      // Unique identifier across printings

// All printings always populated from cache or API
for _, printing := range card.Printings {
    fmt.Printf("%s (%s)\n", printing.SetName, printing.SetCode)
}

// Oracle ID workflow - store Oracle ID for cached subsequent lookups
oracleID := *card.OracleID
// Later... cached lookup avoids API call
sameCard, _ := scryball.QueryCardByOracleID(oracleID)
```

**Key Fields from client.Card:**
- `Name`: Card name (always present)
- `ManaCost`: Mana cost string (pointer - may be nil)
- `CMC`: Converted mana cost (always present)
- `TypeLine`: Full type line (always present)
- `OracleText`: Rules text (pointer - may be nil)
- `Power`, `Toughness`: Creature stats (pointers - may be nil)
- `Loyalty`: Planeswalker starting loyalty (pointer - may be nil)
- `Colors`: Color array (may be nil/empty)
- `ColorIdentity`: Color identity array (always present)
- `Prices`: Price information (map values are pointers - individual prices may be nil)
- `ImageURIs`: Card image URLs (map values are strings when present)

---

### Printing

Represents a single printing of a card in a specific set.

```go
type Printing struct {
    SetCode     string   `json:"set_code"`      // "neo"
    SetName     string   `json:"set_name"`      // "Kamigawa: Neon Dynasty"
    Rarity      string   `json:"rarity"`        // "common", "uncommon", "rare", "mythic"  
    ImageURI    string   `json:"image_uri"`     // High-res card image URL
    ScryfallURI string   `json:"scryfall_uri"`  // Scryfall page URL
    Games       []string `json:"games"`         // ["paper", "arena", "mtgo"]
    ReleasedAt  string   `json:"released_at"`   // "2022-02-18"
}
```

---

### Decklist

Represents a Magic deck with maindeck and sideboard.

```go
type Decklist struct {
    Maindeck  map[*MagicCard]int  // Card to quantity mapping
    Sideboard map[*MagicCard]int  // Sideboard cards to quantity mapping
}
```

---

## Scryball Instance Methods  

Methods available on `*Scryball` instances created with `WithConfig()`.

### Query Methods

#### `(s *Scryball) Query(query string) ([]*MagicCard, error)`

Instance version of package-level `Query()`.

#### `(s *Scryball) QueryWithContext(ctx context.Context, query string) ([]*MagicCard, error)`

Instance version of package-level `QueryWithContext()`.

#### `(s *Scryball) QueryCard(cardQuery string) (*MagicCard, error)`

Instance version of package-level `QueryCard()`.

#### `(s *Scryball) QueryCardWithContext(ctx context.Context, cardQuery string) (*MagicCard, error)`

Instance version of package-level `QueryCardWithContext()`.

#### `(s *Scryball) QueryCardByOracleID(oracleID string) (*MagicCard, error)`

Instance version of package-level `QueryCardByOracleID()`.

#### `(s *Scryball) QueryCardByOracleIDWithContext(ctx context.Context, oracleID string) (*MagicCard, error)`

Instance version of package-level `QueryCardByOracleIDWithContext()`.

---

### Cache-Only Methods

These methods only check the database cache and never make API calls.

#### `(s *Scryball) FetchCardsByQuery(ctx context.Context, query string) ([]*MagicCard, error)`

Retrieves cached results for a query without making API calls.

**Returns:**
- `[]*MagicCard`: Cached cards with full printing data (may be empty array)
- `error`: `sql.ErrNoRows` if query not cached, or database errors

**Example:**
```go
cached, err := sb.FetchCardsByQuery(ctx, "t:creature")
if err == sql.ErrNoRows {
    fmt.Println("Query not cached - need to use Query()")
} else {
    fmt.Printf("Found %d cached creatures\n", len(cached))
}
```

---

#### `(s *Scryball) FetchCardByExactName(ctx context.Context, name string) (*MagicCard, error)`

Retrieves a cached card by exact name.

**Returns:**
- `*MagicCard`: Cached card with all printings populated
- `error`: `sql.ErrNoRows` if card not cached

---

#### `(s *Scryball) FetchCardByExactOracleID(ctx context.Context, oracleID string) (*MagicCard, error)`

Retrieves a cached card by Oracle ID.

**Returns:**
- `*MagicCard`: Cached card with all printings populated
- `error`: `sql.ErrNoRows` if card not cached

---

#### `(s *Scryball) FetchCardsByExactNames(ctx context.Context, names []string) ([]*MagicCard, error)`

Retrieves multiple cached cards by exact names.

**Behavior:**
- Requires ALL names to exist in cache
- Stops and returns error on first missing card  
- Returns cards with full printing data in same order as input names

---

#### `(s *Scryball) FetchCardsByExactOracleIDs(ctx context.Context, oracleIDs []string) ([]*MagicCard, error)`

Retrieves multiple cached cards by Oracle IDs.

---

### Database Management

#### `(s *Scryball) OverwriteDB(freshDB *ScryballDB) *ScryballDB`

Replaces the instance's database with a new one.

**Returns:**
- `*ScryballDB`: The previous database

---

#### `(s *Scryball) RetrieveDB() *ScryballDB`

Returns a reference to the instance's current database.

---

### Decklist Methods

#### `(s *Scryball) ParseDecklist(decklistString string) (*Decklist, error)`

Instance version of package-level `ParseDecklist()`.

#### `(s *Scryball) ParseDecklistWithContext(ctx context.Context, decklistString string) (*Decklist, error)`

Instance version of package-level `ParseDecklistWithContext()`.

---

## Decklist Methods

Methods available on `*Decklist` instances.

### Information Methods

#### `(d *Decklist) NumberOfCards() int`

Returns total number of cards in maindeck.

#### `(d *Decklist) NumberOfSideboardCards() int`

Returns total number of cards in sideboard.

---

### Card Access Methods

#### `(d *Decklist) GetMaindeck() []*MagicCard`

Returns all maindeck cards as flat list (including duplicates).

**Example:**
```go
// If deck has "4 Lightning Bolt", this returns 4 separate MagicCard instances
allCards := deck.GetMaindeck()
fmt.Printf("Deck has %d total cards\n", len(allCards))
```

#### `(d *Decklist) GetSideboard() []*MagicCard`

Returns all sideboard cards as flat list (including duplicates).

---

### Validation Methods

#### `(d *Decklist) ValidateConstructed() error`

Validates deck for Constructed formats (60+ cards, 15 sideboard).

**Rules Enforced:**
- Minimum 60 cards in maindeck  
- Maximum 15 cards in sideboard
- Maximum 4 copies of each card (except basic lands and special cards)

#### `(d *Decklist) ValidateLimited() error`

Validates deck for Limited formats like Draft (40+ cards, 15 sideboard).

#### `(d *Decklist) ValidateSingleton() error` 

Validates deck for Singleton formats (max 1 copy of each card).

#### `(d *Decklist) ValidateFourOfs() error`

Validates 4-copy rule across maindeck and sideboard.

#### `(d *Decklist) ValidateDecklist(minCards, maxCards, maxSideboard int) error`

Custom validation with specific limits.

**Parameters:**
- `minCards`: Minimum maindeck size
- `maxCards`: Maximum maindeck size (0 = no limit)  
- `maxSideboard`: Maximum sideboard size

---

### Export Methods

#### `(d *Decklist) String() string`

Returns decklist in Arena export format.

**Example:**
```go
fmt.Println(deck.String())
// Output:
// 4 Lightning Bolt
// 20 Mountain
//
// Sideboard  
// 3 Pyroblast
```

---

## Query Syntax Reference

Scryball supports the complete [Scryfall search syntax](https://scryfall.com/docs/syntax). Here are common patterns:

### Colors
- `c:blue` - Blue cards
- `c:uw` - Blue OR white cards
- `c=uw` - Exactly blue and white  
- `ci:red` - Red color identity

### Types
- `t:creature` - Creatures
- `t:instant` - Instants
- `t:legendary` - Legendary permanents
- `t:dragon` - Dragon subtype

### Mana Values
- `cmc=1` - Mana value exactly 1
- `cmc<=3` - Mana value 3 or less
- `mana:{R}{R}` - Costs exactly RR

### Power/Toughness  
- `pow=3` - Power exactly 3
- `tou>4` - Toughness greater than 4
- `pow>=tou` - Power greater than or equal to toughness

### Sets and Formats
- `set:neo` - Kamigawa: Neon Dynasty
- `legal:standard` - Standard legal
- `banned:modern` - Banned in Modern

### Text Searches
- `o:flying` - Oracle text contains "flying"
- `o:"draw a card"` - Exact phrase in oracle text
- `name:bolt` - Name contains "bolt"

### Oracle ID Searches
- `oracleid:4457ed35-7c10-48c8-9776-456485fdf070` - Exact Oracle ID match
- `QueryCardByOracleID()` - Direct Oracle ID lookup (recommended for known IDs)

### Complex Queries
- `t:creature c:red cmc<=3 pow>=2` - Red creatures, ≤3 mana, ≥2 power
- `legal:commander rarity:mythic` - Mythic rares legal in Commander
- `is:split c:uw` - Blue/white split cards

### Best Practices

1. **Always check errors** from query functions
2. **Use context timeouts** for network operations
3. **Handle `sql.ErrNoRows`** specifically for cache-only methods
4. **Validate decklists** before processing
5. **Use descriptive User-Agent** strings for API calls
6. **Prefer Oracle ID lookups** when available - cached lookups avoid API calls

---

## Thread Safety

Scryball is fully thread-safe. All public methods can be called concurrently from multiple goroutines without additional synchronization.

**Internal Protection:**
- Database writes protected by `sync.Mutex`
- SQLite provides concurrent read safety

**Safe Concurrent Usage:**
```go
// Multiple goroutines can safely query concurrently
var wg sync.WaitGroup
queries := []string{"c:red", "c:blue", "c:green", "c:white", "c:black"}

for _, query := range queries {
    wg.Add(1)
    go func(q string) {
        defer wg.Done()
        cards, err := scryball.Query(q)
        if err == nil {
            fmt.Printf("%s: %d cards\n", q, len(cards))
        }
    }(query)
}
wg.Wait()
```

#### Note: Would not recommend to abuse this as Scryfall WILL rate limit you. Requests come from the scryball's `Client`