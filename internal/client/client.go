package client

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ninesl/scryball/internal/scryfall"
	_ "modernc.org/sqlite"
)

const (
	APIBaseURL       = "https://api.scryfall.com"
	DefaultUserAgent = "MTGScryfallClient/1.0"
	DefaultAccept    = "application/json;q=0.9,*/*;q=0.8"
)

var (
	DefaultClientOptions = ClientOptions{
		APIURL:    APIBaseURL,
		UserAgent: DefaultUserAgent,
		Accept:    DefaultAccept,
		Client:    &http.Client{},
	}
)

type Client struct {
	baseURL   string
	userAgent string
	accept    string
	client    *http.Client
	db        *sql.DB
}

type ClientOptions struct {
	APIURL    string       // default is "https://api.scryfall.com"
	UserAgent string       // API docs recomend "{AppName}/1.0"
	Accept    string       // "application/json;q=0.9,*/*;q=0.8". could be used to take csv? TODO:
	Client    *http.Client // any http client can be used
	ProxyURL  string       // optional proxy URL (e.g., "http://proxy:8080")
}

// Uses DefaultClientOptions
func NewClient(appName string) (*Client, error) {
	DefaultClientOptions.UserAgent = fmt.Sprintf("%s/1.0", strings.TrimSpace(appName))

	// Check for proxy URL in environment variable
	if proxyURL := os.Getenv("SCRYFALL_PROXY_URL"); proxyURL != "" {
		DefaultClientOptions.ProxyURL = proxyURL
	}

	return NewClientWithOptions(DefaultClientOptions)
}

func NewClientWithOptions(co ClientOptions) (*Client, error) {
	// Initialize database
	db, err := sql.Open("sqlite", "scryfall.db")
	if err != nil {
		return nil, err
	}

	// Configure HTTP client with proxy if provided
	client := co.Client
	if co.ProxyURL != "" {
		proxyURL, err := url.Parse(co.ProxyURL)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("invalid proxy URL '%s': %v", co.ProxyURL, err)
		}

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client = &http.Client{Transport: transport}

		fmt.Printf("Using proxy: %s\n", co.ProxyURL)
	}

	return &Client{
		baseURL:   co.APIURL,
		userAgent: co.UserAgent,
		accept:    co.Accept,
		client:    client,
		db:        db,
	}, nil
}

func (c *Client) makeRequest(endpoint string, result interface{}) error {
	// Respect Scryfall's rate limit: 50-100ms delay between requests (10 requests per second)
	time.Sleep(100 * time.Millisecond)

	fullURL := c.baseURL + endpoint

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", c.accept)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) GetCard(id string) (*Card, error) {
	var card Card
	err := c.makeRequest("/cards/"+url.PathEscape(id), &card)
	return &card, err
}

func (c *Client) getSet(code string) (*Set, error) {
	var set Set
	err := c.makeRequest("/sets/"+url.PathEscape(code), &set)
	return &set, err
}

func (c *Client) SearchCards(query string) (*List, error) {
	var list List
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}

// searchCards is a private helper method that wraps SearchCards for internal use
// This maintains compatibility with existing code that expects searchCards
func (c *Client) searchCards(query string) (*List, error) {
	return c.SearchCards(query)
}

func (c *Client) SearchCardsByName(name string) (*List, error) {
	var list List
	query := "!\"" + name + "\""
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}

// FetchAllPrintings retrieves all printings for a given card using its PrintsSearchURI.
// This function handles pagination to retrieve ALL printings across all pages.
// Returns an array of Cards (each representing a printing) or an error if the request fails.
func (c *Client) FetchAllPrintings(card *Card) ([]Card, error) {
	var allPrintings []Card

	if card.PrintsSearchURI.String() == "" {
		return nil, fmt.Errorf("card has no prints_search_uri: %s", card.Name)
	}

	// Get first page of printings
	var list List
	// Use the full URL from PrintsSearchURI directly
	err := c.makeRequest(card.PrintsSearchURI.RequestURI(), &list)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch printings for card '%s' from URI '%s': %w", card.Name, card.PrintsSearchURI.String(), err)
	}

	// Add first page results
	allPrintings = append(allPrintings, list.Data...)

	// Follow pagination to get all pages
	for list.HasMore && list.NextPage != nil {
		// Use the full URL from NextPage directly
		err = c.makeRequest(list.NextPage.RequestURI(), &list)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page of printings for card '%s': %w", card.Name, err)
		}

		// Add this page's results
		allPrintings = append(allPrintings, list.Data...)
	}

	return allPrintings, nil
}

// Helper functions

