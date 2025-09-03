package client

import (
	"encoding/json"
	"net/url"
)

// A List object represents a requested sequence of other objects (Cards, Sets, etc).
//
// List objects may be paginated, and also include information about
// issues raised when generating the list.
type List struct {
	//A content type for this object, always
	//  `list`
	Object string `json:"object"`

	//An array of the requested objects, in a specific order.
	Data []Card `json:"data"`

	//True if this List is paginated and there is a page beyond the current page.
	HasMore bool `json:"has_more"`

	// If this is a list of Card objects, this field will contain the
	// total number of cards found across all pages.
	//NULLABLE
	NextPage *url.URL `json:"next_page"`

	//If this is a list of Card objects, this field will contain
	// the total number of cards found across all pages.
	TotalCards int `json:"total_cards"`

	//An array of human-readable warnings issued when generating
	// this list, as strings. Warnings are non-fatal issues
	// that the API discovered with your input.
	// In general, they indicate that the List will not contain
	// the all of the information you requested. You should fix
	// the warnings and re-submit your request.
	//NULLABLE
	Warnings []string `json:"warnings"`
}
type SetType string

const (
	Core            SetType = "core"             // A yearly Magic core set
	Expansion       SetType = "expansion"        // A rotational expansion set in a block
	Masters         SetType = "masters"          // A reprint set that contains no new cards (Modern Masters, etc)
	Alchemy         SetType = "alchemy"          // An Arena set designed for Alchemy
	Masterpiece     SetType = "masterpiece"      // Masterpiece Series premium foil cards
	Arsenal         SetType = "arsenal"          // A Commander-oriented gift set
	FromTheVault    SetType = "from_the_vault"   // From the Vault gift sets
	Spellbook       SetType = "spellbook"        // Spellbook series gift sets
	PremiumDeck     SetType = "premium_deck"     // Premium Deck Series decks
	DuelDeck        SetType = "duel_deck"        // Duel Decks
	DraftInnovation SetType = "draft_innovation" // Special draft sets, like Conspiracy and Battlebond
	TreasureChest   SetType = "treasure_chest"   // Magic Online treasure chest prize sets
	Commander       SetType = "commander"        // Commander preconstructed decks
	Planechase      SetType = "planechase"       // Planechase sets
	Archenemy       SetType = "archenemy"        // Archenemy sets
	Vanguard        SetType = "vanguard"         // Vanguard card sets
	Funny           SetType = "funny"            // A funny un-set or set with funny promos (Unglued, Happy Holidays, etc)
	Starter         SetType = "starter"          // A starter/introductory set (Portal, etc)
	Box             SetType = "box"              // A gift box set
	Promo           SetType = "promo"            // A set that contains purely promotional cards
	Token           SetType = "token"            // A set made up of tokens and emblems
	Memorabilia     SetType = "memorabilia"      // A set made up of gold-bordered, oversize, or trophy cards that are not legal
	Minigame        SetType = "minigame"         // A set that contains minigame card inserts from booster packs
)

type Set struct {
	//A content type for this object, always "set"
	Object string `json:"object"`

	//A unique ID for this set on Scryfall that will not change
	ID string `json:"id"`

	//The unique three to six-letter code for this set
	Code string `json:"code"`

	//The unique code for this set on MTGO, which may differ from the regular code
	//NULLABLE
	MTGOCode *string `json:"mtgo_code"`

	//The unique code for this set on Arena, which may differ from the regular code
	//NULLABLE
	ArenaCode *string `json:"arena_code"`

	//This set's ID on TCGplayer's API, also known as the groupId
	//NULLABLE
	TCGPlayerID *int `json:"tcgplayer_id"`

	//The English name of the set
	Name string `json:"name"`

	//A computer-readable classification for this set
	SetType SetType `json:"set_type"`

	//The date the set was released or the first card was printed in the set
	//NULLABLE
	ReleasedAt *string `json:"released_at"`

	//The block code for this set, if any
	//NULLABLE
	BlockCode *string `json:"block_code"`

	//The block or group name code for this set, if any
	//NULLABLE
	Block *string `json:"block"`

	//The set code for the parent set, if any
	//NULLABLE
	ParentSetCode *string `json:"parent_set_code"`

	//The number of cards in this set
	CardCount int `json:"card_count"`

	//The denominator for the set's printed collector numbers
	//NULLABLE
	PrintedSize *int `json:"printed_size"`

	//True if this set was only released in a video game
	Digital bool `json:"digital"`

	//True if this set contains only foil cards
	FoilOnly bool `json:"foil_only"`

	//True if this set contains only nonfoil cards
	NonfoilOnly bool `json:"nonfoil_only"`

	//A link to this set's permapage on Scryfall's website
	ScryfallURI url.URL `json:"scryfall_uri"`

	//A link to this set object on Scryfall's API
	URI url.URL `json:"uri"`

	//A URI to an SVG file for this set's icon on Scryfall's CDN
	IconSVGURI url.URL `json:"icon_svg_uri"`

	//A Scryfall API URI that you can request to begin paginating over the cards in this set
	SearchURI url.URL `json:"search_uri"`
}

