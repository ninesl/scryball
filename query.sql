-- Get all cards with their printings
-- name: GetCardsWithPrintings :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at
FROM cards c
JOIN printings p ON c.oracle_id = p.oracle_id
ORDER BY c.name, p.released_at DESC;

-- Get eternal artisan exception cards with their printings, excluding banned cards
-- name: GetEternalArtisanCards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at
FROM cards c
JOIN printings p ON c.oracle_id = p.oracle_id
JOIN eternal_artisan_exception eae ON c.oracle_id = eae.oracle_id
LEFT JOIN banned_cards bc ON c.oracle_id = bc.oracle_id
LEFT JOIN digital_mechanic_cards dmc ON c.oracle_id = dmc.oracle_id
WHERE bc.oracle_id IS NULL AND dmc.oracle_id IS NULL
ORDER BY c.name, p.released_at DESC;

-- Get arena only EA cards with their printings, excluding banned cards
-- name: GetArenaOnlyEACards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at
FROM cards c
JOIN printings p ON c.oracle_id = p.oracle_id
JOIN arena_only_ea_cards aoea ON c.oracle_id = aoea.oracle_id
LEFT JOIN banned_cards bc ON c.oracle_id = bc.oracle_id
LEFT JOIN digital_mechanic_cards dmc ON c.oracle_id = dmc.oracle_id
WHERE bc.oracle_id IS NULL AND dmc.oracle_id IS NULL
ORDER BY c.name, p.released_at DESC;

-- Insert oracle_id into eternal artisan exception table
-- name: AddEternalArtisanException :exec
INSERT INTO eternal_artisan_exception (oracle_id) VALUES (?);

-- Insert oracle_id into arena only EA cards table
-- name: AddArenaOnlyEACard :exec
INSERT INTO arena_only_ea_cards (oracle_id) VALUES (?);

-- Remove oracle_id from eternal artisan exception table
-- name: RemoveEternalArtisanException :exec
DELETE FROM eternal_artisan_exception WHERE oracle_id = ?;

-- Remove oracle_id from arena only EA cards table
-- name: RemoveArenaOnlyEACard :exec
DELETE FROM arena_only_ea_cards WHERE oracle_id = ?;

-- Insert oracle_id into banned cards table
-- name: AddBannedCard :exec
INSERT INTO banned_cards (oracle_id) VALUES (?);

-- Remove oracle_id from banned cards table
-- name: RemoveBannedCard :exec
DELETE FROM banned_cards WHERE oracle_id = ?;

-- Get banned cards with their card details
-- name: GetBannedCards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at,
    p.image_uris,
    bc.added_at
FROM cards c
JOIN banned_cards bc ON c.oracle_id = bc.oracle_id
LEFT JOIN printings p ON c.oracle_id = p.oracle_id
ORDER BY c.name, p.released_at DESC;

-- Insert oracle_id into watchlist cards table
-- name: AddWatchlistCard :exec
INSERT INTO watchlist_cards (oracle_id) VALUES (?);

-- Remove oracle_id from watchlist cards table
-- name: RemoveWatchlistCard :exec
DELETE FROM watchlist_cards WHERE oracle_id = ?;

-- Get watchlist cards with their card details
-- name: GetWatchlistCards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at,
    p.image_uris,
    wc.added_at
FROM cards c
JOIN watchlist_cards wc ON c.oracle_id = wc.oracle_id
LEFT JOIN printings p ON c.oracle_id = p.oracle_id
ORDER BY c.name, p.released_at DESC;

-- Insert oracle_id into digital mechanic cards table
-- name: AddDigitalMechanicCard :exec
INSERT INTO digital_mechanic_cards (oracle_id, mechanic_keyword) VALUES (?, ?);

-- Remove oracle_id from digital mechanic cards table
-- name: RemoveDigitalMechanicCard :exec
DELETE FROM digital_mechanic_cards WHERE oracle_id = ?;

-- Get digital mechanic cards with their card details
-- name: GetDigitalMechanicCards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    p.id as printing_id,
    p.rarity,
    p.games,
    p."set",
    p.set_name,
    p.released_at,
    p.image_uris,
    dmc.mechanic_keyword,
    dmc.added_at
FROM cards c
JOIN digital_mechanic_cards dmc ON c.oracle_id = dmc.oracle_id
LEFT JOIN printings p ON c.oracle_id = p.oracle_id
ORDER BY c.name, p.released_at DESC;

-- Get arena cards that contain specific mechanic in oracle text
-- name: GetArenaCardsByMechanic :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness
FROM cards c
JOIN arena_only_ea_cards aoea ON c.oracle_id = aoea.oracle_id
WHERE c.oracle_text IS NOT NULL AND c.oracle_text LIKE '%' || ? || '%'
ORDER BY c.name;

