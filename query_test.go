package scryball

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// setupTestScryball creates a temporary database for testing
func setupTestScryball(t *testing.T) *Scryball {
	t.Helper()

	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a new Scryball instance with test database
	config := ScryballConfig{
		DBPath: dbPath,
	}
	sb, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create test Scryball: %v", err)
	}

	return sb
}

func TestQuery_EmptyDatabase(t *testing.T) {
	// This test verifies that Query() works correctly with an empty database
	// and properly fetches from the API when no cached results exist

	sb := setupTestScryball(t)
	defer sb.db.Close()

	// Override the global scryball instance for this test
	CurrentScryball = sb

	query := "Lightning Bolt"

	// First call should hit the API since DB is empty
	cards, err := Query(query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(cards) == 0 {
		t.Fatal("Expected cards to be returned, got empty slice")
	}

	// Verify the card has expected properties
	if cards[0].Name == "" {
		t.Error("Expected card name to be set")
	}

	if cards[0].OracleID == nil || *cards[0].OracleID == "" {
		t.Error("Expected oracle ID to be set")
	}

	// Second call should hit the cache
	cachedCards, err := Query(query)
	if err != nil {
		t.Fatalf("Cached query failed: %v", err)
	}

	if len(cachedCards) != len(cards) {
		t.Errorf("Expected %d cached cards, got %d", len(cards), len(cachedCards))
	}

	// Verify cached result has same oracle ID
	if cachedCards[0].OracleID == nil || cards[0].OracleID == nil || *cachedCards[0].OracleID != *cards[0].OracleID {
		t.Error("Cached card should have same oracle ID as original")
	}
}

func TestQueryWithContext_EmptyDatabase(t *testing.T) {
	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := "Sol Ring"

	// First call should hit the API
	cards, err := QueryWithContext(ctx, query)
	if err != nil {
		t.Fatalf("QueryWithContext failed: %v", err)
	}

	if len(cards) == 0 {
		t.Fatal("Expected cards to be returned, got empty slice")
	}

	// Verify card properties
	if cards[0].Name == "" {
		t.Error("Expected card name to be set")
	}
}

func TestQueryCard_EmptyDatabase(t *testing.T) {
	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb

	// Test querying for a specific card
	cardQuery := "Black Lotus"

	card, err := QueryCard(cardQuery)
	if err != nil {
		t.Fatalf("QueryCard failed: %v", err)
	}

	if card == nil {
		t.Fatal("Expected card to be returned, got nil")
	}

	if card.Name == "" {
		t.Error("Expected card name to be set")
	}

	if card.OracleID == nil || *card.OracleID == "" {
		t.Error("Expected oracle ID to be set")
	}
}

func TestQueryCardWithContext_EmptyDatabase(t *testing.T) {
	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cardQuery := "Ancestral Recall"

	card, err := QueryCardWithContext(ctx, cardQuery)
	if err != nil {
		t.Fatalf("QueryCardWithContext failed: %v", err)
	}

	if card == nil {
		t.Fatal("Expected card to be returned, got nil")
	}

	if card.Name == "" {
		t.Error("Expected card name to be set")
	}
}

func TestQuery_CacheBehavior(t *testing.T) {
	// This test verifies the cache behavior works correctly
	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb

	ctx := context.Background()
	query := "Counterspell"

	// Verify query is not in cache initially
	_, err := sb.FetchCardsByQuery(ctx, query)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows for uncached query, got: %v", err)
	}

	// First Query() call should hit API and cache results
	cards, err := Query(query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(cards) == 0 {
		t.Fatal("Expected cards to be returned")
	}

	// Now the query should be in cache
	cachedCards, err := sb.FetchCardsByQuery(ctx, query)
	if err != nil {
		t.Fatalf("FetchCardsByQuery should succeed after caching: %v", err)
	}

	if len(cachedCards) != len(cards) {
		t.Errorf("Expected %d cached cards, got %d", len(cards), len(cachedCards))
	}
}

func TestQuery_EmptyResults(t *testing.T) {
	// Test handling of queries that return no results
	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb

	// Note: Scryfall API returns 404 for queries with no results
	// This is expected behavior from the API
	query := "cmc=99" // No card has exactly CMC 99

	cards, err := Query(query)

	// Scryfall returns 404 for empty results, which our client treats as an error
	// This is expected behavior, so we check for the error
	if err != nil {
		// This is expected - Scryfall returns 404 for empty results
		if cards != nil {
			t.Errorf("Expected nil cards when error occurs, got %v", cards)
		}
		return
	}

	// If no error (some versions of API might return empty list),
	// should return empty slice
	if cards == nil {
		t.Fatal("Expected empty slice, got nil")
	}

	if len(cards) != 0 {
		t.Errorf("Expected empty slice, got %d cards", len(cards))
	}
}

func TestScryball_Query_Methods(t *testing.T) {
	// Test the instance methods on Scryball
	sb := setupTestScryball(t)
	defer sb.db.Close()

	query := "Lightning Bolt"

	// Test Query() method
	cards, err := sb.Query(query)
	if err != nil {
		t.Fatalf("Scryball.Query failed: %v", err)
	}

	if len(cards) == 0 {
		t.Fatal("Expected cards to be returned")
	}

	// Test QueryWithContext() method
	ctx := context.Background()
	cards2, err := sb.QueryWithContext(ctx, query)
	if err != nil {
		t.Fatalf("Scryball.QueryWithContext failed: %v", err)
	}

	if len(cards2) == 0 {
		t.Fatal("Expected cards to be returned")
	}

	// Should be same results (from cache)
	if len(cards) != len(cards2) {
		t.Errorf("Expected same number of cards: %d vs %d", len(cards), len(cards2))
	}
}

func TestFetchCardsByQuery_SqlErrNoRows(t *testing.T) {
	// Test that FetchCardsByQuery returns sql.ErrNoRows for uncached queries
	sb := setupTestScryball(t)
	defer sb.db.Close()

	ctx := context.Background()
	query := "uncached query test"

	cards, err := sb.FetchCardsByQuery(ctx, query)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got: %v", err)
	}

	if cards != nil {
		t.Errorf("Expected nil cards, got: %v", cards)
	}
}