// Card objects represent individual Magic: The Gathering cards that players could obtain and add to their collection.
//
// Cards are the API's most complex object. You are encouraged to thoroughly read this document and also the article about layouts and images.
//
// Card Names:
// Internally, Scryfall tracks the uniqueness of "Oracle names." (i.e. names you can pick when an effect asks you to "choose a card name").
// Each unique Oracle name is separately available in the card names catalog.
// Note that while most Oracle card names are unique, Scryfall also indexes other objects such as tokens and Unstable set variants
// which do not always have a unique name.
//
// Multiface Cards:
// Magic cards can have multiple faces or multiple cards printed on one card stock. The faces could be shown divided on the front
// of the card as in split cards and flip cards, or the card can be double-faced as in transform cards and modal DFCs.
// Scryfall represents multi-face cards as a single object with a card_faces array describing the distinct faces.
type Card struct {
	// Core Card Fields

	// This card's Arena ID, if any. A large percentage of cards are not available on Arena and do not have this ID.
	// NULLABLE
	ArenaID *int `json:"arena_id"`

	// A unique ID for this card in Scryfall's database.
	ID string `json:"id"`

	// A language code for this printing.
	Lang string `json:"lang"`

	// This card's Magic Online ID (also known as the Catalog ID), if any. A large percentage of cards are not available on Magic Online and do not have this ID.
	// NULLABLE
	MTGOID *int `json:"mtgo_id"`

	// This card's foil Magic Online ID (also known as the Catalog ID), if any. A large percentage of cards are not available on Magic Online and do not have this ID.
	// NULLABLE
	MTGOFoilID *int `json:"mtgo_foil_id"`

	// This card's multiverse IDs on Gatherer, if any, as an array of integers. Note that Scryfall includes many promo cards, tokens, and other esoteric objects that do not have these identifiers.
	// NULLABLE
	MultiverseIDs []int `json:"multiverse_ids"`

	// This card's ID on TCGplayer's API, also known as the productId.
	// NULLABLE
	TCGPlayerID *int `json:"tcgplayer_id"`

	// This card's ID on TCGplayer's API, for its etched version if that version is a separate product.
	// NULLABLE
	TCGPlayerEtchedID *int `json:"tcgplayer_etched_id"`

	// This card's ID on Cardmarket's API, also known as the idProduct.
	// NULLABLE
	CardmarketID *int `json:"cardmarket_id"`

	// A content type for this object, always card.
	Object string `json:"object"`

	// A code for this card's layout.
	Layout string `json:"layout"`

	// A unique ID for this card's oracle identity. This value is consistent across reprinted card editions, and unique among different cards with the same name (tokens, Unstable variants, etc). Always present except for the reversible_card layout where it will be absent; oracle_id will be found on each face instead.
	// NULLABLE
	OracleID *string `json:"oracle_id"`

	// A link to where you can begin paginating all re/prints for this card on Scryfall's API.
	PrintsSearchURI url.URL `json:"prints_search_uri"`

	// A link to this card's rulings list on Scryfall's API.
	RulingsURI url.URL `json:"rulings_uri"`

	// A link to this card's permapage on Scryfall's website.
	ScryfallURI url.URL `json:"scryfall_uri"`

	// A link to this card object on Scryfall's API.
	URI url.URL `json:"uri"`

	// Gameplay Fields

	// If this card is closely related to other cards, this property will be an array with Related Card Objects.
	// NULLABLE
	AllParts []RelatedCard `json:"all_parts"`

	// An array of Card Face objects, if this card is multifaced.
	// NULLABLE
	CardFaces []CardFace `json:"card_faces"`

	// The card's mana value. Note that some funny cards have fractional mana costs.
	CMC float64 `json:"cmc"`

	// This card's color identity.
	ColorIdentity []string `json:"color_identity"`

	// The colors in this card's color indicator, if any. A null value for this field indicates the card does not have one.
	// NULLABLE
	ColorIndicator []string `json:"color_indicator"`

	// This card's colors, if the overall card has colors defined by the rules. Otherwise the colors will be on the card_faces objects.
	// NULLABLE
	Colors []string `json:"colors"`

	// This face's defense, if any.
	// NULLABLE
	Defense *string `json:"defense"`

	// This card's overall rank/popularity on EDHREC. Not all cards are ranked.
	// NULLABLE
	EDHRecRank *int `json:"edhrec_rank"`

	// True if this card is on the Commander Game Changer list.
	// NULLABLE
	GameChanger *bool `json:"game_changer"`

	// This card's hand modifier, if it is Vanguard card. This value will contain a delta, such as -1.
	// NULLABLE
	HandModifier *string `json:"hand_modifier"`

	// An array of keywords that this card uses, such as 'Flying' and 'Cumulative upkeep'.
	Keywords []string `json:"keywords"`

	// An object describing the legality of this card across play formats. Possible legalities are legal, not_legal, restricted, and banned.
	Legalities map[string]string `json:"legalities"`

	// This card's life modifier, if it is Vanguard card. This value will contain a delta, such as +2.
	// NULLABLE
	LifeModifier *string `json:"life_modifier"`

	// This loyalty if any. Note that some cards have loyalties that are not numeric, such as X.
	// NULLABLE
	Loyalty *string `json:"loyalty"`

	// The mana cost for this card. This value will be any empty string "" if the cost is absent. Remember that per the game rules, a missing mana cost and a mana cost of {0} are different values. Multi-faced cards will report this value in card faces.
	// NULLABLE
	ManaCost *string `json:"mana_cost"`

	// The name of this card. If this card has multiple faces, this field will contain both names separated by ␣//␣.
	Name string `json:"name"`

	// The Oracle text for this card, if any.
	// NULLABLE
	OracleText *string `json:"oracle_text"`

	// This card's rank/popularity on Penny Dreadful. Not all cards are ranked.
	// NULLABLE
	PennyRank *int `json:"penny_rank"`

	// This card's power, if any. Note that some cards have powers that are not numeric, such as *.
	// NULLABLE
	Power *string `json:"power"`

	// Colors of mana that this card could produce.
	// NULLABLE
	ProducedMana []string `json:"produced_mana"`

	// True if this card is on the Reserved List.
	Reserved bool `json:"reserved"`

	// This card's toughness, if any. Note that some cards have toughnesses that are not numeric, such as *.
	// NULLABLE
	Toughness *string `json:"toughness"`

	// The type line of this card.
	TypeLine string `json:"type_line"`

	// Print Fields - Properties unique to their particular re/print

	// The name of the illustrator of this card. Newly spoiled cards may not have this field yet.
	// NULLABLE
	Artist *string `json:"artist"`

	// The IDs of the artists that illustrated this card. Newly spoiled cards may not have this field yet.
	// NULLABLE
	ArtistIDs []string `json:"artist_ids"`

	// The lit Unfinity attractions lights on this card, if any.
	// NULLABLE
	AttractionLights []int `json:"attraction_lights"`

	// Whether this card is found in boosters.
	Booster bool `json:"booster"`

	// This card's border color: black, white, borderless, yellow, silver, or gold.
	BorderColor string `json:"border_color"`

	// The Scryfall ID for the card back design present on this card.
	CardBackID string `json:"card_back_id"`

	// This card's collector number. Note that collector numbers can contain non-numeric characters, such as letters or ★.
	CollectorNumber string `json:"collector_number"`

	// True if you should consider avoiding use of this print downstream.
	// NULLABLE
	ContentWarning *bool `json:"content_warning"`

	// True if this card was only released in a video game.
	Digital bool `json:"digital"`

	// An array of computer-readable flags that indicate if this card can come in foil, nonfoil, or etched finishes.
	Finishes []string `json:"finishes"`

	// The just-for-fun name printed on the card (such as for Godzilla series cards).
	// NULLABLE
	FlavorName *string `json:"flavor_name"`

	// The flavor text, if any.
	// NULLABLE
	FlavorText *string `json:"flavor_text"`

	// This card's frame effects, if any.
	// NULLABLE
	FrameEffects []string `json:"frame_effects"`

	// This card's frame layout.
	Frame string `json:"frame"`

	// True if this card's artwork is larger than normal.
	FullArt bool `json:"full_art"`

	// A list of games that this card print is available in, paper, arena, and/or mtgo.
	Games []string `json:"games"`

	// True if this card's imagery is high resolution.
	HighresImage bool `json:"highres_image"`

	// A unique identifier for the card artwork that remains consistent across reprints. Newly spoiled cards may not have this field yet.
	// NULLABLE
	IllustrationID *string `json:"illustration_id"`

	// A computer-readable indicator for the state of this card's image, one of missing, placeholder, lowres, or highres_scan.
	ImageStatus string `json:"image_status"`

	// An object listing available imagery for this card. See the Card Imagery article for more information.
	// NULLABLE
	ImageURIs map[string]string `json:"image_uris"`

	// True if this card is oversized.
	Oversized bool `json:"oversized"`

	// An object containing daily price information for this card, including usd, usd_foil, usd_etched, eur, eur_foil, eur_etched, and tix prices, as strings.
	Prices map[string]*string `json:"prices"`

	// The localized name printed on this card, if any.
	// NULLABLE
	PrintedName *string `json:"printed_name"`

	// The localized text printed on this card, if any.
	// NULLABLE
	PrintedText *string `json:"printed_text"`

	// The localized type line printed on this card, if any.
	// NULLABLE
	PrintedTypeLine *string `json:"printed_type_line"`

	// True if this card is a promotional print.
	Promo bool `json:"promo"`

	// An array of strings describing what categories of promo cards this card falls into.
	// NULLABLE
	PromoTypes []string `json:"promo_types"`

	// An object providing URIs to this card's listing on major marketplaces. Omitted if the card is unpurchaseable.
	// NULLABLE
	PurchaseURIs map[string]string `json:"purchase_uris"`

	// This card's rarity. One of common, uncommon, rare, special, mythic, or bonus.
	Rarity string `json:"rarity"`

	// An object providing URIs to this card's listing on other Magic: The Gathering online resources.
	RelatedURIs map[string]string `json:"related_uris"`

	// The date this card was first released.
	ReleasedAt string `json:"released_at"`

	// True if this card is a reprint.
	Reprint bool `json:"reprint"`

	// A link to this card's set on Scryfall's website.
	ScryfallSetURI url.URL `json:"scryfall_set_uri"`

	// This card's full set name.
	SetName string `json:"set_name"`

	// A link to where you can begin paginating this card's set on the Scryfall API.
	SetSearchURI url.URL `json:"set_search_uri"`

	// The type of set this printing is in.
	SetType string `json:"set_type"`

	// A link to this card's set object on Scryfall's API.
	SetURI url.URL `json:"set_uri"`

	// This card's set code.
	Set string `json:"set"`

	// This card's Set object UUID.
	SetID string `json:"set_id"`

	// True if this card is a Story Spotlight.
	StorySpotlight bool `json:"story_spotlight"`

	// True if the card is printed without text.
	Textless bool `json:"textless"`

	// Whether this card is a variation of another printing.
	Variation bool `json:"variation"`

	// The printing ID of the printing this card is a variation of.
	// NULLABLE
	VariationOf *string `json:"variation_of"`

	// The security stamp on this card, if any. One of oval, triangle, acorn, circle, arena, or heart.
	// NULLABLE
	SecurityStamp *string `json:"security_stamp"`

	// This card's watermark, if any.
	// NULLABLE
	Watermark *string `json:"watermark"`

	// Preview information containing previewed_at, source_uri, and source fields.
	Preview *CardPreview `json:"preview"`
}

