// Package scryball provides a caching layer for the Scryfall Magic: The Gathering card API.
//
// Basic Usage:
//
//	// Search for cards using Scryfall syntax
//	cards, err := scryball.Query("color:blue cmc=1")
//
//	// Get a specific card by name
//	card, err := scryball.QueryCard("Lightning Bolt")
//
// The package efficiently caches API responses in a local SQLite database with one API call per unique card.
// Each card insertion fetches all printings across all sets. Cache hits return with zero API calls.
//
// Configuration:
//
//	config := scryball.ScryballConfig{
//		DBPath: "/path/to/cache.db",
//	}
//	err := scryball.SetConfig(config)
//
// By default, uses an in-memory database that doesn't persist between runs.
//
// Advanced Usage:
//
//	// Create independent instance
//	sb, err := scryball.WithConfig(config)
//	cards, err := sb.Query("set:neo")
//
// See https://scryfall.com/docs/syntax for query syntax documentation.
package scryball

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ninesl/scryball/internal/client"
	"github.com/ninesl/scryball/internal/scryfall"
)

// InsertCardFromAPI stores a Scryfall API card response in the database.
//
// Behavior:
//   - Converts API response to database format
//   - Upserts card data (overwrites if oracle_id exists)
//   - Upserts printing data (overwrites if printing id exists)
//   - Fetches and returns the stored card as MagicCard
//
// Returns:
//   - *MagicCard: The stored card with all printings loaded
//   - error: Conversion errors, database errors, or fetch errors
//
// Note: This is primarily for internal use. Public callers should use Query functions.
func (s *Scryball) InsertCardFromAPI(ctx context.Context, apiCard *client.Card) (*MagicCard, error) {
	cardParams, printingParams, err := convertAPICardToDBParams(apiCard)
	if err != nil {
		return nil, fmt.Errorf("could not convert API card to DB params: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Insert the card first
	err = s.queries.UpsertCard(ctx, cardParams)
	if err != nil {
		return nil, fmt.Errorf("could not upsert card %s: %v", apiCard.Name, err)
	}

	// Insert the initial printing
	err = s.queries.UpsertPrinting(ctx, printingParams)
	if err != nil {
		return nil, fmt.Errorf("could not upsert printing for %s: %v", apiCard.Name, err)
	}

	// Fetch ALL printings for this card and store them
	if apiCard.OracleID != nil {
		allPrintings, err := s.client.FetchAllPrintings(apiCard)
		if err != nil {
			// Don't fail the entire operation if printing fetch fails
			// Just log and continue with the single printing we have
		} else {
			// Store all printings
			for _, printing := range allPrintings {
				// Skip printings without oracle_id
				if printing.OracleID == nil {
					continue
				}

				// Convert printing to DB params
				_, printingParams, err := convertAPICardToDBParams(&printing)
				if err != nil {
					continue // Skip invalid printings
				}

				// Upsert the printing
				err = s.queries.UpsertPrinting(ctx, printingParams)
				if err != nil {
					continue // Skip failed printings
				}
			}
		}
	}

	// Fetch the newly stored card with ALL printings as a MagicCard
	magicCard, err := s.FetchCardByExactOracleID(ctx, cardParams.OracleID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch newly stored card %s: %v", apiCard.Name, err)
	}

	return magicCard, nil
}

// caches the given oracleIDs to the query
func (sb *Scryball) cacheQuery(ctx context.Context, query string, oracleIDs []string) error {
	oracleIDsJSON, err := json.Marshal(oracleIDs)
	if err != nil {
		return fmt.Errorf("could not marshal oracle IDs: %v", err)
	}

	sb.mu.Lock()
	defer sb.mu.Unlock()
	err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
		QueryText: query,
		OracleIds: string(oracleIDsJSON),
	})
	if err != nil {
		return fmt.Errorf("could not cache query: %v", err)
	}
	return nil
}

