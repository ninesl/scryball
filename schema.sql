-- Normalized schema with Cards (oracle-level) and Printings (printing-level) tables

-- Cards table: One row per unique card (oracle_id level)
CREATE TABLE IF NOT EXISTS cards (
    oracle_id TEXT PRIMARY KEY NOT NULL, -- Shared across all printings of the same card
    name TEXT NOT NULL,
    layout TEXT NOT NULL,
    prints_search_uri TEXT NOT NULL,
    rulings_uri TEXT NOT NULL,
    
    -- Gameplay Fields (oracle-level, same across all printings)
    all_parts TEXT, -- JSON array of RelatedCard objects
    card_faces TEXT, -- JSON array of CardFace objects
    cmc REAL NOT NULL,
    color_identity TEXT NOT NULL, -- JSON array of strings
    color_indicator TEXT, -- JSON array of strings
    colors TEXT, -- JSON array of strings
    defense TEXT,
    edhrec_rank INTEGER,
    game_changer BOOLEAN,
    hand_modifier TEXT,
    keywords TEXT NOT NULL, -- JSON array of strings
    legalities TEXT NOT NULL, -- JSON object map[string]string
    life_modifier TEXT,
    loyalty TEXT,
    mana_cost TEXT,
    oracle_text TEXT,
    penny_rank INTEGER,
    power TEXT,
    produced_mana TEXT, -- JSON array of strings
    reserved BOOLEAN NOT NULL,
    toughness TEXT,
    type_line TEXT NOT NULL
);

-- Printings table: Multiple rows per card for different printings
CREATE TABLE IF NOT EXISTS printings (
    id TEXT PRIMARY KEY NOT NULL, -- Unique per printing
    oracle_id TEXT NOT NULL, -- Foreign key to cards table
    
    -- Core Print Fields
    arena_id INTEGER,
    lang TEXT NOT NULL,
    mtgo_id INTEGER,
    mtgo_foil_id INTEGER,
    multiverse_ids TEXT, -- JSON array of integers
    tcgplayer_id INTEGER,
    tcgplayer_etched_id INTEGER,
    cardmarket_id INTEGER,
    object TEXT NOT NULL,
    scryfall_uri TEXT NOT NULL,
    uri TEXT NOT NULL,
    
    -- Print-specific Fields
    artist TEXT,
    artist_ids TEXT, -- JSON array of strings
    attraction_lights TEXT, -- JSON array of integers
    booster BOOLEAN NOT NULL,
    border_color TEXT NOT NULL,
    card_back_id TEXT NOT NULL,
    collector_number TEXT NOT NULL,
    content_warning BOOLEAN,
    digital BOOLEAN NOT NULL,
    finishes TEXT NOT NULL, -- JSON array of strings
    flavor_name TEXT,
    flavor_text TEXT,
    foil BOOLEAN NOT NULL,
    nonfoil BOOLEAN NOT NULL,
    frame_effects TEXT, -- JSON array of strings
    frame TEXT NOT NULL,
    full_art BOOLEAN NOT NULL,
    games TEXT NOT NULL, -- JSON array of strings
    highres_image BOOLEAN NOT NULL,
    illustration_id TEXT,
    image_status TEXT NOT NULL,
    image_uris TEXT, -- JSON object map[string]string
    oversized BOOLEAN NOT NULL,
    prices TEXT NOT NULL, -- JSON object map[string]*string
    printed_name TEXT,
    printed_text TEXT,
    printed_type_line TEXT,
    promo BOOLEAN NOT NULL,
    promo_types TEXT, -- JSON array of strings
    purchase_uris TEXT, -- JSON object map[string]string
    rarity TEXT NOT NULL,
    related_uris TEXT NOT NULL, -- JSON object map[string]string
    released_at TEXT NOT NULL,
    reprint BOOLEAN NOT NULL,
    scryfall_set_uri TEXT NOT NULL,
    set_name TEXT NOT NULL,
    set_search_uri TEXT NOT NULL,
    set_type TEXT NOT NULL,
    set_uri TEXT NOT NULL,
    "set" TEXT NOT NULL, -- This is the "set" field in the API
    set_id TEXT NOT NULL,
    story_spotlight BOOLEAN NOT NULL,
    textless BOOLEAN NOT NULL,
    variation BOOLEAN NOT NULL,
    variation_of TEXT,
    security_stamp TEXT,
    watermark TEXT,
    preview TEXT, -- JSON object CardPreview
    
    -- Foreign key constraint
    FOREIGN KEY (oracle_id) REFERENCES cards(oracle_id)
);

-- Indexes for Cards table
CREATE INDEX IF NOT EXISTS idx_cards_name ON cards(name);

-- Indexes for Printings table
CREATE INDEX IF NOT EXISTS idx_printings_oracle_id ON printings(oracle_id);
CREATE INDEX IF NOT EXISTS idx_printings_set ON printings("set");
CREATE INDEX IF NOT EXISTS idx_printings_rarity ON printings(rarity);
CREATE INDEX IF NOT EXISTS idx_printings_games ON printings(games);

-- Query Cache table: Stores search queries and their results
CREATE TABLE IF NOT EXISTS query_cache (
    query_id INTEGER PRIMARY KEY AUTOINCREMENT,
    query_text TEXT NOT NULL UNIQUE, -- The exact search query string
    oracle_ids TEXT NOT NULL, -- JSON array of oracle_ids like ["id1", "id2", ...]
    cached_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_accessed TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hit_count INTEGER NOT NULL DEFAULT 1
);

-- Indexes for Query Cache table
CREATE INDEX IF NOT EXISTS idx_query_cache_query_text ON query_cache(query_text);
CREATE INDEX IF NOT EXISTS idx_query_cache_cached_at ON query_cache(cached_at);
CREATE INDEX IF NOT EXISTS idx_query_cache_last_accessed ON query_cache(last_accessed);
