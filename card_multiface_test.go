package scryball

import (
	"context"
	"strings"
	"testing"
)

func TestImportAndFetchMultifaceCard(t *testing.T) {
	sb := testHelper(t)
	defer sb.db.Close()

	const bulk = `[{"object":"card","id":"d0d484a6-5610-4f1d-95ec-eda273c255e4","oracle_id":"727f3201-1cfc-4ab2-9dfe-be4f7251f42f","name":"Boggart Trawler // Boggart Bog","lang":"en","released_at":"2024-06-14","layout":"modal_dfc","cmc":3,"type_line":"Creature - Goblin // Land","color_identity":["B"],"keywords":[],"games":["arena"],"finishes":["nonfoil"],"set":"mh3","set_name":"Modern Horizons 3","rarity":"uncommon","card_faces":[{"object":"card_face","name":"Boggart Trawler","mana_cost":"{2}{B}","type_line":"Creature - Goblin","oracle_text":"When this creature enters, exile target player's graveyard.","colors":["B"],"power":"3","toughness":"1","image_uris":{"normal":"https://example.test/boggart-trawler.jpg","art_crop":"https://example.test/boggart-trawler-art.jpg"}},{"object":"card_face","name":"Boggart Bog","mana_cost":"","type_line":"Land","oracle_text":"This land enters tapped.","colors":[],"image_uris":{"normal":"https://example.test/boggart-bog.jpg"}}]}]`

	if _, err := sb.ImportCardsJSON(context.Background(), strings.NewReader(bulk)); err != nil {
		t.Fatalf("ImportCardsJSON() error = %v", err)
	}

	card, err := sb.FetchCardByExactOracleID(context.Background(), "727f3201-1cfc-4ab2-9dfe-be4f7251f42f")
	if err != nil {
		t.Fatalf("FetchCardByExactOracleID() error = %v", err)
	}
	if len(card.CardFaces) != 2 {
		t.Fatalf("len(CardFaces) = %d, want 2", len(card.CardFaces))
	}
	if card.CardFaces[0].OracleText == nil || *card.CardFaces[0].OracleText != "When this creature enters, exile target player's graveyard." {
		t.Fatalf("front face OracleText = %v", card.CardFaces[0].OracleText)
	}
	if len(card.Printings) != 1 {
		t.Fatalf("len(Printings) = %d, want 1", len(card.Printings))
	}
	if got := card.Printings[0].ImageURI; got != "https://example.test/boggart-trawler.jpg" {
		t.Fatalf("printing ImageURI = %q", got)
	}
	if got := card.Printings[0].ArtCropURI; got != "https://example.test/boggart-trawler-art.jpg" {
		t.Fatalf("printing ArtCropURI = %q", got)
	}
}
