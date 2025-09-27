package scryball

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ninesl/scryball/internal/client"
)

// Decklist represents a Magic: The Gathering deck with maindeck and sideboard.
type Decklist struct {
	Maindeck  map[*MagicCard]int // Card to quantity mapping
	Sideboard map[*MagicCard]int // Card to quantity mapping (max 15 cards total)
}

// // Returns the decklist in text format, able to be exported to Arena or similar platform.
// func (decklist *Decklist) String() string {
// 	var sb strings.Builder
// 	sb.WriteString("Deck\n")

// 	for card, quantity := range decklist.Maindeck {
// 		sb.WriteString(fmt.Sprintf("%d %s", quantity, card.Name))
// 	}
// 	sb.WriteString("\nSideboard\n")
// 	for card, quantity := range decklist.Sideboard {
// 		sb.WriteString(fmt.Sprintf("%d %s", quantity, card.Name))
// 	}

// 	return sb.String()
// }

// shared parsing implementation
func (sb *Scryball) parseDecklist(ctx context.Context, decklistString string) (*Decklist, error) {
	decklist := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}

	lines := strings.Split(decklistString, "\n")
	var inDeck bool // must start with "Deck"
	var inSideboard bool
	var sideboardTotal int

	var hasAbout = false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 {
			if strings.EqualFold(line, "About") {
				hasAbout = true
				continue
			}
		} else if i == 1 {
			if hasAbout {
				parts := strings.Split(line, " ")
				if strings.EqualFold(parts[0], "Name") {
					continue
				} else {
					return nil, fmt.Errorf("must have deck name even if unused with 'About'")
				}
			}
		}

		if !inDeck {
			if line == "" {
				continue
			}
		}

		if strings.EqualFold(line, "Deck") {
			if inDeck {
				return nil, fmt.Errorf("already parsing Deck, did you input a deck twice?")
			} else {
				inDeck = true
			}

			if inSideboard {
				return nil, fmt.Errorf("already submitting sideboard, found on line %d", i)
			}

			continue
		}

		if strings.EqualFold(line, "Sideboard") {
			if inSideboard {
				return nil, fmt.Errorf("cannot have sideboard twice, found on line %d", i)
			}
			inSideboard = true
			continue
		}

		quantity, cardName, err := parseCardLine(line)
		if err != nil {
			return nil, err
		}

		var magicCard *MagicCard

		// First check cache
		magicCard, err = sb.FetchCardByExactName(ctx, cardName)
		if err == sql.ErrNoRows {
			// Not in cache, try API
			// Search for exact match using the instance's client
			cards, searchErr := sb.client.QueryForCards(fmt.Sprintf("!\"%s\"", cardName))
			if searchErr != nil || len(cards) == 0 {
				// Try broader search
				cards, searchErr = sb.client.QueryForCards(cardName)
				if searchErr != nil || len(cards) == 0 {
					return nil, fmt.Errorf("card not found: %s", cardName)
				}
			}

			// Check for exact name match in results
			var exactMatch *client.Card
			for i := range cards {
				if strings.EqualFold(cards[i].Name, cardName) {
					exactMatch = &cards[i]
					break
				}
			}

			var apiCard *client.Card
			if exactMatch != nil {
				apiCard = exactMatch
			} else if len(cards) == 1 {
				// If only one result, use it
				apiCard = &cards[0]
			} else {
				// Multiple cards, ambiguous
				var names []string
				for _, c := range cards {
					names = append(names, c.Name)
				}
				return nil, fmt.Errorf("ambiguous card name '%s', could be: %s",
					cardName, strings.Join(names, ", "))
			}

			// Cache the card (InsertCardFromAPI now fetches ALL printings automatically)
			magicCard, err = sb.InsertCardFromAPI(ctx, apiCard)
			if err != nil {
				return nil, fmt.Errorf("failed to cache card %s: %v", cardName, err)
			}
		} else if err != nil {
			// Database error
			return nil, fmt.Errorf("database error fetching %s: %v", cardName, err)
		}

		// Add to appropriate section
		if inSideboard {
			sideboardTotal += quantity
			if sideboardTotal > 15 {
				return nil, fmt.Errorf("sideboard exceeds 15 cards (has %d)", sideboardTotal)
			}

			if key, exists := doesCardExistInMap(magicCard, decklist.Sideboard); exists {
				decklist.Sideboard[key] += quantity
			} else {
				decklist.Sideboard[key] = quantity
			}
		} else {
			if key, exists := doesCardExistInMap(magicCard, decklist.Maindeck); exists {
				decklist.Maindeck[key] += quantity
			} else {
				decklist.Maindeck[key] = quantity
			}
		}

	}

	return decklist, nil
}

// if it does, it returns the key pointer
func doesCardExistInMap(magicCard *MagicCard, list map[*MagicCard]int) (*MagicCard, bool) {
	for card := range list {
		if strings.Compare(*magicCard.OracleID, *card.OracleID) == 0 {
			return card, true
		}
	}
	return magicCard, false
}