// Helper function to convert int slice to comma-separated string
func intsToString(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	strs := make([]string, len(ints))
	for i, v := range ints {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// Helper function to convert pointer to sql.NullString
func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Helper function to convert pointer to sql.NullInt64
func ptrToNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// Helper function to convert pointer to sql.NullBool
func ptrToNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{Valid: false}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// Helper function to convert string to sql.NullString
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Helper function to convert any value to JSON string
func toJSONString(v interface{}) sql.NullString {
	if v == nil {
		return sql.NullString{Valid: false}
	}

	// Handle empty slices and maps
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case []int:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case map[string]string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case map[string]*string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: string(jsonBytes), Valid: true}
}

// toJSONStringDirect converts interface{} to JSON string directly (not sql.NullString)
func toJSONStringDirect(v interface{}) string {
	if v == nil {
		return "[]"
	}

	// Handle empty slices and maps
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return "[]"
		}
	case []int:
		if len(val) == 0 {
			return "[]"
		}
	case map[string]string:
		if len(val) == 0 {
			return "{}"
		}
	case map[string]*string:
		if len(val) == 0 {
			return "{}"
		}
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// containsFinish checks if a finish type exists in the finishes array
func containsFinish(finishes []string, finish string) bool {
	for _, f := range finishes {
		if f == finish {
			return true
		}
	}
	return false
}

func isArenaSet(games []string) bool {
	for _, game := range games {
		if game == "arena" {
			return true
		}
	}
	return false
}

func shouldIncludeCard(printings []Card) bool {
	// Check if any printing is common/uncommon on Arena
	for _, printing := range printings {
		if isArenaSet(printing.Games) && (printing.Rarity == "common" || printing.Rarity == "uncommon") {
			return false
		}
	}
	return true
}

// queryAndInsertCards fetches cards from Scryfall API and inserts them into database
func (c *Client) queryAndInsertCards(db *sql.DB) error {
	ctx := context.Background()
	queries := scryfall.New(db)

	searchQuery := "(game:paper game:mtgo -game:arena in:common or in:uncommon) game:arena r>=rare"
	fmt.Printf("Searching for query: %s\n", searchQuery)

	results, err := c.searchCards(searchQuery)
	if err != nil {
		return fmt.Errorf("search error: %v", err)
	}

	fmt.Printf("Found %d cards\n", results.TotalCards)

	insertedCount := 0
	for _, card := range results.Data {
		fmt.Printf("Fetching printings for %s...\n", card.Name)

		printings, err := c.FetchAllPrintings(&card)
		if err != nil {
			log.Printf("Error fetching printings for %s: %v", card.Name, err)
			continue
		}

		// Filter out cards that have common/uncommon Arena printings
		if !shouldIncludeCard(printings) {
			fmt.Printf("Skipping %s - has common/uncommon Arena printing\n", card.Name)
			continue
		}

		// First, insert the card (oracle-level data) - this will be upserted if it already exists
		err = queries.UpsertCard(ctx, scryfall.UpsertCardParams{
			OracleID:        *card.OracleID,
			Name:            card.Name,
			Layout:          card.Layout,
			PrintsSearchUri: card.PrintsSearchURI.String(),
			RulingsUri:      card.RulingsURI.String(),
			AllParts:        toJSONString(card.AllParts),
			CardFaces:       toJSONString(card.CardFaces),
			Cmc:             card.CMC,
			ColorIdentity:   toJSONStringDirect(card.ColorIdentity),
			ColorIndicator:  toJSONString(card.ColorIndicator),
			Colors:          toJSONString(card.Colors),
			Defense:         ptrToNullString(card.Defense),
			EdhrecRank:      ptrToNullInt64(card.EDHRecRank),
			GameChanger:     ptrToNullBool(card.GameChanger),
			HandModifier:    ptrToNullString(card.HandModifier),
			Keywords:        toJSONStringDirect(card.Keywords),
			Legalities:      toJSONStringDirect(card.Legalities),
			LifeModifier:    ptrToNullString(card.LifeModifier),
			Loyalty:         ptrToNullString(card.Loyalty),
			ManaCost:        ptrToNullString(card.ManaCost),
			OracleText:      ptrToNullString(card.OracleText),
			PennyRank:       ptrToNullInt64(card.PennyRank),
			Power:           ptrToNullString(card.Power),
			ProducedMana:    toJSONString(card.ProducedMana),
			Reserved:        card.Reserved,
			Toughness:       ptrToNullString(card.Toughness),
			TypeLine:        card.TypeLine,
		})

		if err != nil {
			log.Printf("Error inserting card %s: %v", card.Name, err)
			continue
		}

		// Add to eternal_artisan_exception table
		err = queries.AddEternalArtisanException(ctx, *card.OracleID)
		if err != nil {
			log.Printf("Error adding to eternal_artisan_exception %s: %v", card.Name, err)
			continue
		}

		// Then insert ALL printings of this card
		for _, printing := range printings {
			err = queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
				ID:                printing.ID,
				OracleID:          *printing.OracleID,
				ArenaID:           ptrToNullInt64(printing.ArenaID),
				Lang:              printing.Lang,
				MtgoID:            ptrToNullInt64(printing.MTGOID),
				MtgoFoilID:        ptrToNullInt64(printing.MTGOFoilID),
				MultiverseIds:     toJSONString(printing.MultiverseIDs),
				TcgplayerID:       ptrToNullInt64(printing.TCGPlayerID),
				TcgplayerEtchedID: ptrToNullInt64(printing.TCGPlayerEtchedID),
				CardmarketID:      ptrToNullInt64(printing.CardmarketID),
				Object:            printing.Object,
				ScryfallUri:       printing.ScryfallURI.String(),
				Uri:               printing.URI.String(),
				Artist:            ptrToNullString(printing.Artist),
				ArtistIds:         toJSONString(printing.ArtistIDs),
				AttractionLights:  toJSONString(printing.AttractionLights),
				Booster:           printing.Booster,
				BorderColor:       printing.BorderColor,
				CardBackID:        printing.CardBackID,
				CollectorNumber:   printing.CollectorNumber,
				ContentWarning:    ptrToNullBool(printing.ContentWarning),
				Digital:           printing.Digital,
				Finishes:          toJSONStringDirect(printing.Finishes),
				FlavorName:        ptrToNullString(printing.FlavorName),
				FlavorText:        ptrToNullString(printing.FlavorText),
				Foil:              containsFinish(printing.Finishes, "foil"),
				Nonfoil:           containsFinish(printing.Finishes, "nonfoil"),
				FrameEffects:      toJSONString(printing.FrameEffects),
				Frame:             printing.Frame,
				FullArt:           printing.FullArt,
				Games:             toJSONStringDirect(printing.Games),
				HighresImage:      printing.HighresImage,
				IllustrationID:    ptrToNullString(printing.IllustrationID),
				ImageStatus:       printing.ImageStatus,
				ImageUris:         toJSONString(printing.ImageURIs),
				Oversized:         printing.Oversized,
				Prices:            toJSONStringDirect(printing.Prices),
				PrintedName:       ptrToNullString(printing.PrintedName),
				PrintedText:       ptrToNullString(printing.PrintedText),
				PrintedTypeLine:   ptrToNullString(printing.PrintedTypeLine),
				Promo:             printing.Promo,
				PromoTypes:        toJSONString(printing.PromoTypes),
				PurchaseUris:      toJSONString(printing.PurchaseURIs),
				Rarity:            printing.Rarity,
				RelatedUris:       toJSONStringDirect(printing.RelatedURIs),
				ReleasedAt:        printing.ReleasedAt,
				Reprint:           printing.Reprint,
				ScryfallSetUri:    printing.ScryfallSetURI.String(),
				SetName:           printing.SetName,
				SetSearchUri:      printing.SetSearchURI.String(),
				SetType:           printing.SetType,
				SetUri:            printing.SetURI.String(),
				Set:               printing.Set,
				SetID:             printing.SetID,
				StorySpotlight:    printing.StorySpotlight,
				Textless:          printing.Textless,
				Variation:         printing.Variation,
				VariationOf:       ptrToNullString(printing.VariationOf),
				SecurityStamp:     ptrToNullString(printing.SecurityStamp),
				Watermark:         ptrToNullString(printing.Watermark),
				Preview:           toJSONString(printing.Preview),
			})

			if err != nil {
				log.Printf("Error inserting printing %s (%s): %v", printing.Name, printing.Set, err)
				continue
			}

			insertedCount++
			fmt.Printf("Inserted %s (%s - %s)\n", printing.Name, printing.Set, printing.Rarity)
		}
	}

	fmt.Printf("\nInserted %d filtered cards into database\n", insertedCount)
	return nil
}