-- Get all cards from all tables (one row per card)
-- name: GetAllCategorizedCards :many
SELECT 
    c.oracle_id,
    c.name,
    c.layout,
    c.cmc,
    c.color_identity,
    c.colors,
    c.mana_cost,
    c.oracle_text,
    c.type_line,
    c.power,
    c.toughness,
    CASE 
        WHEN eae.oracle_id IS NOT NULL THEN 'Eternal Artisan Exception'
        WHEN aoea.oracle_id IS NOT NULL THEN 'Arena Only EA'
        WHEN bc.oracle_id IS NOT NULL THEN 'Banned'
        WHEN wc.oracle_id IS NOT NULL THEN 'Watchlist'
        WHEN dmc.oracle_id IS NOT NULL THEN 'Digital Mechanic'
        ELSE 'Unknown'
    END as category,
    COALESCE(dmc.mechanic_keyword, '') as mechanic_keyword
FROM cards c
LEFT JOIN eternal_artisan_exception eae ON c.oracle_id = eae.oracle_id
LEFT JOIN arena_only_ea_cards aoea ON c.oracle_id = aoea.oracle_id
LEFT JOIN banned_cards bc ON c.oracle_id = bc.oracle_id
LEFT JOIN watchlist_cards wc ON c.oracle_id = wc.oracle_id
LEFT JOIN digital_mechanic_cards dmc ON c.oracle_id = dmc.oracle_id
WHERE eae.oracle_id IS NOT NULL 
   OR aoea.oracle_id IS NOT NULL 
   OR bc.oracle_id IS NOT NULL 
   OR wc.oracle_id IS NOT NULL 
   OR dmc.oracle_id IS NOT NULL
ORDER BY category, c.name;

-- Check if a card exists by oracle_id
-- name: CardExistsByOracleID :one
SELECT COUNT(*) FROM cards WHERE oracle_id = ? LIMIT 1;

-- Get a card by oracle_id
-- name: GetCardByOracleID :one
SELECT oracle_id, name, layout, cmc, color_identity, colors, mana_cost, oracle_text, type_line, power, toughness
FROM cards 
WHERE oracle_id = ? 
LIMIT 1;

-- Get a card by exact name
-- name: GetCardByName :one
SELECT oracle_id, name, layout, cmc, color_identity, colors, mana_cost, oracle_text, type_line, power, toughness
FROM cards 
WHERE LOWER(name) = LOWER(?) 
LIMIT 1;

-- Get printings by oracle_id
-- name: GetPrintingsByOracleID :many
SELECT 
    id,
    oracle_id,
    set_name,
    "set" as set_code,
    rarity,
    games,
    image_uris,
    artist,
    collector_number,
    released_at,
    scryfall_uri
FROM printings
WHERE oracle_id = ?
ORDER BY released_at DESC;

-- Get the best printing for image data (prioritize Arena, then most recent)
-- name: GetBestPrintingForImages :one
SELECT 
    image_uris
FROM printings
WHERE oracle_id = ? AND image_uris IS NOT NULL AND image_uris != ''
ORDER BY 
    CASE WHEN games LIKE '%arena%' THEN 0 ELSE 1 END,
    released_at DESC
LIMIT 1;

