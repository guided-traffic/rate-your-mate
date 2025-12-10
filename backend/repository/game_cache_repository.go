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
	AppID      int       `json:"app_id"`
	Name       string    `json:"name"`
	Categories string    `json:"categories"` // JSON array stored as string
	FetchedAt  time.Time `json:"fetched_at"`
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
		SELECT app_id, name, categories, fetched_at
		FROM game_cache WHERE app_id = ?`, appID,
	).Scan(&cache.AppID, &cache.Name, &cache.Categories, &cache.FetchedAt)

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
		SELECT app_id, name, categories, fetched_at
		FROM game_cache ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all game cache: %w", err)
	}
	defer rows.Close()

	var games []GameCache
	for rows.Next() {
		var game GameCache
		err := rows.Scan(&game.AppID, &game.Name, &game.Categories, &game.FetchedAt)
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
		SELECT app_id, name, categories, fetched_at
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
		err := rows.Scan(&game.AppID, &game.Name, &game.Categories, &game.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game cache row: %w", err)
		}
		games = append(games, game)
	}

	return games, nil
}

// Upsert creates or updates a cached game
func (r *GameCacheRepository) Upsert(appID int, name string, categories []string) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	_, err = database.DB.Exec(`
		INSERT INTO game_cache (app_id, name, categories, fetched_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(app_id) DO UPDATE SET
			name = excluded.name,
			categories = excluded.categories,
			fetched_at = CURRENT_TIMESTAMP`,
		appID, name, string(categoriesJSON),
	)
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