// ParseDecklist parses an pasted string decklist and returns a Decklist.
//
// Format expected:
//
//	4 Lightning Bolt
//	2 Counterspell
//	4 Island
//
//	Sideboard
//	3 Pyroblast
//
// Also supports format with set codes like when exported from Arena
// (does not affect card.Printings, each MagicCard will have all it's printings)
//
//	4 Lightning Bolt (2ED) 161
//	2 Counterspell (ICE) 64
//
// Behavior:
//   - Fetches missing cards with single API call per unique card
//   - Each fetched card includes all printings across all sets
//   - Handles exact name matches
//   - Returns error for ambiguous card names
//   - Sideboard section must be preceded by "Sideboard" header
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
//
// Example:
//
//	deckString := `
//	4 Lightning Bolt
//	4 Counterspell
//	20 Island
//	20 Mountain
//
//	Sideboard
//	3 Pyroblast
//	2 Red Elemental Blast
//	`
//	deck, err := scryball.ParseDecklist(deckString)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Deck has %d cards\n", deck.NumberOfCards()) // 48
//	fmt.Printf("Sideboard has %d cards\n", deck.NumberOfSideboardCards()) // 5
func ParseDecklist(decklist string) (*Decklist, error) {
	ctx := context.Background()
	return ParseDecklistWithContext(ctx, decklist)
}

// ParseDecklistWithContext parses an Arena-format decklist with context support.
//
// Accepts same format as ParseDecklist but supports context cancellation and timeouts.
// Each card name is looked up via the global Scryball instance for caching and validation.
//
// Returns:
//   - *Decklist: Parsed deck with card objects and quantities
//   - error: Context errors, parse errors, or card lookup failures
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func ParseDecklistWithContext(ctx context.Context, decklistString string) (*Decklist, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}
	return sb.parseDecklist(ctx, decklistString)
}

// ParseDecklist parses a decklist using this Scryball instance's client and database.
//
// Format supported: Arena export format (see ParseDecklist for details)
//
// Behavior:
//   - Uses this instance's database for caching
//   - Uses this instance's client for API calls
//   - Fetches missing cards with single API call per unique card
//   - Returns error for ambiguous card names
func (s *Scryball) ParseDecklist(decklistString string) (*Decklist, error) {
	ctx := context.Background()
	return s.ParseDecklistWithContext(ctx, decklistString)
}

// ParseDecklistWithContext parses a decklist using this Scryball instance's client and database with context support.
//
// Behavior:
//   - Uses this instance's database for caching
//   - Uses this instance's client for API calls
//   - Fetches missing cards with single API call per unique card
//   - Returns error for ambiguous card names
//   - Respects context cancellation and timeouts
func (s *Scryball) ParseDecklistWithContext(ctx context.Context, decklistString string) (*Decklist, error) {
	return s.parseDecklist(ctx, decklistString)
}

// parseCardLine extracts quantity and card name from a deck line.
func parseCardLine(line string) (int, string, error) {
	var quantity int
	var cardName string

	// Check if line has parentheses for set code
	parenStart := strings.LastIndex(line, "(")
	parenEnd := strings.LastIndex(line, ")")

	if parenStart != -1 && parenEnd != -1 && parenStart < parenEnd {
		// Format with set code: "4 Thoughtcast (J25) 374"
		beforeParen := strings.TrimSpace(line[:parenStart])

		parts := strings.SplitN(beforeParen, " ", 2)
		if len(parts) < 2 {
			return 0, "", fmt.Errorf("invalid format: %s", line)
		}

		q, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, "", fmt.Errorf("invalid quantity: %s", parts[0])
		}
		quantity = q
		cardName = strings.TrimSpace(parts[1])

	} else {
		// Format without set code: "4 Lightning Bolt"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return 0, "", fmt.Errorf("invalid format: %s", line)
		}

		q, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, "", fmt.Errorf("invalid quantity: %s", parts[0])
		}
		quantity = q
		cardName = strings.TrimSpace(parts[1])
	}

	return quantity, cardName, nil
}

// NumberOfCards returns the total number of cards in the maindeck.
//
// This counts individual cards, so 4 Lightning Bolts = 4 cards.
func (d *Decklist) NumberOfCards() int {
	total := 0
	for _, qty := range d.Maindeck {
		total += qty
	}
	return total
}

// NumberOfSideboardCards returns the total number of cards in the sideboard.
//
// This counts individual cards, so 3 Pyroblasts = 3 cards.
func (d *Decklist) NumberOfSideboardCards() int {
	total := 0
	for _, qty := range d.Sideboard {
		total += qty
	}
	return total
}

// GetMaindeck returns all maindeck cards as a flat list (including duplicates).
//
// Example: If decklist has "4 Lightning Bolt", this returns 4 separate MagicCard instances.
// Useful for statistical analysis or iterating over every card.
func (d *Decklist) GetMaindeck() []*MagicCard {
	var cards []*MagicCard

	for card, qty := range d.Maindeck {
		for range qty {
			cards = append(cards, card)
		}
	}

	return cards
}