-- Insert or update a card (oracle-level)
-- name: UpsertCard :exec
INSERT INTO cards (
    oracle_id, name, layout, prints_search_uri, rulings_uri,
    all_parts, card_faces, cmc, color_identity, color_indicator, colors,
    defense, edhrec_rank, game_changer, hand_modifier, keywords, legalities,
    life_modifier, loyalty, mana_cost, oracle_text, penny_rank, power,
    produced_mana, reserved, toughness, type_line
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(oracle_id) DO UPDATE SET
    name = excluded.name,
    layout = excluded.layout,
    prints_search_uri = excluded.prints_search_uri,
    rulings_uri = excluded.rulings_uri,
    all_parts = excluded.all_parts,
    card_faces = excluded.card_faces,
    cmc = excluded.cmc,
    color_identity = excluded.color_identity,
    color_indicator = excluded.color_indicator,
    colors = excluded.colors,
    defense = excluded.defense,
    edhrec_rank = excluded.edhrec_rank,
    game_changer = excluded.game_changer,
    hand_modifier = excluded.hand_modifier,
    keywords = excluded.keywords,
    legalities = excluded.legalities,
    life_modifier = excluded.life_modifier,
    loyalty = excluded.loyalty,
    mana_cost = excluded.mana_cost,
    oracle_text = excluded.oracle_text,
    penny_rank = excluded.penny_rank,
    power = excluded.power,
    produced_mana = excluded.produced_mana,
    reserved = excluded.reserved,
    toughness = excluded.toughness,
    type_line = excluded.type_line;

-- Query Cache Operations

-- Get cached query result
-- name: GetCachedQuery :one
SELECT query_id, query_text, oracle_ids, cached_at, last_accessed, hit_count
FROM query_cache
WHERE query_text = ?
LIMIT 1;

-- Insert new query cache entry
-- name: InsertQueryCache :exec
INSERT INTO query_cache (query_text, oracle_ids)
VALUES (?, ?);

-- Update query cache hit (increment hit count and update last_accessed)
-- name: UpdateQueryCacheHit :exec
UPDATE query_cache
SET hit_count = hit_count + 1,
    last_accessed = CURRENT_TIMESTAMP
WHERE query_text = ?;

-- Delete old query cache entries (older than specified timestamp)
-- name: DeleteOldQueryCache :exec
DELETE FROM query_cache
WHERE cached_at < ?;

-- Get query cache stats
-- name: GetQueryCacheStats :one
SELECT 
    COUNT(*) as total_cached_queries,
    SUM(hit_count) as total_hits,
    AVG(hit_count) as avg_hits_per_query
FROM query_cache;



-- Insert or update a printing
-- name: UpsertPrinting :exec
INSERT INTO printings (
    id, oracle_id, arena_id, lang, mtgo_id, mtgo_foil_id, multiverse_ids,
    tcgplayer_id, tcgplayer_etched_id, cardmarket_id, object, scryfall_uri, uri,
    artist, artist_ids, attraction_lights, booster, border_color, card_back_id,
    collector_number, content_warning, digital, finishes, flavor_name, flavor_text,
    foil, nonfoil, frame_effects, frame, full_art, games, highres_image,
    illustration_id, image_status, image_uris, oversized, prices, printed_name,
    printed_text, printed_type_line, promo, promo_types, purchase_uris, rarity,
    related_uris, released_at, reprint, scryfall_set_uri, set_name, set_search_uri,
    set_type, set_uri, "set", set_id, story_spotlight, textless, variation,
    variation_of, security_stamp, watermark, preview
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(id) DO UPDATE SET
    oracle_id = excluded.oracle_id,
    arena_id = excluded.arena_id,
    lang = excluded.lang,
    mtgo_id = excluded.mtgo_id,
    mtgo_foil_id = excluded.mtgo_foil_id,
    multiverse_ids = excluded.multiverse_ids,
    tcgplayer_id = excluded.tcgplayer_id,
    tcgplayer_etched_id = excluded.tcgplayer_etched_id,
    cardmarket_id = excluded.cardmarket_id,
    object = excluded.object,
    scryfall_uri = excluded.scryfall_uri,
    uri = excluded.uri,
    artist = excluded.artist,
    artist_ids = excluded.artist_ids,
    attraction_lights = excluded.attraction_lights,
    booster = excluded.booster,
    border_color = excluded.border_color,
    card_back_id = excluded.card_back_id,
    collector_number = excluded.collector_number,
    content_warning = excluded.content_warning,
    digital = excluded.digital,
    finishes = excluded.finishes,
    flavor_name = excluded.flavor_name,
    flavor_text = excluded.flavor_text,
    foil = excluded.foil,
    nonfoil = excluded.nonfoil,
    frame_effects = excluded.frame_effects,
    frame = excluded.frame,
    full_art = excluded.full_art,
    games = excluded.games,
    highres_image = excluded.highres_image,
    illustration_id = excluded.illustration_id,
    image_status = excluded.image_status,
    image_uris = excluded.image_uris,
    oversized = excluded.oversized,
    prices = excluded.prices,
    printed_name = excluded.printed_name,
    printed_text = excluded.printed_text,
    printed_type_line = excluded.printed_type_line,
    promo = excluded.promo,
    promo_types = excluded.promo_types,
    purchase_uris = excluded.purchase_uris,
    rarity = excluded.rarity,
    related_uris = excluded.related_uris,
    released_at = excluded.released_at,
    reprint = excluded.reprint,
    scryfall_set_uri = excluded.scryfall_set_uri,
    set_name = excluded.set_name,
    set_search_uri = excluded.set_search_uri,
    set_type = excluded.set_type,
    set_uri = excluded.set_uri,
    "set" = excluded."set",
    set_id = excluded.set_id,
    story_spotlight = excluded.story_spotlight,
    textless = excluded.textless,
    variation = excluded.variation,
    variation_of = excluded.variation_of,
    security_stamp = excluded.security_stamp,
    watermark = excluded.watermark,
    preview = excluded.preview;