// loadCardsFromDatabase loads cards from database and returns them as []Card with printings grouped
func (c *Client) loadCardsFromDatabase(db *sql.DB) ([]Card, error) {
	ctx := context.Background()
	queries := scryfall.New(db)

	cardPrintings, err := queries.GetCardsWithPrintings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading cards: %v", err)
	}

	// Group printings by oracle_id to create unique cards
	cardMap := make(map[string]*Card)

	for _, row := range cardPrintings {
		// Check if we already have this card
		if existingCard, exists := cardMap[row.OracleID]; exists {
			// Add this printing's games to the existing card's games
			if row.Games != "" {
				var printingGames []string
				json.Unmarshal([]byte(row.Games), &printingGames)

				// Merge games without duplicates
				gameSet := make(map[string]bool)
				for _, game := range existingCard.Games {
					gameSet[game] = true
				}
				for _, game := range printingGames {
					gameSet[game] = true
				}

				// Convert back to slice
				var allGames []string
				for game := range gameSet {
					allGames = append(allGames, game)
				}
				existingCard.Games = allGames
			}
		} else {
			// Create new card entry
			card := Card{
				ID:       row.OracleID, // Use oracle_id as the main ID for the card
				Name:     row.Name,
				Layout:   row.Layout,
				OracleID: &row.OracleID,
				CMC:      row.Cmc,
				TypeLine: row.TypeLine,
			}

			// Handle nullable fields
			if row.ManaCost.Valid {
				card.ManaCost = &row.ManaCost.String
			}
			if row.OracleText.Valid {
				card.OracleText = &row.OracleText.String
			}

			// Parse JSON fields
			if row.Games != "" {
				json.Unmarshal([]byte(row.Games), &card.Games)
			}
			if row.ColorIdentity != "" {
				json.Unmarshal([]byte(row.ColorIdentity), &card.ColorIdentity)
			}
			if row.Colors.Valid && row.Colors.String != "" {
				json.Unmarshal([]byte(row.Colors.String), &card.Colors)
			}

			cardMap[row.OracleID] = &card
		}
	}

	// Convert map to slice
	var cards []Card
	for _, card := range cardMap {
		cards = append(cards, *card)
	}

	return cards, nil
}

// SearchCardsByQuery searches Scryfall API and returns just the cards (not the List wrapper)
// This method handles pagination and returns ALL matching cards, not just the first page
func (c *Client) SearchCardsByQuery(query string) ([]Card, error) {
	return c.SearchAllCardsByQuery(query)
}

// SearchAllCardsByQuery searches Scryfall API and returns ALL cards across all pages
func (c *Client) SearchAllCardsByQuery(query string) ([]Card, error) {
	var allCards []Card

	// Get first page
	list, err := c.searchCards(query)
	if err != nil {
		return nil, err
	}

	// Add first page results
	allCards = append(allCards, list.Data...)

	// Follow pagination to get all pages
	for list.HasMore && list.NextPage != nil {
		// Extract the path and query from the next page URL
		nextEndpoint := list.NextPage.Path
		if list.NextPage.RawQuery != "" {
			nextEndpoint += "?" + list.NextPage.RawQuery
		}

		// Make request for next page
		err = c.makeRequest(nextEndpoint, &list)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page: %v", err)
		}

		// Add this page's results
		allCards = append(allCards, list.Data...)
	}

	return allCards, nil
}

// FetchFilteredScryfallAPI fetches filtered cards from Scryfall API and populates the database
func (c *Client) FetchFilteredScryfallAPI() error {
	return c.queryAndInsertCards(c.db)
}

// GetFilteredCards returns all filtered cards from the database as []Card
func (c *Client) GetFilteredCards() ([]Card, error) {
	return c.loadCardsFromDatabase(c.db)
}

// queryAndInsertArenaOnlyCards fetches Arena-only cards from Scryfall API and inserts them into database
func (c *Client) queryAndInsertArenaOnlyCards(db *sql.DB) error {
	ctx := context.Background()
	queries := scryfall.New(db)

	// Use the exact query for Arena-only common/uncommon original cards
	searchQuery := "in:arena -in:paper (rarity:common or rarity:uncommon) -is:rebalanced"
	fmt.Printf("Searching for Arena-only cards: %s\n", searchQuery)

	results, err := c.searchCards(searchQuery)
	if err != nil {
		return fmt.Errorf("search error: %v", err)
	}

	fmt.Printf("Found %d Arena-only cards\n", results.TotalCards)

	insertedCount := 0
	for _, card := range results.Data {
		fmt.Printf("Processing Arena-only card: %s...\n", card.Name)

		// First, insert the card (oracle-level data) - this will be upserted if it already exists
		err = queries.UpsertCard(ctx, scryfall.UpsertCardParams{
			OracleID:        *card.OracleID,
			Name:            card.Name,
			Layout:          card.Layout,
			PrintsSearchUri: card.PrintsSearchURI.String(),
			RulingsUri:      card.RulingsURI.String(),
			AllParts:        toJSONString(card.AllParts),
			CardFaces:       toJSONString(card.CardFaces),
			Cmc:             card.CMC,
			ColorIdentity:   toJSONStringDirect(card.ColorIdentity),
			ColorIndicator:  toJSONString(card.ColorIndicator),
			Colors:          toJSONString(card.Colors),
			Defense:         ptrToNullString(card.Defense),
			EdhrecRank:      ptrToNullInt64(card.EDHRecRank),
			GameChanger:     ptrToNullBool(card.GameChanger),
			HandModifier:    ptrToNullString(card.HandModifier),
			Keywords:        toJSONStringDirect(card.Keywords),
			Legalities:      toJSONStringDirect(card.Legalities),
			LifeModifier:    ptrToNullString(card.LifeModifier),
			Loyalty:         ptrToNullString(card.Loyalty),
			ManaCost:        ptrToNullString(card.ManaCost),
			OracleText:      ptrToNullString(card.OracleText),
			PennyRank:       ptrToNullInt64(card.PennyRank),
			Power:           ptrToNullString(card.Power),
			ProducedMana:    toJSONString(card.ProducedMana),
			Reserved:        card.Reserved,
			Toughness:       ptrToNullString(card.Toughness),
			TypeLine:        card.TypeLine,
		})

		if err != nil {
			log.Printf("Error inserting card %s: %v", card.Name, err)
			continue
		}

		// Insert the printing data for this Arena-only card
		err = queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
			ID:                card.ID,
			OracleID:          *card.OracleID,
			ArenaID:           ptrToNullInt64(card.ArenaID),
			Lang:              card.Lang,
			MtgoID:            ptrToNullInt64(card.MTGOID),
			MtgoFoilID:        ptrToNullInt64(card.MTGOFoilID),
			MultiverseIds:     toJSONString(card.MultiverseIDs),
			TcgplayerID:       ptrToNullInt64(card.TCGPlayerID),
			TcgplayerEtchedID: ptrToNullInt64(card.TCGPlayerEtchedID),
			CardmarketID:      ptrToNullInt64(card.CardmarketID),
			Object:            card.Object,
			ScryfallUri:       card.ScryfallURI.String(),
			Uri:               card.URI.String(),
			Artist:            ptrToNullString(card.Artist),
			ArtistIds:         toJSONString(card.ArtistIDs),
			AttractionLights:  toJSONString(card.AttractionLights),
			Booster:           card.Booster,
			BorderColor:       card.BorderColor,
			CardBackID:        card.CardBackID,
			CollectorNumber:   card.CollectorNumber,
			ContentWarning:    ptrToNullBool(card.ContentWarning),
			Digital:           card.Digital,
			Finishes:          toJSONStringDirect(card.Finishes),
			FlavorName:        ptrToNullString(card.FlavorName),
			FlavorText:        ptrToNullString(card.FlavorText),
			Foil:              containsFinish(card.Finishes, "foil"),
			Nonfoil:           containsFinish(card.Finishes, "nonfoil"),
			FrameEffects:      toJSONString(card.FrameEffects),
			Frame:             card.Frame,
			FullArt:           card.FullArt,
			Games:             toJSONStringDirect(card.Games),
			HighresImage:      card.HighresImage,
			IllustrationID:    ptrToNullString(card.IllustrationID),
			ImageStatus:       card.ImageStatus,
			ImageUris:         toJSONString(card.ImageURIs),
			Oversized:         card.Oversized,
			Prices:            toJSONStringDirect(card.Prices),
			PrintedName:       ptrToNullString(card.PrintedName),
			PrintedText:       ptrToNullString(card.PrintedText),
			PrintedTypeLine:   ptrToNullString(card.PrintedTypeLine),
			Promo:             card.Promo,
			PromoTypes:        toJSONString(card.PromoTypes),
			PurchaseUris:      toJSONString(card.PurchaseURIs),
			Rarity:            card.Rarity,
			RelatedUris:       toJSONStringDirect(card.RelatedURIs),
			ReleasedAt:        card.ReleasedAt,
			Reprint:           card.Reprint,
			ScryfallSetUri:    card.ScryfallSetURI.String(),
			SetName:           card.SetName,
			SetSearchUri:      card.SetSearchURI.String(),
			SetType:           card.SetType,
			SetUri:            card.SetURI.String(),
			Set:               card.Set,
			SetID:             card.SetID,
			StorySpotlight:    card.StorySpotlight,
			Textless:          card.Textless,
			Variation:         card.Variation,
			VariationOf:       ptrToNullString(card.VariationOf),
			SecurityStamp:     ptrToNullString(card.SecurityStamp),
			Watermark:         ptrToNullString(card.Watermark),
			Preview:           toJSONString(card.Preview),
		})

		if err != nil {
			log.Printf("Error inserting printing for %s: %v", card.Name, err)
			continue
		}

		// Add to arena_only_ea_cards table
		err = queries.AddArenaOnlyEACard(ctx, *card.OracleID)
		if err != nil {
			log.Printf("Error adding to arena_only_ea_cards %s: %v", card.Name, err)
			continue
		}

		insertedCount++
		fmt.Printf("Inserted Arena-only card: %s (%s - %s)\n", card.Name, card.Set, card.Rarity)
	}

	fmt.Printf("\nInserted %d Arena-only cards into database\n", insertedCount)
	return nil
}

