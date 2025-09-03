package scryball

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ninesl/scryball/internal/scryfall"
)

// testHelper creates a temporary database for testing
func testHelper(t *testing.T) *Scryball {
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

func TestQuery(t *testing.T) {
	sb := testHelper(t)
	defer sb.db.Close()
	CurrentScryball = sb

	t.Run("basic_query", func(t *testing.T) {
		query := "Lightning Bolt"

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
	})

	t.Run("cache_hit", func(t *testing.T) {
		query := "Sol Ring"

		// First call should hit the API since DB is empty
		cards, err := Query(query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(cards) == 0 {
			t.Fatal("Expected cards to be returned, got empty slice")
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
	})

	t.Run("with_context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "Counterspell"

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
	})
}

func TestQueryCard(t *testing.T) {
	sb := testHelper(t)
	defer sb.db.Close()
	CurrentScryball = sb

	t.Run("basic_card_query", func(t *testing.T) {
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
	})

	t.Run("with_context", func(t *testing.T) {
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
	})
}

func TestOracleID(t *testing.T) {
	sb := testHelper(t)
	defer sb.db.Close()
	CurrentScryball = sb

	t.Run("basic_oracle_id_query", func(t *testing.T) {
		// Test using Lightning Bolt's Oracle ID
		lightningBoltOracleID := "4457ed35-7c10-48c8-9776-456485fdf070"

		card, err := QueryCardByOracleID(lightningBoltOracleID)
		if err != nil {
			t.Fatalf("Failed to query card by Oracle ID: %v", err)
		}

		if card == nil {
			t.Fatal("Expected card, got nil")
		}

		if card.OracleID == nil || *card.OracleID != lightningBoltOracleID {
			t.Fatalf("Expected Oracle ID %s, got %v", lightningBoltOracleID, card.OracleID)
		}

		if card.Name != "Lightning Bolt" {
			t.Fatalf("Expected name 'Lightning Bolt', got %s", card.Name)
		}
	})

	t.Run("with_context", func(t *testing.T) {
		ctx := context.Background()
		lightningBoltOracleID := "4457ed35-7c10-48c8-9776-456485fdf070"

		card, err := QueryCardByOracleIDWithContext(ctx, lightningBoltOracleID)
		if err != nil {
			t.Fatalf("Failed to query card by Oracle ID with context: %v", err)
		}

		if card == nil {
			t.Fatal("Expected card with context, got nil")
		}

		if card.Name != "Lightning Bolt" {
			t.Fatalf("Expected name 'Lightning Bolt' with context, got %s", card.Name)
		}
	})

	t.Run("caching_behavior", func(t *testing.T) {
		lightningBoltOracleID := "4457ed35-7c10-48c8-9776-456485fdf070"

		// First call - should fetch from API
		start1 := time.Now()
		card1, err := QueryCardByOracleID(lightningBoltOracleID)
		duration1 := time.Since(start1)

		if err != nil {
			t.Fatalf("Failed first query: %v", err)
		}

		if card1 == nil {
			t.Fatal("Expected card from first query, got nil")
		}

		// Second call - should use cache
		start2 := time.Now()
		card2, err := QueryCardByOracleID(lightningBoltOracleID)
		duration2 := time.Since(start2)

		if err != nil {
			t.Fatalf("Failed second query: %v", err)
		}

		if card2 == nil {
			t.Fatal("Expected card from second query, got nil")
		}

		// Verify cards are the same
		if card1.Name != card2.Name {
			t.Fatalf("Cards have different names: %s vs %s", card1.Name, card2.Name)
		}

		if *card1.OracleID != *card2.OracleID {
			t.Fatalf("Cards have different Oracle IDs: %s vs %s", *card1.OracleID, *card2.OracleID)
		}

		t.Logf("First call (API): %v", duration1)
		t.Logf("Second call (cache): %v", duration2)
		t.Logf("Successfully cached card: %s", card1.Name)
	})
}

func TestScryballInstance(t *testing.T) {
	t.Run("basic_instance_methods", func(t *testing.T) {
		// Test the instance methods
		config := ScryballConfig{
			DBPath: ":memory:",
		}
		sb, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("Failed to create Scryball instance: %v", err)
		}

		// Test using Lightning Bolt's Oracle ID
		lightningBoltOracleID := "4457ed35-7c10-48c8-9776-456485fdf070"

		card, err := sb.QueryCardByOracleID(lightningBoltOracleID)
		if err != nil {
			t.Fatalf("Failed to query card by Oracle ID on instance: %v", err)
		}

		if card == nil {
			t.Fatal("Expected card, got nil")
		}

		if card.Name != "Lightning Bolt" {
			t.Fatalf("Expected name 'Lightning Bolt', got %s", card.Name)
		}

		// Test with context
		ctx := context.Background()
		card2, err := sb.QueryCardByOracleIDWithContext(ctx, lightningBoltOracleID)
		if err != nil {
			t.Fatalf("Failed to query card by Oracle ID with context on instance: %v", err)
		}

		if card2 == nil {
			t.Fatal("Expected card with context, got nil")
		}

		if card2.Name != "Lightning Bolt" {
			t.Fatalf("Expected name 'Lightning Bolt' with context, got %s", card2.Name)
		}
	})

	t.Run("query_methods", func(t *testing.T) {
		sb := testHelper(t)
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
	})
}

func TestCacheBehavior(t *testing.T) {
	sb := testHelper(t)
	defer sb.db.Close()
	CurrentScryball = sb

	t.Run("cache_miss_then_hit", func(t *testing.T) {
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
	})

	t.Run("fetch_card_by_exact_name", func(t *testing.T) {
		ctx := context.Background()
		name := "uncached card name"

		card, err := sb.FetchCardByExactName(ctx, name)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got: %v", err)
		}

		if card != nil {
			t.Errorf("Expected nil card, got: %v", card)
		}
	})

	t.Run("fetch_cards_by_query", func(t *testing.T) {
		ctx := context.Background()
		query := "uncached query test"

		cards, err := sb.FetchCardsByQuery(ctx, query)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got: %v", err)
		}

		if cards != nil {
			t.Errorf("Expected nil cards, got: %v", cards)
		}
	})
}