// GetSideboard returns all sideboard cards as a flat list (including duplicates).
//
// Example: If sideboard has "3 Pyroblast", this returns 3 separate MagicCard instances.
// Useful for statistical analysis or iterating over every sideboard card.
func (d *Decklist) GetSideboard() []*MagicCard {
	var cards []*MagicCard

	for card, qty := range d.Sideboard {
		for range qty {
			cards = append(cards, card)
		}
	}

	return cards
}

// String returns the decklist in Arena export format.
//
// The output can be passed back to ParseDecklist() to recreate the same deck.
// Format: "4 Lightning Bolt\n3 Mountain\n\nSideboard\n2 Pyroblast"
func (d *Decklist) String() string {
	var sb strings.Builder

	for card, qty := range d.Maindeck {
		sb.WriteString(fmt.Sprintf("%d %s\n", qty, card.Name))
	}

	if len(d.Sideboard) > 0 {
		sb.WriteString("\nSideboard\n")
		for card, qty := range d.Sideboard {
			sb.WriteString(fmt.Sprintf("%d %s\n", qty, card.Name))
		}
	}

	return sb.String()
}

// ValidateDecklist checks if a decklist meets format requirements, returns nil if legal.
//
// Set maxCards to 0 for no maindeck limit.
//
// See d.ValidateConstructed()... etc.
func (d *Decklist) ValidateDecklist(minCards, maxCards, maxSideboard int) error {
	mainTotal := d.NumberOfCards()
	sideTotal := d.NumberOfSideboardCards()

	if mainTotal < minCards {
		return fmt.Errorf("maindeck has %d cards, minimum is %d", mainTotal, minCards)
	}

	if maxCards > 0 && mainTotal > maxCards {
		return fmt.Errorf("maindeck has %d cards, maximum is %d", mainTotal, maxCards)
	}

	if sideTotal > maxSideboard {
		return fmt.Errorf("sideboard has %d cards, maximum is %d", sideTotal, maxSideboard)
	}

	// Count total copies across main and sideboard
	totalCopies := make(map[string]int)
	for card, qty := range d.Maindeck {
		totalCopies[card.Name] += qty
	}
	for card, qty := range d.Sideboard {
		totalCopies[card.Name] += qty
	}

	for cardName, total := range totalCopies {
		if total > 4 && !isBasicLandName(cardName) && !isSpecialCardName(cardName) {
			return fmt.Errorf("total of %d copies of %s between maindeck and sideboard, maximum is 4", total, cardName)
		}
	}

	return nil
}

// ValidateConstructed validates the deck for Constructed formats (60+ cards, 15 card sideboard).
//
// Enforces the 4-copy rule (except basic lands and special cards ie. Relentless Rats)
//
// Minimum 60 cards in maindeck, maximum 15 in sideboard.
func (d *Decklist) ValidateConstructed() error {
	d.ValidateFourOfs()
	return d.ValidateDecklist(60, 0, 15)
}

// ValidateLimited validates the deck for Limited formats like Draft or Sealed (40+ cards).
//
// Minimum 40 cards in maindeck, no maximum in the sideboard.
func (d *Decklist) ValidateLimited() error {
	return d.ValidateDecklist(40, 0, 0)
}

func (d *Decklist) ValidateSingleton() error {
	for card, qty := range d.Maindeck {
		if qty > 1 && !isBasicLand(card) && !isSpecialCard(card) {
			return fmt.Errorf("maindeck has %d copies of %s, maximum is 1", qty, card.Name)
		}
	}
	return nil
}

func (d *Decklist) ValidateFourOfs() error {
	for card, qty := range d.Maindeck {
		if qty > 4 && !isBasicLand(card) && !isSpecialCard(card) {
			return fmt.Errorf("maindeck has %d copies of %s, maximum is 4", qty, card.Name)
		}
	}
	return nil
}

func isBasicLand(card *MagicCard) bool {
	return isBasicLandName(card.Name)
}

func isBasicLandName(name string) bool {
	basicLands := []string{
		"Plains", "Island", "Swamp", "Mountain", "Forest",
		"Snow-Covered Plains", "Snow-Covered Island",
		"Snow-Covered Swamp", "Snow-Covered Mountain", "Snow-Covered Forest",
		"Wastes", "Snow-Covered Wastes",
	}

	return slices.Contains(basicLands, name)
}

func isSpecialCard(card *MagicCard) bool {
	return isSpecialCardName(card.Name)
}

// TODO: a better impl than this.
func isSpecialCardName(name string) bool {
	// Cards that can have any number in deck
	specialCards := []string{
		"Relentless Rats",
		"Shadowborn Apostle",
		"Rat Colony",
		"Persistent Petitioners",
		"Dragon's Approach",
		"Seven Dwarves", // Can have up to 7
		"Nazgûl",        // Can have up to 9
	}

	for _, special := range specialCards {
		if strings.EqualFold(name, special) {
			return true
		}
	}
	return false
}