// FetchArenaOnlyCards fetches Arena-only cards from Scryfall API and populates the database
func (c *Client) FetchArenaOnlyCards() error {
	return c.queryAndInsertArenaOnlyCards(c.db)
}

// BackfillAllPrintings fetches missing printing data for all cards in all tables
func (c *Client) BackfillAllPrintings() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get all unique oracle_ids from all card tables
	fmt.Println("Gathering all cards from database...")
	allCards, err := queries.GetAllCategorizedCards(ctx)
	if err != nil {
		return fmt.Errorf("error getting all cards: %v", err)
	}

	if len(allCards) == 0 {
		fmt.Println("No cards found in database.")
		return nil
	}

	fmt.Printf("Found %d cards to backfill. This may take a while...\n", len(allCards))

	successCount := 0
	errorCount := 0

	for i, card := range allCards {
		fmt.Printf("Processing %d/%d: %s... ", i+1, len(allCards), card.Name)

		// Fetch all printings for this oracle_id using the search endpoint with unique=prints
		searchQuery := fmt.Sprintf("oracleid:%s unique:prints", card.OracleID)
		printings, err := c.searchCards(searchQuery)
		if err != nil {
			fmt.Printf("ERROR (API request failed: %v)\n", err)
			errorCount++
			continue
		}

		if len(printings.Data) == 0 {
			fmt.Printf("WARNING (no printings found)\n")
			continue
		}

		// Store all printings for this card
		printingsStored := 0
		for _, printing := range printings.Data {
			err = queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
				ID:                printing.ID,
				OracleID:          *printing.OracleID,
				ArenaID:           ptrToNullInt64(printing.ArenaID),
				Lang:              printing.Lang,
				MtgoID:            ptrToNullInt64(printing.MTGOID),
				MtgoFoilID:        ptrToNullInt64(printing.MTGOFoilID),
				MultiverseIds:     toJSONString(printing.MultiverseIDs),
				TcgplayerID:       ptrToNullInt64(printing.TCGPlayerID),
				TcgplayerEtchedID: ptrToNullInt64(printing.TCGPlayerEtchedID),
				CardmarketID:      ptrToNullInt64(printing.CardmarketID),
				Object:            printing.Object,
				ScryfallUri:       printing.ScryfallURI.String(),
				Uri:               printing.URI.String(),
				Artist:            ptrToNullString(printing.Artist),
				ArtistIds:         toJSONString(printing.ArtistIDs),
				AttractionLights:  toJSONString(printing.AttractionLights),
				Booster:           printing.Booster,
				BorderColor:       printing.BorderColor,
				CardBackID:        printing.CardBackID,
				CollectorNumber:   printing.CollectorNumber,
				ContentWarning:    ptrToNullBool(printing.ContentWarning),
				Digital:           printing.Digital,
				Finishes:          toJSONStringDirect(printing.Finishes),
				FlavorName:        ptrToNullString(printing.FlavorName),
				FlavorText:        ptrToNullString(printing.FlavorText),
				Foil:              containsFinish(printing.Finishes, "foil"),
				Nonfoil:           containsFinish(printing.Finishes, "nonfoil"),
				FrameEffects:      toJSONString(printing.FrameEffects),
				Frame:             printing.Frame,
				FullArt:           printing.FullArt,
				Games:             toJSONStringDirect(printing.Games),
				HighresImage:      printing.HighresImage,
				IllustrationID:    ptrToNullString(printing.IllustrationID),
				ImageStatus:       printing.ImageStatus,
				ImageUris:         toJSONString(printing.ImageURIs),
				Oversized:         printing.Oversized,
				Prices:            toJSONStringDirect(printing.Prices),
				PrintedName:       ptrToNullString(printing.PrintedName),
				PrintedText:       ptrToNullString(printing.PrintedText),
				PrintedTypeLine:   ptrToNullString(printing.PrintedTypeLine),
				Promo:             printing.Promo,
				PromoTypes:        toJSONString(printing.PromoTypes),
				PurchaseUris:      toJSONString(printing.PurchaseURIs),
				Rarity:            printing.Rarity,
				RelatedUris:       toJSONStringDirect(printing.RelatedURIs),
				ReleasedAt:        printing.ReleasedAt,
				Reprint:           printing.Reprint,
				ScryfallSetUri:    printing.ScryfallSetURI.String(),
				SetName:           printing.SetName,
				SetSearchUri:      printing.SetSearchURI.String(),
				SetType:           printing.SetType,
				SetUri:            printing.SetURI.String(),
				Set:               printing.Set,
				SetID:             printing.SetID,
				StorySpotlight:    printing.StorySpotlight,
				Textless:          printing.Textless,
				Variation:         printing.Variation,
				VariationOf:       ptrToNullString(printing.VariationOf),
				SecurityStamp:     ptrToNullString(printing.SecurityStamp),
				Watermark:         ptrToNullString(printing.Watermark),
				Preview:           toJSONString(printing.Preview),
			})

			if err != nil {
				fmt.Printf("ERROR (failed to store printing %s: %v)\n", printing.ID, err)
				errorCount++
				break
			}
			printingsStored++
		}

		if printingsStored > 0 {
			fmt.Printf("OK (%d printings stored)\n", printingsStored)
			successCount++
		}

		// Be nice to Scryfall API - add a small delay
		if i%10 == 9 {
			fmt.Println("Pausing briefly to be nice to Scryfall API...")
			// In a real implementation, you'd add time.Sleep(100 * time.Millisecond) here
		}
	}

	fmt.Printf("\nBackfill complete! Successfully processed %d cards, %d errors.\n", successCount, errorCount)
	return nil
}

