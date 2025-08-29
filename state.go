package scryball

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/ninesl/scryball/internal/client"
	"github.com/ninesl/scryball/internal/scryfall"
	_ "modernc.org/sqlite"
)

var (
	// Global singleton state
	CurrentScryball *Scryball
	initOnce        sync.Once
	mu              sync.RWMutex

	baseClientOptions = client.ClientOptions{
		APIURL:    "https://api.scryfall.com",
		UserAgent: "MTGScryball/1.0",
		Accept:    "application/json;q=0.9,*/*;q=0.8",
		Client:    &http.Client{},
	}
)

func ensureCurrentScryball() (*Scryball, error) {
	var topError error
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		if CurrentScryball == nil {
			newInstance, err := createDefaultInstance()
			if err != nil {
				topError = err
				return
			}
			CurrentScryball = newInstance
		}
	})
	if topError != nil {
		return nil, topError
	}
	return CurrentScryball, nil
}

func createDefaultInstance() (*Scryball, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Printf("Failed to create in-memory database: %v\n", err)
		return nil, err
	}

	if _, err := db.Exec(embeddedSchema); err != nil {
		db.Close()
		fmt.Printf("Failed to apply embedded schema: %v\n", err)
		return nil, err
	}

	scryballDB := &ScryballDB{DB: db}
	queries := scryfall.New(db)

	cClient, err := client.NewClientWithOptions(baseClientOptions)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return nil, err
	}

	return &Scryball{
		db:      scryballDB,
		client:  cClient,
		queries: queries,
	}, nil
}

