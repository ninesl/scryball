package scryball

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ninesl/scryball/internal/client"
)

// MagicCard represents a Magic: The Gathering card with all its printings.
//
// Fields:
//   - Embeds client.Card containing all card data from Scryfall
//   - printings: All different set printings of this card
//
// Access card fields directly (e.g., card.Name, card.ManaCost).
// Oracle ID uniquely identifies the card across all printings.
type MagicCard struct {
	*client.Card
	Printings []Printing
}

// Printing represents a single printing of a card in a specific set.
// Each MagicCard may have multiple printings across different sets.
type Printing struct {
	SetCode     string   `json:"set_code"`
	SetName     string   `json:"set_name"`
	Rarity      string   `json:"rarity"`
	ImageURI    string   `json:"image_uri"`
	ScryfallURI string   `json:"scryfall_uri"`
	Games       []string `json:"games"`
	ReleasedAt  string   `json:"released_at"`
}

// FetchCardsByQuery retrieves cards from a previously cached query.
//
// Behavior:
//   - Only checks database cache, never queries API
//   - Returns empty slice if query is cached but had no results
//   - Returns sql.ErrNoRows if query has never been cached
//   - Fetches full card details including all printings
//
// Returns:
//   - []*MagicCard: Cached cards for this query (may be empty)
//   - error: sql.ErrNoRows if query not cached, or database errors
//
// Note: Use Query() or QueryWithContext() to automatically handle cache misses.
func (s *Scryball) FetchCardsByQuery(ctx context.Context, query string) ([]*MagicCard, error) {
	queryCache, err := s.queries.GetCachedQuery(ctx, query)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached query: %v", err)
	}

	var oracleIDs []string
	if err := json.Unmarshal([]byte(queryCache.OracleIds), &oracleIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal oracle IDs: %v", err)
	}

	var result = []*MagicCard{}
	for _, oracleID := range oracleIDs {
		magicCard, err := s.FetchCardByExactOracleID(ctx, oracleID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch card by oracle ID %s: %v", oracleID, err)
		}
		result = append(result, magicCard)
	}

	return result, nil
}