// searchAndSelectCard searches for cards and lets user select one
func (c *Client) searchAndSelectCard(query string, actionName string) (*Card, error) {
	// Search for cards using the query
	results, err := c.searchCards(query)
	if err != nil {
		return nil, fmt.Errorf("search error: %v", err)
	}

	if len(results.Data) == 0 {
		fmt.Println("No cards found for query:", query)
		return nil, nil
	}

	// Display results and let user pick
	fmt.Printf("Found %d cards:\n", len(results.Data))
	for i, card := range results.Data {
		if i >= 20 { // Limit to first 20 results
			fmt.Printf("... and %d more cards\n", len(results.Data)-20)
			break
		}
		fmt.Printf("%d. %s (%s - %s) [%s]\n", i+1, card.Name, card.Set, card.Rarity, *card.OracleID)
	}

	fmt.Printf("Enter card number to %s (0 to cancel): ", actionName)
	var choice int
	fmt.Scanln(&choice)

	if choice <= 0 || choice > len(results.Data) {
		fmt.Println("Cancelled or invalid choice.")
		return nil, nil
	}

	return &results.Data[choice-1], nil
}

// storeCardWithPrinting stores both card and printing data for a selected card
func (c *Client) storeCardWithPrinting(selectedCard *Card) error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// First ensure the card is in our database
	err := queries.UpsertCard(ctx, scryfall.UpsertCardParams{
		OracleID:        *selectedCard.OracleID,
		Name:            selectedCard.Name,
		Layout:          selectedCard.Layout,
		PrintsSearchUri: selectedCard.PrintsSearchURI.String(),
		RulingsUri:      selectedCard.RulingsURI.String(),
		AllParts:        toJSONString(selectedCard.AllParts),
		CardFaces:       toJSONString(selectedCard.CardFaces),
		Cmc:             selectedCard.CMC,
		ColorIdentity:   toJSONStringDirect(selectedCard.ColorIdentity),
		ColorIndicator:  toJSONString(selectedCard.ColorIndicator),
		Colors:          toJSONString(selectedCard.Colors),
		Defense:         ptrToNullString(selectedCard.Defense),
		EdhrecRank:      ptrToNullInt64(selectedCard.EDHRecRank),
		GameChanger:     ptrToNullBool(selectedCard.GameChanger),
		HandModifier:    ptrToNullString(selectedCard.HandModifier),
		Keywords:        toJSONStringDirect(selectedCard.Keywords),
		Legalities:      toJSONStringDirect(selectedCard.Legalities),
		LifeModifier:    ptrToNullString(selectedCard.LifeModifier),
		Loyalty:         ptrToNullString(selectedCard.Loyalty),
		ManaCost:        ptrToNullString(selectedCard.ManaCost),
		OracleText:      ptrToNullString(selectedCard.OracleText),
		PennyRank:       ptrToNullInt64(selectedCard.PennyRank),
		Power:           ptrToNullString(selectedCard.Power),
		ProducedMana:    toJSONString(selectedCard.ProducedMana),
		Reserved:        selectedCard.Reserved,
		Toughness:       ptrToNullString(selectedCard.Toughness),
		TypeLine:        selectedCard.TypeLine,
	})

	if err != nil {
		return fmt.Errorf("error storing card: %v", err)
	}

	// Also store the printing data for this specific card
	err = queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
		ID:                selectedCard.ID,
		OracleID:          *selectedCard.OracleID,
		ArenaID:           ptrToNullInt64(selectedCard.ArenaID),
		Lang:              selectedCard.Lang,
		MtgoID:            ptrToNullInt64(selectedCard.MTGOID),
		MtgoFoilID:        ptrToNullInt64(selectedCard.MTGOFoilID),
		MultiverseIds:     toJSONString(selectedCard.MultiverseIDs),
		TcgplayerID:       ptrToNullInt64(selectedCard.TCGPlayerID),
		TcgplayerEtchedID: ptrToNullInt64(selectedCard.TCGPlayerEtchedID),
		CardmarketID:      ptrToNullInt64(selectedCard.CardmarketID),
		Object:            selectedCard.Object,
		ScryfallUri:       selectedCard.ScryfallURI.String(),
		Uri:               selectedCard.URI.String(),
		Artist:            ptrToNullString(selectedCard.Artist),
		ArtistIds:         toJSONString(selectedCard.ArtistIDs),
		AttractionLights:  toJSONString(selectedCard.AttractionLights),
		Booster:           selectedCard.Booster,
		BorderColor:       selectedCard.BorderColor,
		CardBackID:        selectedCard.CardBackID,
		CollectorNumber:   selectedCard.CollectorNumber,
		ContentWarning:    ptrToNullBool(selectedCard.ContentWarning),
		Digital:           selectedCard.Digital,
		Finishes:          toJSONStringDirect(selectedCard.Finishes),
		FlavorName:        ptrToNullString(selectedCard.FlavorName),
		FlavorText:        ptrToNullString(selectedCard.FlavorText),
		Foil:              containsFinish(selectedCard.Finishes, "foil"),
		Nonfoil:           containsFinish(selectedCard.Finishes, "nonfoil"),
		FrameEffects:      toJSONString(selectedCard.FrameEffects),
		Frame:             selectedCard.Frame,
		FullArt:           selectedCard.FullArt,
		Games:             toJSONStringDirect(selectedCard.Games),
		HighresImage:      selectedCard.HighresImage,
		IllustrationID:    ptrToNullString(selectedCard.IllustrationID),
		ImageStatus:       selectedCard.ImageStatus,
		ImageUris:         toJSONString(selectedCard.ImageURIs),
		Oversized:         selectedCard.Oversized,
		Prices:            toJSONStringDirect(selectedCard.Prices),
		PrintedName:       ptrToNullString(selectedCard.PrintedName),
		PrintedText:       ptrToNullString(selectedCard.PrintedText),
		PrintedTypeLine:   ptrToNullString(selectedCard.PrintedTypeLine),
		Promo:             selectedCard.Promo,
		PromoTypes:        toJSONString(selectedCard.PromoTypes),
		PurchaseUris:      toJSONString(selectedCard.PurchaseURIs),
		Rarity:            selectedCard.Rarity,
		RelatedUris:       toJSONStringDirect(selectedCard.RelatedURIs),
		ReleasedAt:        selectedCard.ReleasedAt,
		Reprint:           selectedCard.Reprint,
		ScryfallSetUri:    selectedCard.ScryfallSetURI.String(),
		SetName:           selectedCard.SetName,
		SetSearchUri:      selectedCard.SetSearchURI.String(),
		SetType:           selectedCard.SetType,
		SetUri:            selectedCard.SetURI.String(),
		Set:               selectedCard.Set,
		SetID:             selectedCard.SetID,
		StorySpotlight:    selectedCard.StorySpotlight,
		Textless:          selectedCard.Textless,
		Variation:         selectedCard.Variation,
		VariationOf:       ptrToNullString(selectedCard.VariationOf),
		SecurityStamp:     ptrToNullString(selectedCard.SecurityStamp),
		Watermark:         ptrToNullString(selectedCard.Watermark),
		Preview:           toJSONString(selectedCard.Preview),
	})

	if err != nil {
		return fmt.Errorf("error storing printing: %v", err)
	}

	return nil
}