func convertAPICardToDBParams(card *client.Card) (scryfall.UpsertCardParams, scryfall.UpsertPrintingParams, error) {
	derefString := func(s *string) string {
		if s != nil {
			return *s
		}
		return ""
	}

	derefInt := func(i *int) int64 {
		if i != nil {
			return int64(*i)
		}
		return 0
	}

	derefBool := func(b *bool) bool {
		if b != nil {
			return *b
		}
		return false
	}

	// Get oracle_id
	oracleID := derefString(card.OracleID)
	if oracleID == "" {
		return scryfall.UpsertCardParams{}, scryfall.UpsertPrintingParams{},
			fmt.Errorf("card %s has no oracle_id", card.Name)
	}

	allPartsJSON, _ := json.Marshal(card.AllParts)
	cardFacesJSON, _ := json.Marshal(card.CardFaces)
	colorIdentityJSON, _ := json.Marshal(card.ColorIdentity)
	colorIndicatorJSON, _ := json.Marshal(card.ColorIndicator)
	colorsJSON, _ := json.Marshal(card.Colors)
	keywordsJSON, _ := json.Marshal(card.Keywords)
	legalitiesJSON, _ := json.Marshal(card.Legalities)
	producedManaJSON, _ := json.Marshal(card.ProducedMana)

	cardParams := scryfall.UpsertCardParams{
		OracleID:        oracleID,
		Name:            card.Name,
		Layout:          card.Layout,
		PrintsSearchUri: card.PrintsSearchURI.String(),
		RulingsUri:      card.RulingsURI.String(),
		AllParts:        sql.NullString{String: string(allPartsJSON), Valid: len(allPartsJSON) > 2},
		CardFaces:       sql.NullString{String: string(cardFacesJSON), Valid: len(cardFacesJSON) > 2},
		Cmc:             card.CMC,
		ColorIdentity:   string(colorIdentityJSON),
		ColorIndicator:  sql.NullString{String: string(colorIndicatorJSON), Valid: len(colorIndicatorJSON) > 2},
		Colors:          sql.NullString{String: string(colorsJSON), Valid: len(colorsJSON) > 2},
		Defense:         sql.NullString{String: derefString(card.Defense), Valid: card.Defense != nil},
		EdhrecRank:      sql.NullInt64{Int64: derefInt(card.EDHRecRank), Valid: card.EDHRecRank != nil},
		GameChanger:     sql.NullBool{Bool: derefBool(card.GameChanger), Valid: card.GameChanger != nil},
		HandModifier:    sql.NullString{String: derefString(card.HandModifier), Valid: card.HandModifier != nil},
		Keywords:        string(keywordsJSON),
		Legalities:      string(legalitiesJSON),
		LifeModifier:    sql.NullString{String: derefString(card.LifeModifier), Valid: card.LifeModifier != nil},
		Loyalty:         sql.NullString{String: derefString(card.Loyalty), Valid: card.Loyalty != nil},
		ManaCost:        sql.NullString{String: derefString(card.ManaCost), Valid: card.ManaCost != nil},
		OracleText:      sql.NullString{String: derefString(card.OracleText), Valid: card.OracleText != nil},
		PennyRank:       sql.NullInt64{Int64: derefInt(card.PennyRank), Valid: card.PennyRank != nil},
		Power:           sql.NullString{String: derefString(card.Power), Valid: card.Power != nil},
		ProducedMana:    sql.NullString{String: string(producedManaJSON), Valid: len(producedManaJSON) > 2},
		Reserved:        card.Reserved,
		Toughness:       sql.NullString{String: derefString(card.Toughness), Valid: card.Toughness != nil},
		TypeLine:        card.TypeLine,
	}

	multiverseIDsJSON, _ := json.Marshal(card.MultiverseIDs)
	artistIDsJSON, _ := json.Marshal(card.ArtistIDs)
	attractionLightsJSON, _ := json.Marshal(card.AttractionLights)
	finishesJSON, _ := json.Marshal(card.Finishes)
	frameEffectsJSON, _ := json.Marshal(card.FrameEffects)
	gamesJSON, _ := json.Marshal(card.Games)
	imageUrisJSON, _ := json.Marshal(card.ImageURIs)
	pricesJSON, _ := json.Marshal(card.Prices)
	promoTypesJSON, _ := json.Marshal(card.PromoTypes)
	purchaseUrisJSON, _ := json.Marshal(card.PurchaseURIs)
	relatedUrisJSON, _ := json.Marshal(card.RelatedURIs)
	previewJSON, _ := json.Marshal(card.Preview)

	containsFinish := func(finishes []string, finish string) bool {
		for _, f := range finishes {
			if f == finish {
				return true
			}
		}
		return false
	}

	printingParams := scryfall.UpsertPrintingParams{
		ID:                card.ID,
		OracleID:          oracleID,
		ArenaID:           sql.NullInt64{Int64: derefInt(card.ArenaID), Valid: card.ArenaID != nil},
		Lang:              card.Lang,
		MtgoID:            sql.NullInt64{Int64: derefInt(card.MTGOID), Valid: card.MTGOID != nil},
		MtgoFoilID:        sql.NullInt64{Int64: derefInt(card.MTGOFoilID), Valid: card.MTGOFoilID != nil},
		MultiverseIds:     sql.NullString{String: string(multiverseIDsJSON), Valid: len(multiverseIDsJSON) > 2},
		TcgplayerID:       sql.NullInt64{Int64: derefInt(card.TCGPlayerID), Valid: card.TCGPlayerID != nil},
		TcgplayerEtchedID: sql.NullInt64{Int64: derefInt(card.TCGPlayerEtchedID), Valid: card.TCGPlayerEtchedID != nil},
		CardmarketID:      sql.NullInt64{Int64: derefInt(card.CardmarketID), Valid: card.CardmarketID != nil},
		Object:            card.Object,
		ScryfallUri:       card.ScryfallURI.String(),
		Uri:               card.URI.String(),
		Artist:            sql.NullString{String: derefString(card.Artist), Valid: card.Artist != nil},
		ArtistIds:         sql.NullString{String: string(artistIDsJSON), Valid: len(artistIDsJSON) > 2},
		AttractionLights:  sql.NullString{String: string(attractionLightsJSON), Valid: len(attractionLightsJSON) > 2},
		Booster:           card.Booster,
		BorderColor:       card.BorderColor,
		CardBackID:        card.CardBackID,
		CollectorNumber:   card.CollectorNumber,
		ContentWarning:    sql.NullBool{Bool: derefBool(card.ContentWarning), Valid: card.ContentWarning != nil},
		Digital:           card.Digital,
		Finishes:          string(finishesJSON),
		FlavorName:        sql.NullString{String: derefString(card.FlavorName), Valid: card.FlavorName != nil},
		FlavorText:        sql.NullString{String: derefString(card.FlavorText), Valid: card.FlavorText != nil},
		Foil:              containsFinish(card.Finishes, "foil"),
		Nonfoil:           containsFinish(card.Finishes, "nonfoil"),
		FrameEffects:      sql.NullString{String: string(frameEffectsJSON), Valid: len(frameEffectsJSON) > 2},
		Frame:             card.Frame,
		FullArt:           card.FullArt,
		Games:             string(gamesJSON),
		HighresImage:      card.HighresImage,
		IllustrationID:    sql.NullString{String: derefString(card.IllustrationID), Valid: card.IllustrationID != nil},
		ImageStatus:       card.ImageStatus,
		ImageUris:         sql.NullString{String: string(imageUrisJSON), Valid: len(imageUrisJSON) > 2},
		Oversized:         card.Oversized,
		Prices:            string(pricesJSON),
		PrintedName:       sql.NullString{String: derefString(card.PrintedName), Valid: card.PrintedName != nil},
		PrintedText:       sql.NullString{String: derefString(card.PrintedText), Valid: card.PrintedText != nil},
		PrintedTypeLine:   sql.NullString{String: derefString(card.PrintedTypeLine), Valid: card.PrintedTypeLine != nil},
		Promo:             card.Promo,
		PromoTypes:        sql.NullString{String: string(promoTypesJSON), Valid: len(promoTypesJSON) > 2},
		PurchaseUris:      sql.NullString{String: string(purchaseUrisJSON), Valid: len(purchaseUrisJSON) > 2},
		Rarity:            card.Rarity,
		RelatedUris:       string(relatedUrisJSON),
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
		VariationOf:       sql.NullString{String: derefString(card.VariationOf), Valid: card.VariationOf != nil},
		SecurityStamp:     sql.NullString{String: derefString(card.SecurityStamp), Valid: card.SecurityStamp != nil},
		Watermark:         sql.NullString{String: derefString(card.Watermark), Valid: card.Watermark != nil},
		Preview:           sql.NullString{String: string(previewJSON), Valid: len(previewJSON) > 2},
	}

	return cardParams, printingParams, nil
}
