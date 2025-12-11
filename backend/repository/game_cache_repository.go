package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guided-traffic/lan-party-manager/backend/database"
)

// GameCache represents a cached game entry in the database
type GameCache struct {
	AppID           int       `json:"app_id"`
	Name            string    `json:"name"`
	Categories      string    `json:"categories"` // JSON array stored as string
	IsFree          bool      `json:"is_free"`
	PriceCents      int       `json:"price_cents"`
	OriginalCents   int       `json:"original_cents"`
	DiscountPercent int       `json:"discount_percent"`
	PriceFormatted  string    `json:"price_formatted"`
	ReviewScore     int       `json:"review_score"` // Percentage of positive reviews (0-100), -1 if not enough reviews
	FetchFailed     bool      `json:"fetch_failed"` // True if game was not found (e.g., removed from Steam Store)
	FetchedAt       time.Time `json:"fetched_at"`
}

// GameCacheRepository handles game cache database operations
type GameCacheRepository struct{}

// NewGameCacheRepository creates a new game cache repository
func NewGameCacheRepository() *GameCacheRepository {
	return &GameCacheRepository{}
}

// GetByAppID finds a cached game by App ID
func (r *GameCacheRepository) GetByAppID(appID int) (*GameCache, error) {
	cache := &GameCache{}
	err := database.DB.QueryRow(`
		SELECT app_id, name, categories, is_free, price_cents, original_cents, discount_percent, price_formatted, review_score, fetch_failed, fetched_at
		FROM game_cache WHERE app_id = ?`, appID,
	).Scan(&cache.AppID, &cache.Name, &cache.Categories, &cache.IsFree, &cache.PriceCents, &cache.OriginalCents, &cache.DiscountPercent, &cache.PriceFormatted, &cache.ReviewScore, &cache.FetchFailed, &cache.FetchedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get game cache by app id: %w", err)
	}

	return cache, nil
}

// GetAll returns all cached games
func (r *GameCacheRepository) GetAll() ([]GameCache, error) {
	rows, err := database.DB.Query(`
		SELECT app_id, name, categories, is_free, price_cents, original_cents, discount_percent, price_formatted, review_score, fetch_failed, fetched_at
		FROM game_cache ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all game cache: %w", err)
	}
	defer rows.Close()

	var games []GameCache
	for rows.Next() {
		var game GameCache
		err := rows.Scan(&game.AppID, &game.Name, &game.Categories, &game.IsFree, &game.PriceCents, &game.OriginalCents, &game.DiscountPercent, &game.PriceFormatted, &game.ReviewScore, &game.FetchFailed, &game.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game cache row: %w", err)
		}
		games = append(games, game)
	}

	return games, nil
}

// GetStaleGames returns all games that need to be refreshed (older than maxAge)
func (r *GameCacheRepository) GetStaleGames(maxAge time.Duration) ([]GameCache, error) {
	cutoff := time.Now().Add(-maxAge)
	rows, err := database.DB.Query(`
		SELECT app_id, name, categories, is_free, price_cents, original_cents, discount_percent, price_formatted, review_score, fetch_failed, fetched_at
		FROM game_cache
		WHERE fetched_at < ?
		ORDER BY fetched_at ASC`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale games: %w", err)
	}
	defer rows.Close()

	var games []GameCache
	for rows.Next() {
		var game GameCache
		err := rows.Scan(&game.AppID, &game.Name, &game.Categories, &game.IsFree, &game.PriceCents, &game.OriginalCents, &game.DiscountPercent, &game.PriceFormatted, &game.ReviewScore, &game.FetchFailed, &game.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game cache row: %w", err)
		}
		games = append(games, game)
	}

	return games, nil
}

// GamePriceInfo contains price and review information for caching
type GamePriceInfo struct {
	IsFree          bool
	PriceCents      int
	OriginalCents   int
	DiscountPercent int
	PriceFormatted  string
	ReviewScore     int // Percentage of positive reviews (0-100), -1 if not enough reviews
}

// Upsert creates or updates a cached game
func (r *GameCacheRepository) Upsert(appID int, name string, categories []string, price *GamePriceInfo) error {
	return r.UpsertWithStatus(appID, name, categories, price, false)
}

// UpsertWithStatus creates or updates a cached game with fetch status
func (r *GameCacheRepository) UpsertWithStatus(appID int, name string, categories []string, price *GamePriceInfo, fetchFailed bool) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	// Default price info if nil
	if price == nil {
		price = &GamePriceInfo{ReviewScore: -1}
	}

	// Use database-specific upsert syntax
	if database.IsSQLite() {
		_, err = database.DB.Exec(`
			INSERT INTO game_cache (app_id, name, categories, is_free, price_cents, original_cents, discount_percent, price_formatted, review_score, fetch_failed, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(app_id) DO UPDATE SET
				name = excluded.name,
				categories = excluded.categories,
				is_free = excluded.is_free,
				price_cents = excluded.price_cents,
				original_cents = excluded.original_cents,
				discount_percent = excluded.discount_percent,
				price_formatted = excluded.price_formatted,
				review_score = excluded.review_score,
				fetch_failed = excluded.fetch_failed,
				fetched_at = CURRENT_TIMESTAMP`,
			appID, name, string(categoriesJSON), price.IsFree, price.PriceCents, price.OriginalCents, price.DiscountPercent, price.PriceFormatted, price.ReviewScore, fetchFailed,
		)
	} else {
		// MySQL/MariaDB syntax
		_, err = database.DB.Exec(`
			INSERT INTO game_cache (app_id, name, categories, is_free, price_cents, original_cents, discount_percent, price_formatted, review_score, fetch_failed, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				categories = VALUES(categories),
				is_free = VALUES(is_free),
				price_cents = VALUES(price_cents),
				original_cents = VALUES(original_cents),
				discount_percent = VALUES(discount_percent),
				price_formatted = VALUES(price_formatted),
				review_score = VALUES(review_score),
				fetch_failed = VALUES(fetch_failed),
				fetched_at = CURRENT_TIMESTAMP`,
			appID, name, string(categoriesJSON), price.IsFree, price.PriceCents, price.OriginalCents, price.DiscountPercent, price.PriceFormatted, price.ReviewScore, fetchFailed,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to upsert game cache: %w", err)
	}

	return nil
}

// GetCategories parses the categories JSON and returns a string slice
func (c *GameCache) GetCategories() []string {
	var categories []string
	if c.Categories != "" {
		json.Unmarshal([]byte(c.Categories), &categories)
	}
	return categories
}

// IsStale checks if the cache entry is older than the given duration
func (c *GameCache) IsStale(maxAge time.Duration) bool {
	return time.Since(c.FetchedAt) > maxAge
}

// Delete removes a cached game by App ID
func (r *GameCacheRepository) Delete(appID int) error {
	_, err := database.DB.Exec(`DELETE FROM game_cache WHERE app_id = ?`, appID)
	if err != nil {
		return fmt.Errorf("failed to delete game cache: %w", err)
	}
	return nil
}

// DeleteAll removes all cached games
func (r *GameCacheRepository) DeleteAll() error {
	_, err := database.DB.Exec(`DELETE FROM game_cache`)
	if err != nil {
		return fmt.Errorf("failed to delete all game cache: %w", err)
	}
	return nil
}

// InvalidateAll marks all cached games as stale by resetting fetched_at to epoch
func (r *GameCacheRepository) InvalidateAll() error {
	_, err := database.DB.Exec(`UPDATE game_cache SET fetched_at = '1970-01-01 00:00:00'`)
	if err != nil {
		return fmt.Errorf("failed to invalidate game cache: %w", err)
	}
	return nil
}