// AddCardToBannedList searches for cards and adds selected card to banned list
func (c *Client) AddCardToBannedList(query string) error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Search and select card
	selectedCard, err := c.searchAndSelectCard(query, "add to banned list")
	if err != nil {
		return err
	}
	if selectedCard == nil {
		return nil // User cancelled or no cards found
	}

	// Store card and printing data
	if err := c.storeCardWithPrinting(selectedCard); err != nil {
		return err
	}

	// Add to banned list
	err = queries.AddBannedCard(ctx, *selectedCard.OracleID)
	if err != nil {
		return fmt.Errorf("error adding to banned list: %v", err)
	}

	fmt.Printf("Added %s to banned list\n", selectedCard.Name)
	return nil
}

// RemoveCardFromBannedList displays banned cards and removes selected card
func (c *Client) RemoveCardFromBannedList() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get all banned cards
	bannedCards, err := queries.GetBannedCards(ctx)
	if err != nil {
		return fmt.Errorf("error getting banned cards: %v", err)
	}

	if len(bannedCards) == 0 {
		fmt.Println("No cards in banned list.")
		return nil
	}

	// Display banned cards
	fmt.Printf("Banned cards (%d):\n", len(bannedCards))
	for i, card := range bannedCards {
		fmt.Printf("%d. %s [%s]\n", i+1, card.Name, card.OracleID)
	}

	fmt.Print("Enter card number to remove from banned list (0 to cancel): ")
	var choice int
	fmt.Scanln(&choice)

	if choice <= 0 || choice > len(bannedCards) {
		fmt.Println("Cancelled or invalid choice.")
		return nil
	}

	selectedCard := bannedCards[choice-1]

	// Remove from banned list
	err = queries.RemoveBannedCard(ctx, selectedCard.OracleID)
	if err != nil {
		return fmt.Errorf("error removing from banned list: %v", err)
	}

	fmt.Printf("Removed %s from banned list\n", selectedCard.Name)
	return nil
}

// AddCardToWatchlist searches for cards and adds selected card to watchlist
func (c *Client) AddCardToWatchlist(query string) error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Search and select card
	selectedCard, err := c.searchAndSelectCard(query, "add to watchlist")
	if err != nil {
		return err
	}
	if selectedCard == nil {
		return nil // User cancelled or no cards found
	}

	// Store card and printing data
	if err := c.storeCardWithPrinting(selectedCard); err != nil {
		return err
	}

	// Add to watchlist
	err = queries.AddWatchlistCard(ctx, *selectedCard.OracleID)
	if err != nil {
		return fmt.Errorf("error adding to watchlist: %v", err)
	}

	fmt.Printf("Added %s to watchlist\n", selectedCard.Name)
	return nil
}

// RemoveCardFromWatchlist displays watchlist cards and removes selected card
func (c *Client) RemoveCardFromWatchlist() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get all watchlist cards
	watchlistCards, err := queries.GetWatchlistCards(ctx)
	if err != nil {
		return fmt.Errorf("error getting watchlist cards: %v", err)
	}

	if len(watchlistCards) == 0 {
		fmt.Println("No cards in watchlist.")
		return nil
	}

	// Display watchlist cards
	fmt.Printf("Watchlist cards (%d):\n", len(watchlistCards))
	for i, card := range watchlistCards {
		fmt.Printf("%d. %s [%s]\n", i+1, card.Name, card.OracleID)
	}

	fmt.Print("Enter card number to remove from watchlist (0 to cancel): ")
	var choice int
	fmt.Scanln(&choice)

	if choice <= 0 || choice > len(watchlistCards) {
		fmt.Println("Cancelled or invalid choice.")
		return nil
	}

	selectedCard := watchlistCards[choice-1]

	// Remove from watchlist
	err = queries.RemoveWatchlistCard(ctx, selectedCard.OracleID)
	if err != nil {
		return fmt.Errorf("error removing from watchlist: %v", err)
	}

	fmt.Printf("Removed %s from watchlist\n", selectedCard.Name)
	return nil
}

// AddDigitalMechanicCards filters arena cards by mechanic keyword and adds them
func (c *Client) AddDigitalMechanicCards(mechanic string) error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get arena cards that contain the mechanic in oracle text
	arenaCards, err := queries.GetArenaCardsByMechanic(ctx, stringToNullString(mechanic))
	if err != nil {
		return fmt.Errorf("error getting arena cards: %v", err)
	}

	if len(arenaCards) == 0 {
		fmt.Printf("No arena cards found with mechanic: %s\n", mechanic)
		return nil
	}

	fmt.Printf("Found %d arena cards with mechanic '%s':\n", len(arenaCards), mechanic)
	addedCount := 0

	for _, card := range arenaCards {
		// Add to digital mechanic cards
		err = queries.AddDigitalMechanicCard(ctx, scryfall.AddDigitalMechanicCardParams{
			OracleID:        card.OracleID,
			MechanicKeyword: stringToNullString(mechanic),
		})
		if err != nil {
			log.Printf("Error adding %s to digital mechanic cards: %v", card.Name, err)
			continue
		}
		fmt.Printf("Added: %s\n", card.Name)
		addedCount++
	}

	fmt.Printf("Added %d cards to digital mechanic list\n", addedCount)
	return nil
}