// FetchCardsByExactNames retrieves multiple cards by exact name from the database.
//
// Behavior:
//   - Only checks database cache, never queries API
//   - Requires ALL names to exist in cache
//   - Stops and returns error on first missing card
//   - Names must match exactly (case-sensitive)
//
// Returns:
//   - []*MagicCard: Array of cards in same order as input names
//   - error: sql.ErrNoRows if any card not cached, or database errors
//
// Note: Use Query() with name queries for automatic API fallback.
func (s *Scryball) FetchCardsByExactNames(ctx context.Context, names []string) ([]*MagicCard, error) {
	var (
		cards = make([]*MagicCard, len(names))
		err   error
	)
	for i, name := range names {
		cards[i], err = s.FetchCardByExactName(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	return cards, nil
}

// FetchCardByExactName retrieves a single card by exact name from the database.
//
// Behavior:
//   - Only checks database cache, never queries API
//   - Name must match exactly (case-sensitive)
//   - Returns the card with all printings
//
// Returns:
//   - *MagicCard: The card if found in cache
//   - error: sql.ErrNoRows if not cached, or database errors
//
// Note: Use QueryCard() for automatic API fallback with case-insensitive matching.
func (s *Scryball) FetchCardByExactName(ctx context.Context, name string) (*MagicCard, error) {
	dbCard, err := s.queries.GetCardByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("database error searching for name %s: %v", name, err)
	}

	return s.buildMagicCardFromDB(ctx, dbCard.OracleID, dbCard.Name, dbCard.Layout, dbCard.Cmc,
		dbCard.ColorIdentity, dbCard.Colors, dbCard.ManaCost, dbCard.OracleText,
		dbCard.TypeLine, dbCard.Power, dbCard.Toughness)
}

// FetchCardByExactOracleID retrieves a card by its Oracle ID from the database.
//
// Behavior:
//   - Only checks database cache, never queries API
//   - Oracle ID must match exactly
//   - Returns error (not sql.ErrNoRows) if card not found
//   - Loads all printings for the card
//
// Returns:
//   - *MagicCard: The card if found in cache
//   - error: Formatted error if not found, or database errors
//
// Note: This method assumes the card exists and returns a descriptive error if not.
// Used internally after API inserts to guarantee card existence.
func (s *Scryball) FetchCardByExactOracleID(ctx context.Context, oracleID string) (*MagicCard, error) {
	dbCard, err := s.queries.GetCardByOracleID(ctx, oracleID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no card found with oracle_id: %s", oracleID)
	}
	if err != nil {
		return nil, fmt.Errorf("database error searching for oracle_id %s: %v", oracleID, err)
	}

	return s.buildMagicCardFromDB(ctx, dbCard.OracleID, dbCard.Name, dbCard.Layout, dbCard.Cmc,
		dbCard.ColorIdentity, dbCard.Colors, dbCard.ManaCost, dbCard.OracleText,
		dbCard.TypeLine, dbCard.Power, dbCard.Toughness)
}

// FetchCardsByExactOracleIDs retrieves multiple cards by Oracle IDs from the database.
//
// Behavior:
//   - Only checks database cache, never queries API
//   - ALL Oracle IDs must exist in cache
//   - Stops and returns error on first missing card
//   - Returns descriptive error (not sql.ErrNoRows) if any card not found
//
// Returns:
//   - []*MagicCard: Array of cards in same order as input Oracle IDs
//   - error: Formatted error if any card not found, or database errors
//
// Note: This assumes all cards exist. Used internally after batch API inserts.
func (s *Scryball) FetchCardsByExactOracleIDs(ctx context.Context, oracleIDs []string) ([]*MagicCard, error) {
	var (
		cards = make([]*MagicCard, len(oracleIDs))
		err   error
	)
	for i, oracleID := range oracleIDs {
		cards[i], err = s.FetchCardByExactOracleID(ctx, oracleID)
		if err != nil {
			return nil, err
		}
	}
	return cards, nil
}

func (s *Scryball) buildMagicCardFromDB(ctx context.Context, oracleID, name, layout string, cmc float64,
	colorIdentity string, colors sql.NullString, manaCost, oracleText sql.NullString,
	typeLine string, power, toughness sql.NullString) (*MagicCard, error) {

	card := &client.Card{
		Object:   "card",
		Name:     name,
		CMC:      cmc,
		TypeLine: typeLine,
		Layout:   layout,
	}

	if oracleID != "" {
		card.OracleID = &oracleID
	}

	if manaCost.Valid {
		card.ManaCost = &manaCost.String
	}
	if oracleText.Valid {
		card.OracleText = &oracleText.String
	}
	if power.Valid {
		card.Power = &power.String
	}
	if toughness.Valid {
		card.Toughness = &toughness.String
	}

	if colorIdentity != "" {
		var ci []string
		if err := json.Unmarshal([]byte(colorIdentity), &ci); err == nil {
			card.ColorIdentity = ci
		}
	}
	if colors.Valid && colors.String != "" {
		var c []string
		if err := json.Unmarshal([]byte(colors.String), &c); err == nil {
			card.Colors = c
		}
	}

	printings, err := s.getPrintingsFromDB(ctx, oracleID)
	if err != nil {
		return nil, fmt.Errorf("error fetching printings for oracle_id %s: %v", oracleID, err)
	}

	return &MagicCard{
		Card:      card,
		printings: printings,
	}, nil
}

func (s *Scryball) getPrintingsFromDB(ctx context.Context, oracleID string) ([]Printing, error) {
	dbPrintings, err := s.queries.GetPrintingsByOracleID(ctx, oracleID)
	if err != nil {
		return nil, err
	}

	printings := make([]Printing, 0, len(dbPrintings))
	for _, dbPrinting := range dbPrintings {
		printing := Printing{
			SetCode:     dbPrinting.SetCode,
			SetName:     dbPrinting.SetName,
			Rarity:      dbPrinting.Rarity,
			ScryfallURI: dbPrinting.ScryfallUri,
			ReleasedAt:  dbPrinting.ReleasedAt,
		}

		// Parse games JSON field
		if dbPrinting.Games != "" {
			var games []string
			if err := json.Unmarshal([]byte(dbPrinting.Games), &games); err == nil {
				printing.Games = games
			}
		}

		// Parse image URIs JSON field
		if dbPrinting.ImageUris.Valid && dbPrinting.ImageUris.String != "" {
			var imageUris map[string]string
			if err := json.Unmarshal([]byte(dbPrinting.ImageUris.String), &imageUris); err == nil {
				// Use normal image URI if available, fallback to small or large
				if uri, ok := imageUris["normal"]; ok {
					printing.ImageURI = uri
				} else if uri, ok := imageUris["small"]; ok {
					printing.ImageURI = uri
				} else if uri, ok := imageUris["large"]; ok {
					printing.ImageURI = uri
				}
			}
		}

		printings = append(printings, printing)
	}

	return printings, nil
}
