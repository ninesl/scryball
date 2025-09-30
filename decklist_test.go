package scryball

import (
	"context"
	"strings"
	"testing"

	"github.com/ninesl/scryball/internal/client"
)

func TestParseArenaDecklist(t *testing.T) {
	// Simple test decklist
	decklistString := `Deck
4 Lightning Bolt
20 Mountain
1 Mountain

Sideboard
3 Pyroblast
1 Pyroblast
`

	deck, err := ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Failed to parse decklist: %v", err)
	}

	// Check maindeck
	if deck.NumberOfCards() != 25 {
		t.Errorf("Expected 25 maindeck cards, got %d", deck.NumberOfCards())
	}

	// Check sideboard
	if deck.NumberOfSideboardCards() != 4 {
		t.Errorf("Expected 4 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}

	// Check specific cards
	foundBolt := false
	foundMountain := false
	for card, qty := range deck.Maindeck {
		if card.Name == "Lightning Bolt" {
			foundBolt = true
			if qty != 4 {
				t.Errorf("Expected 4 Lightning Bolt, got %d", qty)
			}
		}
		if card.Name == "Mountain" {
			foundMountain = true
			if qty != 21 {
				t.Errorf("Expected 21 Mountain, got %d", qty)
			}
		}
	}

	if !foundBolt {
		t.Error("Lightning Bolt not found in maindeck")
	}
	if !foundMountain {
		t.Error("Mountain not found in maindeck")
	}
}

func TestParseArenaDecklist_WithSetCodes(t *testing.T) {
	// Decklist with set codes and collector numbers
	decklistString := `
4 Lightning Bolt (2ED) 161
4 Counterspell (ICE) 64
10 Island
10 Mountain

Sideboard
2 Pyroblast (ICE) 213
`

	deck, err := ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Failed to parse decklist with set codes: %v", err)
	}

	// Check totals
	if deck.NumberOfCards() != 28 {
		t.Errorf("Expected 28 maindeck cards, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 2 {
		t.Errorf("Expected 2 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}
}

func TestParseArenaDecklist_EmptyDecklist(t *testing.T) {
	deck, err := ParseDecklist("")
	if err != nil {
		t.Fatalf("Failed to parse empty decklist: %v", err)
	}

	if deck.NumberOfCards() != 0 {
		t.Errorf("Expected 0 maindeck cards for empty input, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 0 {
		t.Errorf("Expected 0 sideboard cards for empty input, got %d", deck.NumberOfSideboardCards())
	}
}

func TestParseArenaDecklist_SideboardLimit(t *testing.T) {
	// Test that sideboard is limited to 15 cards
	decklistString := `
4 Lightning Bolt

Sideboard
4 Pyroblast
4 Red Elemental Blast
4 Blood Moon
4 Alpine Moon
`

	_, err := ParseDecklist(decklistString)
	if err == nil {
		t.Error("Expected error for sideboard exceeding 15 cards")
	}
	if !strings.Contains(err.Error(), "exceeds 15 cards") {
		t.Errorf("Expected error about 15 card limit, got: %v", err)
	}
}

func TestValidateStandard(t *testing.T) {
	// Create a valid Standard deck
	validDeck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}

	// Add 60 cards to maindeck (using Mountains as placeholder)
	mountain := &MagicCard{
		Card: &client.Card{
			Name: "Mountain",
		},
	}
	validDeck.Maindeck[mountain] = 60

	err := validDeck.ValidateConstructed()
	if err != nil {
		t.Errorf("Valid 60 card deck failed validation: %v", err)
	}

	// Test deck with too few cards
	smallDeck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}
	smallDeck.Maindeck[mountain] = 59

	err = smallDeck.ValidateConstructed()
	if err == nil {
		t.Error("59 card deck should fail Standard validation")
	}
	if !strings.Contains(err.Error(), "minimum is 60") {
		t.Errorf("Expected minimum cards error, got: %v", err)
	}
}

func TestValidateLimited(t *testing.T) {
	// Create a valid Limited deck
	validDeck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}

	// Add 40 cards to maindeck
	mountain := &MagicCard{
		Card: &client.Card{
			Name: "Mountain",
		},
	}
	validDeck.Maindeck[mountain] = 40

	err := validDeck.ValidateLimited()
	if err != nil {
		t.Errorf("Valid 40 card deck failed validation: %v", err)
	}

	// Test deck with too few cards
	smallDeck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}
	smallDeck.Maindeck[mountain] = 39

	err = smallDeck.ValidateLimited()
	if err == nil {
		t.Error("39 card deck should fail Limited validation")
	}
}