// returns the cards every card found. will insert each card it finds (including pages/List see scryfall docs)
func (sb *Scryball) findQuery(ctx context.Context, query string) ([]*MagicCard, error) {
	cachedCards, err := sb.FetchCardsByQuery(ctx, query)
	if err == nil {
		var oracleIDs = make([]string, len(cachedCards))
		for i, card := range cachedCards {
			if card.OracleID != nil {
				oracleIDs[i] = *card.OracleID
			}
		}
		return cachedCards, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}
	// query does not exist, fetch from API
	// Don't add unique:prints - just use the original query
	apiCards, err := sb.client.QueryForCards(query)
	if err != nil {
		return nil, err
	}

	// Group cards by oracle_id - skip cards with null oracle_id
	oracleMap := make(map[string]*client.Card)
	for i := range apiCards {
		card := &apiCards[i]
		if card.OracleID == nil {
			// Skip cards with null oracle_id
			continue
		}
		oracleID := *card.OracleID
		// Keep the first card we see for this oracle_id
		if _, exists := oracleMap[oracleID]; !exists {
			oracleMap[oracleID] = card
		}
	}

	// Process each unique card (by oracle_id) and ensure ALL printings are fetched
	magicCards := make([]*MagicCard, 0, len(oracleMap))
	oracleIDs := make([]string, 0, len(oracleMap))

	for oracleID, sampleCard := range oracleMap {
		// InsertCardFromAPI already fetches and stores ALL printings for the card
		magicCard, err := sb.InsertCardFromAPI(ctx, sampleCard)
		if err != nil {
			return nil, err
		}

		magicCards = append(magicCards, magicCard)
		oracleIDs = append(oracleIDs, oracleID)
	}

	// Cache the query with oracle IDs from API fetch
	if err = sb.cacheQuery(ctx, query, oracleIDs); err != nil {
		fmt.Printf("Warning: could not cache query: %v\n", err)
	}

	return magicCards, nil
}

// look for the card within the database, if not found will fetch from the scryfall API
func (sb *Scryball) findCard(ctx context.Context, cardQuery string) (*MagicCard, error) {

	magicCard, err := sb.FetchCardByExactName(ctx, cardQuery)
	if err == nil {
		return magicCard, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}
	// card does not exist, fetch from API

	apiCard, err := sb.client.QueryForSpecificCard(cardQuery)
	if err != nil {
		return nil, err
	}

	magicCard, err = sb.InsertCardFromAPI(ctx, apiCard)
	if err != nil {
		return nil, err
	}

	return magicCard, err
}

// findCardOracleID looks for a card within the database by Oracle ID, if not found will fetch from the scryfall API
func (sb *Scryball) findCardOracleID(ctx context.Context, oracleID string) (*MagicCard, error) {
	// Try to get card from database first
	dbCard, err := sb.queries.GetCardByOracleID(ctx, oracleID)
	if err == nil {
		// Card found in database, build and return it
		return sb.buildMagicCardFromDB(ctx, dbCard.OracleID, dbCard.Name, dbCard.Layout, dbCard.Cmc,
			dbCard.ColorIdentity, dbCard.Colors, dbCard.ManaCost, dbCard.OracleText,
			dbCard.TypeLine, dbCard.Power, dbCard.Toughness)
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error searching for oracle_id %s: %v", oracleID, err)
	}
	// card does not exist, fetch from API

	apiCard, err := sb.client.QueryForSpecificCardByOracleID(oracleID)
	if err != nil {
		return nil, err
	}

	magicCard, err := sb.InsertCardFromAPI(ctx, apiCard)
	if err != nil {
		return nil, err
	}

	return magicCard, err
}

// Query searches for Magic cards using Scryfall query syntax.
//
// Behavior:
//   - Cache hits return complete results with zero API calls
//   - Cache misses make single API call per unique card
//   - Each card fetched includes all printings across all sets
//   - All results cached to prevent repeated API calls
//
// Returns:
//   - []*MagicCard: Array of cards matching the query (empty array if no matches)
//   - error: Network errors, API errors, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
// Query syntax: https://scryfall.com/docs/syntax
func Query(query string) ([]*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}
	ctx := context.Background()
	return sb.findQuery(ctx, query)
}

// QueryWithContext searches for Magic cards using Scryfall query syntax with context support.
//
// Behavior:
//   - Cache hits return complete results with zero API calls
//   - Cache misses make single API call per unique card
//   - Each card fetched includes all printings across all sets
//   - All results cached to prevent repeated API calls
//   - Respects context cancellation and timeouts
//
// Returns:
//   - []*MagicCard: Array of cards matching the query (empty array if no matches)
//   - error: Context errors, network errors, API errors, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
// Query syntax: https://scryfall.com/docs/syntax
func QueryWithContext(ctx context.Context, query string) ([]*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}

	return sb.findQuery(ctx, query)
}

// Query searches for Magic cards using Scryfall query syntax.
//
// Behavior:
//   - Cache hits return complete results with zero API calls
//   - Cache misses make single API call per unique card
//   - Each card fetched includes all printings across all sets
//   - All results cached to prevent repeated API calls
//
// Returns:
//   - []*MagicCard: Array of cards matching the query (empty array if no matches)
//   - error: Network errors, API errors, or database errors
//
// Query syntax: https://scryfall.com/docs/syntax
func (sb *Scryball) Query(query string) ([]*MagicCard, error) {
	ctx := context.Background()
	return sb.findQuery(ctx, query)
}