// RemoveDigitalMechanicCard displays digital mechanic cards and removes selected card
func (c *Client) RemoveDigitalMechanicCard() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get all digital mechanic cards
	mechanicCards, err := queries.GetDigitalMechanicCards(ctx)
	if err != nil {
		return fmt.Errorf("error getting digital mechanic cards: %v", err)
	}

	if len(mechanicCards) == 0 {
		fmt.Println("No cards in digital mechanic list.")
		return nil
	}

	// Display digital mechanic cards
	fmt.Printf("Digital mechanic cards (%d):\n", len(mechanicCards))
	for i, card := range mechanicCards {
		mechanicStr := ""
		if card.MechanicKeyword.Valid {
			mechanicStr = fmt.Sprintf(" (%s)", card.MechanicKeyword.String)
		}
		fmt.Printf("%d. %s%s [%s]\n", i+1, card.Name, mechanicStr, card.OracleID)
	}

	fmt.Print("Enter card number to remove from digital mechanic list (0 to cancel): ")
	var choice int
	fmt.Scanln(&choice)

	if choice <= 0 || choice > len(mechanicCards) {
		fmt.Println("Cancelled or invalid choice.")
		return nil
	}

	selectedCard := mechanicCards[choice-1]

	// Remove from digital mechanic list
	err = queries.RemoveDigitalMechanicCard(ctx, selectedCard.OracleID)
	if err != nil {
		return fmt.Errorf("error removing from digital mechanic list: %v", err)
	}

	fmt.Printf("Removed %s from digital mechanic list\n", selectedCard.Name)
	return nil
}

// GetAllCategorizedCards returns all cards from all tables
func (c *Client) GetAllCategorizedCards() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Get all categorized cards
	cards, err := queries.GetAllCategorizedCards(ctx)
	if err != nil {
		return fmt.Errorf("error getting categorized cards: %v", err)
	}

	if len(cards) == 0 {
		fmt.Println("No categorized cards found.")
		return nil
	}

	// Group by category
	categories := make(map[string][]scryfall.GetAllCategorizedCardsRow)
	for _, card := range cards {
		categories[card.Category] = append(categories[card.Category], card)
	}

	// Display by category
	for category, categoryCards := range categories {
		fmt.Printf("\n=== %s (%d cards) ===\n", category, len(categoryCards))
		for _, card := range categoryCards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}

			mechanicInfo := ""
			if card.MechanicKeyword != "" {
				mechanicInfo = fmt.Sprintf(" [%s]", card.MechanicKeyword)
			}

			fmt.Printf("- %s [%s] (%s)%s - %s\n", card.Name, manaCost, card.OracleID, mechanicInfo, card.TypeLine)
		}
	}

	fmt.Printf("\nTotal: %d cards across all categories\n", len(cards))
	return nil
}

// PrintSpecificTable prints cards from a specific table based on user choice
func (c *Client) PrintSpecificTable(choice string) error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	switch choice {
	case "1":
		// Eternal Artisan Exception
		cards, err := queries.GetEternalArtisanCards(ctx)
		if err != nil {
			return fmt.Errorf("error getting eternal artisan cards: %v", err)
		}
		if len(cards) == 0 {
			fmt.Println("No eternal artisan exception cards found.")
			return nil
		}
		fmt.Printf("\n=== Eternal Artisan Exception (%d cards) ===\n", len(cards))
		for _, card := range cards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}
			fmt.Printf("- %s [%s] (%s) - %s\n", card.Name, manaCost, card.OracleID, card.TypeLine)
		}

	case "2":
		// Arena Only EA
		cards, err := queries.GetArenaOnlyEACards(ctx)
		if err != nil {
			return fmt.Errorf("error getting arena only EA cards: %v", err)
		}
		if len(cards) == 0 {
			fmt.Println("No arena only EA cards found.")
			return nil
		}
		fmt.Printf("\n=== Arena Only EA (%d cards) ===\n", len(cards))
		for _, card := range cards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}
			fmt.Printf("- %s [%s] (%s) - %s\n", card.Name, manaCost, card.OracleID, card.TypeLine)
		}

	case "3":
		// Banned Cards
		cards, err := queries.GetBannedCards(ctx)
		if err != nil {
			return fmt.Errorf("error getting banned cards: %v", err)
		}
		if len(cards) == 0 {
			fmt.Println("No banned cards found.")
			return nil
		}
		fmt.Printf("\n=== Banned Cards (%d cards) ===\n", len(cards))
		for _, card := range cards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}
			fmt.Printf("- %s [%s] (%s) - %s\n", card.Name, manaCost, card.OracleID, card.TypeLine)
		}

	case "4":
		// Watchlist
		cards, err := queries.GetWatchlistCards(ctx)
		if err != nil {
			return fmt.Errorf("error getting watchlist cards: %v", err)
		}
		if len(cards) == 0 {
			fmt.Println("No watchlist cards found.")
			return nil
		}
		fmt.Printf("\n=== Watchlist (%d cards) ===\n", len(cards))
		for _, card := range cards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}
			fmt.Printf("- %s [%s] (%s) - %s\n", card.Name, manaCost, card.OracleID, card.TypeLine)
		}

	case "5":
		// Digital Mechanic Cards
		cards, err := queries.GetDigitalMechanicCards(ctx)
		if err != nil {
			return fmt.Errorf("error getting digital mechanic cards: %v", err)
		}
		if len(cards) == 0 {
			fmt.Println("No digital mechanic cards found.")
			return nil
		}
		fmt.Printf("\n=== Digital Mechanic Cards (%d cards) ===\n", len(cards))
		for _, card := range cards {
			manaCost := "no mana cost"
			if card.ManaCost.Valid {
				manaCost = card.ManaCost.String
			}
			mechanicInfo := ""
			if card.MechanicKeyword.Valid {
				mechanicInfo = fmt.Sprintf(" [%s]", card.MechanicKeyword.String)
			}
			fmt.Printf("- %s [%s] (%s)%s - %s\n", card.Name, manaCost, card.OracleID, mechanicInfo, card.TypeLine)
		}

	case "6":
		// All Tables
		return c.GetAllCategorizedCards()

	default:
		fmt.Println("Invalid choice.")
	}

	return nil
}