// CardFace represents a single face of a multiface card.
// Multiface cards have a card_faces property containing at least two Card Face objects.
type CardFace struct {
	// The name of the illustrator of this card face. Newly spoiled cards may not have this field yet.
	// NULLABLE
	Artist *string `json:"artist"`

	// The ID of the illustrator of this card face. Newly spoiled cards may not have this field yet.
	// NULLABLE
	ArtistID *string `json:"artist_id"`

	// The mana value of this particular face, if the card is reversible.
	// NULLABLE
	CMC *float64 `json:"cmc"`

	// The colors in this face's color indicator, if any.
	// NULLABLE
	ColorIndicator []string `json:"color_indicator"`

	// This face's colors, if the game defines colors for the individual face of this card.
	// NULLABLE
	Colors []string `json:"colors"`

	// This face's defense, if any.
	// NULLABLE
	Defense *string `json:"defense"`

	// The flavor text printed on this face, if any.
	// NULLABLE
	FlavorText *string `json:"flavor_text"`

	// A unique identifier for the card face artwork that remains consistent across reprints. Newly spoiled cards may not have this field yet.
	// NULLABLE
	IllustrationID *string `json:"illustration_id"`

	// An object providing URIs to imagery for this face, if this is a double-sided card. If this card is not double-sided, then the image_uris property will be part of the parent object instead.
	// NULLABLE
	ImageURIs map[string]string `json:"image_uris"`

	// The layout of this card face, if the card is reversible.
	// NULLABLE
	Layout *string `json:"layout"`

	// This face's loyalty, if any.
	// NULLABLE
	Loyalty *string `json:"loyalty"`

	// The mana cost for this face. This value will be any empty string "" if the cost is absent. Remember that per the game rules, a missing mana cost and a mana cost of {0} are different values.
	ManaCost string `json:"mana_cost"`

	// The name of this particular face.
	Name string `json:"name"`

	// A content type for this object, always card_face.
	Object string `json:"object"`

	// The Oracle ID of this particular face, if the card is reversible.
	// NULLABLE
	OracleID *string `json:"oracle_id"`

	// The Oracle text for this face, if any.
	// NULLABLE
	OracleText *string `json:"oracle_text"`

	// This face's power, if any. Note that some cards have powers that are not numeric, such as *.
	// NULLABLE
	Power *string `json:"power"`

	// The localized name printed on this face, if any.
	// NULLABLE
	PrintedName *string `json:"printed_name"`

	// The localized text printed on this face, if any.
	// NULLABLE
	PrintedText *string `json:"printed_text"`

	// The localized type line printed on this face, if any.
	// NULLABLE
	PrintedTypeLine *string `json:"printed_type_line"`

	// This face's toughness, if any.
	// NULLABLE
	Toughness *string `json:"toughness"`

	// The type line of this particular face, if the card is reversible.
	// NULLABLE
	TypeLine *string `json:"type_line"`

	// The watermark on this particulary card face, if any.
	// NULLABLE
	Watermark *string `json:"watermark"`
}