func TestValidateDecklist_FourCopyRule(t *testing.T) {
	// Create a deck with enough cards to pass minimum count
	testDeck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}

	// Add 5 copies of a non-basic card (should fail 4-copy rule)
	bolt := &MagicCard{
		Card: &client.Card{
			Name: "Lightning Bolt",
		},
	}
	testDeck.Maindeck[bolt] = 5

	// Add enough basic lands to meet minimum card count
	mountain := &MagicCard{
		Card: &client.Card{
			Name: "Mountain",
		},
	}
	testDeck.Maindeck[mountain] = 55 // 5 bolts + 55 mountains = 60 cards

	err := testDeck.ValidateConstructed()
	if err == nil {
		t.Error("Deck with 5 Lightning Bolts should fail validation")
	}
	if !strings.Contains(err.Error(), "maximum is 4") {
		t.Errorf("Expected 4-copy rule error, got: %v", err)
	}

	// Test with valid card counts
	testDeck.Maindeck[bolt] = 4 // Fix bolt count to 4

	err = testDeck.ValidateConstructed()
	if err != nil && !strings.Contains(err.Error(), "minimum") {
		t.Errorf("Deck with 4 Bolts and 55 Mountains should be valid, got: %v", err)
	}
}

func TestIsBasicLand(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Plains", true},
		{"Island", true},
		{"Swamp", true},
		{"Mountain", true},
		{"Forest", true},
		{"Snow-Covered Plains", true},
		{"Snow-Covered Island", true},
		{"Snow-Covered Swamp", true},
		{"Snow-Covered Mountain", true},
		{"Snow-Covered Forest", true},
		{"Wastes", true},
		{"Lightning Bolt", false},
		{"Volcanic Island", false},
		{"Blood Crypt", false},
	}

	for _, tt := range tests {
		result := isBasicLandName(tt.name)
		if result != tt.expected {
			t.Errorf("isBasicLandName(%s) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestIsSpecialCard(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Relentless Rats", true},
		{"Shadowborn Apostle", true},
		{"Rat Colony", true},
		{"Persistent Petitioners", true},
		{"Dragon's Approach", true},
		{"Seven Dwarves", true},
		{"Nazgûl", true},
		{"Lightning Bolt", false},
		{"Mountain", false},
	}

	for _, tt := range tests {
		result := isSpecialCardName(tt.name)
		if result != tt.expected {
			t.Errorf("isSpecialCardName(%s) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestDecklistString(t *testing.T) {
	ctx := context.Background()

	// Create a simple deck
	deck := &Decklist{
		Maindeck:  make(map[*MagicCard]int),
		Sideboard: make(map[*MagicCard]int),
	}

	// Fetch real cards
	bolt, _ := QueryCardWithContext(ctx, "Lightning Bolt")
	mountain, _ := QueryCardWithContext(ctx, "Mountain")
	pyroblast, _ := QueryCardWithContext(ctx, "Pyroblast")

	if bolt != nil {
		deck.Maindeck[bolt] = 4
	}
	if mountain != nil {
		deck.Maindeck[mountain] = 20
	}
	if pyroblast != nil {
		deck.Sideboard[pyroblast] = 3
	}

	str := deck.String()

	// Check that output contains expected cards
	if bolt != nil && !strings.Contains(str, "4 Lightning Bolt") {
		t.Error("String output missing Lightning Bolt")
	}
	if mountain != nil && !strings.Contains(str, "20 Mountain") {
		t.Error("String output missing Mountain")
	}
	if pyroblast != nil && !strings.Contains(str, "3 Pyroblast") {
		t.Error("String output missing Pyroblast")
	}

	// Check sideboard header
	if pyroblast != nil && !strings.Contains(str, "Sideboard") {
		t.Error("String output missing Sideboard header")
	}
}

func TestParseCardLine(t *testing.T) {
	tests := []struct {
		input        string
		expectedQty  int
		expectedName string
		shouldError  bool
	}{
		{"4 Lightning Bolt", 4, "Lightning Bolt", false},
		{"1 Birds of Paradise", 1, "Birds of Paradise", false},
		{"4 Lightning Bolt (2ED) 161", 4, "Lightning Bolt", false},
		{"2 Counterspell (ICE) 64", 2, "Counterspell", false},
		{"20 Mountain", 20, "Mountain", false},
		{"Lightning Bolt", 0, "", true},              // No quantity
		{"4", 0, "", true},                           // No card name
		{"", 0, "", true},                            // Empty line
		{"not a number Lightning Bolt", 0, "", true}, // Invalid quantity
	}

	for _, tt := range tests {
		qty, name, err := parseCardLine(tt.input)

		if tt.shouldError {
			if err == nil {
				t.Errorf("parseCardLine(%s) expected error but got none", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseCardLine(%s) unexpected error: %v", tt.input, err)
			}
			if qty != tt.expectedQty {
				t.Errorf("parseCardLine(%s) qty = %d, expected %d", tt.input, qty, tt.expectedQty)
			}
			if name != tt.expectedName {
				t.Errorf("parseCardLine(%s) name = %s, expected %s", tt.input, name, tt.expectedName)
			}
		}
	}
}

// TestParseDecklist_Global tests the global ParseDecklist function
func TestParseDecklist_Global(t *testing.T) {
	decklistString := `4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast`

	deck, err := ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Global ParseDecklist failed: %v", err)
	}

	if deck.NumberOfCards() != 24 {
		t.Errorf("Expected 24 maindeck cards, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 3 {
		t.Errorf("Expected 3 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}
}

// TestParseDecklistWithContext_Global tests the global ParseDecklistWithContext function
func TestParseDecklistWithContext_Global(t *testing.T) {
	ctx := context.Background()
	decklistString := `4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast`

	deck, err := ParseDecklistWithContext(ctx, decklistString)
	if err != nil {
		t.Fatalf("Global ParseDecklistWithContext failed: %v", err)
	}

	if deck.NumberOfCards() != 24 {
		t.Errorf("Expected 24 maindeck cards, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 3 {
		t.Errorf("Expected 3 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}
}

// TestParseDecklist_Instance tests the instance ParseDecklist method
func TestParseDecklist_Instance(t *testing.T) {
	sb, err := NewWithConfig(ScryballConfig{})
	if err != nil {
		t.Fatalf("Failed to create Scryball instance: %v", err)
	}
	defer sb.db.Close()

	decklistString := `4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast`

	deck, err := sb.ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Instance ParseDecklist failed: %v", err)
	}

	if deck.NumberOfCards() != 24 {
		t.Errorf("Expected 24 maindeck cards, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 3 {
		t.Errorf("Expected 3 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}
}

// TestParseDecklistWithContext_Instance tests the instance ParseDecklistWithContext method
func TestParseDecklistWithContext_Instance(t *testing.T) {
	ctx := context.Background()

	sb, err := NewWithConfig(ScryballConfig{})
	if err != nil {
		t.Fatalf("Failed to create Scryball instance: %v", err)
	}
	defer sb.db.Close()

	decklistString := `4 Lightning Bolt
20 Mountain

Sideboard
3 Pyroblast`

	deck, err := sb.ParseDecklistWithContext(ctx, decklistString)
	if err != nil {
		t.Fatalf("Instance ParseDecklistWithContext failed: %v", err)
	}

	if deck.NumberOfCards() != 24 {
		t.Errorf("Expected 24 maindeck cards, got %d", deck.NumberOfCards())
	}

	if deck.NumberOfSideboardCards() != 3 {
		t.Errorf("Expected 3 sideboard cards, got %d", deck.NumberOfSideboardCards())
	}
}

// TestParseDecklist_InstanceIndependence tests that instances maintain independent caches
func TestParseDecklist_InstanceIndependence(t *testing.T) {
	sb1, err := NewWithConfig(ScryballConfig{})
	if err != nil {
		t.Fatalf("Failed to create Scryball instance 1: %v", err)
	}
	defer sb1.db.Close()

	sb2, err := NewWithConfig(ScryballConfig{})
	if err != nil {
		t.Fatalf("Failed to create Scryball instance 2: %v", err)
	}
	defer sb2.db.Close()

	decklistString := `4 Lightning Bolt`

	deck1, err := sb1.ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Instance 1 ParseDecklist failed: %v", err)
	}

	deck2, err := sb2.ParseDecklist(decklistString)
	if err != nil {
		t.Fatalf("Instance 2 ParseDecklist failed: %v", err)
	}

	if deck1.NumberOfCards() != 4 || deck2.NumberOfCards() != 4 {
		t.Errorf("Expected 4 cards each, got %d and %d", deck1.NumberOfCards(), deck2.NumberOfCards())
	}

	// Verify independent card objects
	var card1, card2 *MagicCard
	for card := range deck1.Maindeck {
		card1 = card
		break
	}
	for card := range deck2.Maindeck {
		card2 = card
		break
	}

	if card1 == nil || card2 == nil {
		t.Fatal("Could not find cards in decks")
	}

	if card1.Name != card2.Name {
		t.Errorf("Expected same card name, got %s vs %s", card1.Name, card2.Name)
	}

	// Different instances should have different card objects
	if card1 == card2 {
		t.Error("Expected different card instances for independent Scryball instances")
	}
}