// QueryWithContext searches for Magic cards using Scryfall query syntax with context support.
//
// Behavior:
//   - Cache hits return complete results with zero API calls
//   - Cache misses make single API call per unique card
//   - Each card fetched includes all printings across all sets
//   - All results cached to prevent repeated API calls
//   - Respects context cancellation and timeouts
//
// Returns:
//   - []*MagicCard: Array of cards matching the query (empty array if no matches)
//   - error: Context errors, network errors, API errors, or database errors
//
// Query syntax: https://scryfall.com/docs/syntax
func (sb *Scryball) QueryWithContext(ctx context.Context, query string) ([]*MagicCard, error) {
	return sb.findQuery(ctx, query)
}

// QueryCard fetches a single Magic card by exact name match.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Name matching is case-insensitive but otherwise exact
//
// Returns:
//   - *MagicCard: The card with exact name match
//   - error: Returns error if card not found, network issues, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func QueryCard(cardQuery string) (*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}

	ctx := context.Background()
	return sb.findCard(ctx, cardQuery)
}

// QueryCardWithContext fetches a single Magic card by exact name match with context support.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Name matching is case-insensitive but otherwise exact
//   - Respects context cancellation and timeouts
//
// Returns:
//   - *MagicCard: The card with exact name match
//   - error: Returns error if card not found, context cancelled, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func QueryCardWithContext(ctx context.Context, cardQuery string) (*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}
	return sb.findCard(ctx, cardQuery)
}

// QueryCard fetches a single Magic card by exact name match.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Name matching is case-insensitive but otherwise exact
//
// Returns:
//   - *MagicCard: The card with exact name match
//   - error: Returns error if card not found, network issues, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func (sb *Scryball) QueryCard(cardQuery string) (*MagicCard, error) {
	ctx := context.Background()
	return sb.findCard(ctx, cardQuery)
}

// QueryCardWithContext fetches a single Magic card by exact name match with context support.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Name matching is case-insensitive but otherwise exact (see scryfall docs)
//   - Respects context cancellation and timeouts
//
// Returns:
//   - *MagicCard: The card with exact name match
//   - error: Returns error if card not found, context cancelled, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func (sb *Scryball) QueryCardWithContext(ctx context.Context, cardQuery string) (*MagicCard, error) {
	return sb.findCard(ctx, cardQuery)
}

// QueryCardByOracleID fetches a single Magic card by exact Oracle ID match.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Oracle ID matching is case-insensitive and exact
//
// Returns:
//   - *MagicCard: The card with exact Oracle ID match
//   - error: Returns error if card not found, network issues, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func QueryCardByOracleID(oracleID string) (*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}

	ctx := context.Background()
	return sb.findCardOracleID(ctx, oracleID)
}

// QueryCardByOracleIDWithContext fetches a single Magic card by exact Oracle ID match with context support.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Oracle ID matching is case-insensitive and exact
//   - Respects context cancellation and timeouts
//
// Returns:
//   - *MagicCard: The card with exact Oracle ID match
//   - error: Returns error if card not found, context cancelled, or database errors
//
// Note: Uses global Scryball instance. Initialize with SetConfig() or defaults to in-memory DB.
func QueryCardByOracleIDWithContext(ctx context.Context, oracleID string) (*MagicCard, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scryball %v", err)
	}
	return sb.findCardOracleID(ctx, oracleID)
}

// QueryCardByOracleID fetches a single Magic card by exact Oracle ID match.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Oracle ID matching is case-insensitive and exact
//
// Returns:
//   - *MagicCard: The card with exact Oracle ID match
//   - error: Returns error if card not found, network issues, or database errors
func (sb *Scryball) QueryCardByOracleID(oracleID string) (*MagicCard, error) {
	ctx := context.Background()
	return sb.findCardOracleID(ctx, oracleID)
}

// QueryCardByOracleIDWithContext fetches a single Magic card by exact Oracle ID match with context support.
//
// Behavior:
//   - Cache hits return card with all printings and zero API calls
//   - Cache misses make single API call that fetches all printings
//   - All card data cached for future requests
//   - Oracle ID matching is case-insensitive and exact
//   - Respects context cancellation and timeouts
//
// Returns:
//   - *MagicCard: The card with exact Oracle ID match
//   - error: Returns error if card not found, context cancelled, or database errors
func (sb *Scryball) QueryCardByOracleIDWithContext(ctx context.Context, oracleID string) (*MagicCard, error) {
	return sb.findCardOracleID(ctx, oracleID)
}