func TestConfiguration(t *testing.T) {
	t.Run("with_config_defaults_to_memory", func(t *testing.T) {
		// Test that empty DBPath defaults to in-memory
		config := ScryballConfig{
			// DBPath is empty - should default to in-memory
			AppUserAgent: "TestAgent/1.0",
		}

		sb, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("WithConfig with empty DBPath failed: %v", err)
		}
		defer sb.db.Close()

		// Verify we can use the database
		ctx := context.Background()

		// Try to insert a test query cache
		testQuery := "test:query"
		err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
			QueryText: testQuery,
			OracleIds: "[]",
		})
		if err != nil {
			t.Errorf("Failed to insert into in-memory database: %v", err)
		}

		// Verify we can retrieve it
		cached, err := sb.queries.GetCachedQuery(ctx, testQuery)
		if err != nil {
			t.Errorf("Failed to retrieve from in-memory database: %v", err)
		}
		if cached.QueryText != testQuery {
			t.Errorf("Retrieved query text mismatch: got %q, want %q", cached.QueryText, testQuery)
		}
	})

	t.Run("file_database", func(t *testing.T) {
		// Create temp directory for test
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		config := ScryballConfig{
			DBPath: dbPath,
		}

		sb, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("WithConfig with file path failed: %v", err)
		}
		defer sb.db.Close()

		// Verify database is functional
		ctx := context.Background()
		err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
			QueryText: "file:test",
			OracleIds: "[]",
		})
		if err != nil {
			t.Errorf("Failed to use file database: %v", err)
		}
	})

	t.Run("creates_directories", func(t *testing.T) {
		// Test that parent directories are created
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "nested", "dirs", "test.db")

		config := ScryballConfig{
			DBPath: dbPath,
		}

		sb, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("WithConfig failed to create nested directories: %v", err)
		}
		defer sb.db.Close()
	})
}

// TestIntegrationFlow tests the complete flow from empty DB to cached results
func TestIntegrationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sb := testHelper(t)
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