// AddEOSCards fetches EOS cards that were once common/uncommon and adds them with arena game designation
func (c *Client) AddEOSCards() error {
	ctx := context.Background()
	queries := scryfall.New(c.db)

	// Search for EOS cards that have common/uncommon printings in other sets
	searchQuery := "set:eos (in:common or in:uncommon)"

	fmt.Printf("Searching for EOS cards with common/uncommon printings: %s\n", searchQuery)

	results, err := c.searchCards(searchQuery)
	if err != nil {
		return fmt.Errorf("error searching for EOS cards: %v", err)
	}

	if results.TotalCards == 0 {
		fmt.Println("No EOS cards found with common/uncommon printings.")
		return nil
	}

	fmt.Printf("Found %d EOS cards with common/uncommon printings:\n", results.TotalCards)

	insertedCount := 0
	for _, card := range results.Data {
		fmt.Printf("- %s\n", card.Name)

		// First, insert the card (oracle-level data) - this will be upserted if it already exists
		err := queries.UpsertCard(ctx, scryfall.UpsertCardParams{
			OracleID:        *card.OracleID,
			Name:            card.Name,
			Layout:          card.Layout,
			PrintsSearchUri: card.PrintsSearchURI.String(),
			RulingsUri:      card.RulingsURI.String(),
			AllParts:        toJSONString(card.AllParts),
			CardFaces:       toJSONString(card.CardFaces),
			Cmc:             card.CMC,
			ColorIdentity:   toJSONStringDirect(card.ColorIdentity),
			ColorIndicator:  toJSONString(card.ColorIndicator),
			Colors:          toJSONString(card.Colors),
			Defense:         ptrToNullString(card.Defense),
			EdhrecRank:      ptrToNullInt64(card.EDHRecRank),
			GameChanger:     ptrToNullBool(card.GameChanger),
			HandModifier:    ptrToNullString(card.HandModifier),
			Keywords:        toJSONStringDirect(card.Keywords),
			Legalities:      toJSONStringDirect(card.Legalities),
			LifeModifier:    ptrToNullString(card.LifeModifier),
			Loyalty:         ptrToNullString(card.Loyalty),
			ManaCost:        ptrToNullString(card.ManaCost),
			OracleText:      ptrToNullString(card.OracleText),
			PennyRank:       ptrToNullInt64(card.PennyRank),
			Power:           ptrToNullString(card.Power),
			ProducedMana:    toJSONString(card.ProducedMana),
			Reserved:        card.Reserved,
			Toughness:       ptrToNullString(card.Toughness),
			TypeLine:        card.TypeLine,
		})
		if err != nil {
			fmt.Printf("Error upserting card %s: %v\n", card.Name, err)
			continue
		}

		// Get all printings for this card
		printings, err := c.FetchAllPrintings(&card)
		if err != nil {
			fmt.Printf("Error fetching printings for %s: %v\n", card.Name, err)
			continue
		}

		// Add all printings, but hardcode arena for EOS printings
		for _, printing := range printings {
			var gamesString string
			if printing.Set == "eos" {
				// Hardcode arena into the games array for EOS printings
				gamesWithArena := []string{"arena", "paper", "mtgo"}
				gamesJSON, _ := json.Marshal(gamesWithArena)
				gamesString = string(gamesJSON)
			} else {
				gamesString = toJSONStringDirect(printing.Games)
			}

			err := queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
				ID:                printing.ID,
				OracleID:          *printing.OracleID,
				ArenaID:           ptrToNullInt64(printing.ArenaID),
				Lang:              printing.Lang,
				MtgoID:            ptrToNullInt64(printing.MTGOID),
				MtgoFoilID:        ptrToNullInt64(printing.MTGOFoilID),
				MultiverseIds:     toJSONString(printing.MultiverseIDs),
				TcgplayerID:       ptrToNullInt64(printing.TCGPlayerID),
				TcgplayerEtchedID: ptrToNullInt64(printing.TCGPlayerEtchedID),
				CardmarketID:      ptrToNullInt64(printing.CardmarketID),
				Object:            printing.Object,
				ScryfallUri:       printing.ScryfallURI.String(),
				Uri:               printing.URI.String(),
				Artist:            ptrToNullString(printing.Artist),
				ArtistIds:         toJSONString(printing.ArtistIDs),
				AttractionLights:  toJSONString(printing.AttractionLights),
				Booster:           printing.Booster,
				BorderColor:       printing.BorderColor,
				CardBackID:        printing.CardBackID,
				CollectorNumber:   printing.CollectorNumber,
				ContentWarning:    ptrToNullBool(printing.ContentWarning),
				Digital:           printing.Digital,
				Finishes:          toJSONStringDirect(printing.Finishes),
				FlavorName:        ptrToNullString(printing.FlavorName),
				FlavorText:        ptrToNullString(printing.FlavorText),
				Foil:              containsFinish(printing.Finishes, "foil"),
				Nonfoil:           containsFinish(printing.Finishes, "nonfoil"),
				FrameEffects:      toJSONString(printing.FrameEffects),
				Frame:             printing.Frame,
				FullArt:           printing.FullArt,
				Games:             gamesString, // Hardcoded with arena for EOS
				HighresImage:      printing.HighresImage,
				IllustrationID:    ptrToNullString(printing.IllustrationID),
				ImageStatus:       printing.ImageStatus,
				ImageUris:         toJSONString(printing.ImageURIs),
				Oversized:         printing.Oversized,
				Prices:            toJSONStringDirect(printing.Prices),
				PrintedName:       ptrToNullString(printing.PrintedName),
				PrintedText:       ptrToNullString(printing.PrintedText),
				PrintedTypeLine:   ptrToNullString(printing.PrintedTypeLine),
				Promo:             printing.Promo,
				PromoTypes:        toJSONString(printing.PromoTypes),
				PurchaseUris:      toJSONString(printing.PurchaseURIs),
				Rarity:            printing.Rarity,
				RelatedUris:       toJSONStringDirect(printing.RelatedURIs),
				ReleasedAt:        printing.ReleasedAt,
				Reprint:           printing.Reprint,
				ScryfallSetUri:    printing.ScryfallSetURI.String(),
				SetName:           printing.SetName,
				SetSearchUri:      printing.SetSearchURI.String(),
				SetType:           printing.SetType,
				SetUri:            printing.SetURI.String(),
				Set:               printing.Set,
				StorySpotlight:    printing.StorySpotlight,
				Textless:          printing.Textless,
				Variation:         printing.Variation,
				VariationOf:       ptrToNullString(printing.VariationOf),
				SecurityStamp:     ptrToNullString(printing.SecurityStamp),
				Watermark:         ptrToNullString(printing.Watermark),
				Preview:           toJSONString(printing.Preview),
			})
			if err != nil {
				fmt.Printf("Error upserting printing for %s: %v\n", card.Name, err)
			}
		}

		// Add to eternal_artisan_exception table so it shows up in legal cards
		err = queries.AddEternalArtisanException(ctx, *card.OracleID)
		if err != nil {
			fmt.Printf("Error adding to eternal artisan exception %s: %v\n", card.Name, err)
		}

		insertedCount++
	}

	fmt.Printf("Successfully processed %d EOS cards\n", insertedCount)
	return nil
}