// RelatedCard represents cards that are closely related to other cards.
// Cards that are closely related to other cards (because they call them by name, or generate a token, or meld, etc) have an all_parts property that contains Related Card objects.
type RelatedCard struct {
	// An unique ID for this card in Scryfall's database.
	ID string `json:"id"`

	// A content type for this object, always related_card.
	Object string `json:"object"`

	// A field explaining what role this card plays in this relationship, one of token, meld_part, meld_result, or combo_piece.
	Component string `json:"component"`

	// The name of this particular related card.
	Name string `json:"name"`

	// The type line of this card.
	TypeLine string `json:"type_line"`

	// A URI where you can retrieve a full object describing this card on Scryfall's API.
	URI url.URL `json:"uri"`
}

// CardPreview contains preview information for cards.
type CardPreview struct {
	// The date this card was previewed.
	// NULLABLE
	PreviewedAt *string `json:"previewed_at"`

	// A link to the preview for this card.
	// NULLABLE
	SourceURI *url.URL `json:"source_uri"`

	// The name of the source that previewed this card.
	// NULLABLE
	Source *string `json:"source"`
}

// UnmarshalJSON implements custom unmarshalling for List to handle URL fields. Used internally
func (l *List) UnmarshalJSON(data []byte) error {
	type Alias List
	aux := &struct {
		NextPage *string `json:"next_page"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.NextPage != nil {
		parsed, err := url.Parse(*aux.NextPage)
		if err != nil {
			return err
		}
		l.NextPage = parsed
	}

	return nil
}

// UnmarshalJSON implements custom unmarshalling for Set to handle URL fields
func (s *Set) UnmarshalJSON(data []byte) error {
	type Alias Set
	aux := &struct {
		ScryfallURI string `json:"scryfall_uri"`
		URI         string `json:"uri"`
		IconSVGURI  string `json:"icon_svg_uri"`
		SearchURI   string `json:"search_uri"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	var parsed *url.URL
	if parsed, err = url.Parse(aux.ScryfallURI); err != nil {
		return err
	}
	s.ScryfallURI = *parsed

	if parsed, err = url.Parse(aux.URI); err != nil {
		return err
	}
	s.URI = *parsed

	if parsed, err = url.Parse(aux.IconSVGURI); err != nil {
		return err
	}
	s.IconSVGURI = *parsed

	if parsed, err = url.Parse(aux.SearchURI); err != nil {
		return err
	}
	s.SearchURI = *parsed

	return nil
}

// UnmarshalJSON implements custom unmarshalling for Card to handle URL fields
func (c *Card) UnmarshalJSON(data []byte) error {
	type Alias Card
	aux := &struct {
		PrintsSearchURI string `json:"prints_search_uri"`
		RulingsURI      string `json:"rulings_uri"`
		ScryfallURI     string `json:"scryfall_uri"`
		URI             string `json:"uri"`
		ScryfallSetURI  string `json:"scryfall_set_uri"`
		SetSearchURI    string `json:"set_search_uri"`
		SetURI          string `json:"set_uri"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	var parsed *url.URL

	if parsed, err = url.Parse(aux.PrintsSearchURI); err != nil {
		return err
	}
	c.PrintsSearchURI = *parsed

	if parsed, err = url.Parse(aux.RulingsURI); err != nil {
		return err
	}
	c.RulingsURI = *parsed

	if parsed, err = url.Parse(aux.ScryfallURI); err != nil {
		return err
	}
	c.ScryfallURI = *parsed

	if parsed, err = url.Parse(aux.URI); err != nil {
		return err
	}
	c.URI = *parsed

	if parsed, err = url.Parse(aux.ScryfallSetURI); err != nil {
		return err
	}
	c.ScryfallSetURI = *parsed

	if parsed, err = url.Parse(aux.SetSearchURI); err != nil {
		return err
	}
	c.SetSearchURI = *parsed

	if parsed, err = url.Parse(aux.SetURI); err != nil {
		return err
	}
	c.SetURI = *parsed

	return nil
}

// UnmarshalJSON implements custom unmarshalling for RelatedCard to handle URL fields
func (r *RelatedCard) UnmarshalJSON(data []byte) error {
	type Alias RelatedCard
	aux := &struct {
		URI string `json:"uri"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	var parsed *url.URL
	if parsed, err = url.Parse(aux.URI); err != nil {
		return err
	}
	r.URI = *parsed

	return nil
}

// UnmarshalJSON implements custom unmarshalling for CardPreview to handle URL fields
func (p *CardPreview) UnmarshalJSON(data []byte) error {
	type Alias CardPreview
	aux := &struct {
		SourceURI *string `json:"source_uri"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.SourceURI != nil {
		parsed, err := url.Parse(*aux.SourceURI)
		if err != nil {
			return err
		}
		p.SourceURI = parsed
	}

	return nil
}