func TestFetchCardByExactName_SqlErrNoRows(t *testing.T) {
	// Test that FetchCardByExactName returns sql.ErrNoRows for uncached cards
	sb := setupTestScryball(t)
	defer sb.db.Close()

	ctx := context.Background()
	name := "uncached card name"

	card, err := sb.FetchCardByExactName(ctx, name)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got: %v", err)
	}

	if card != nil {
		t.Errorf("Expected nil card, got: %v", card)
	}
}

// TestIntegration_FullFlow tests the complete flow from empty DB to cached results
func TestIntegration_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sb := setupTestScryball(t)
	defer sb.db.Close()

	CurrentScryball = sb
	ctx := context.Background()

	// Test multiple queries to verify caching works across different searches
	queries := []string{
		"Lightning Bolt",
		"Counterspell",
		"Sol Ring",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			// Verify not in cache initially
			_, err := sb.FetchCardsByQuery(ctx, query)
			if err != sql.ErrNoRows {
				t.Errorf("Query %s should not be cached initially, got error: %v", query, err)
			}

			// Query via API (should cache results)
			cards, err := Query(query)
			if err != nil {
				t.Fatalf("Query %s failed: %v", query, err)
			}

			if len(cards) == 0 {
				t.Fatalf("Query %s returned no cards", query)
			}

			// Verify now cached
			cachedCards, err := sb.FetchCardsByQuery(ctx, query)
			if err != nil {
				t.Fatalf("Query %s should be cached after API call: %v", query, err)
			}

			if len(cachedCards) != len(cards) {
				t.Errorf("Query %s: cached cards count mismatch: %d vs %d", query, len(cachedCards), len(cards))
			}
		})
	}
}
