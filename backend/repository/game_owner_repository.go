package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/guided-traffic/rate-your-mate/backend/database"
)

// GameOwner represents a game ownership entry in the database
type GameOwner struct {
	AppID           int       `json:"app_id"`
	SteamID         string    `json:"steam_id"`
	PlaytimeForever int       `json:"playtime_forever"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GameOwnerRepository handles game owner database operations
type GameOwnerRepository struct{}

// NewGameOwnerRepository creates a new game owner repository
func NewGameOwnerRepository() *GameOwnerRepository {
	return &GameOwnerRepository{}
}

// GetOwnersByAppID returns all owners of a specific game
func (r *GameOwnerRepository) GetOwnersByAppID(appID int) ([]GameOwner, error) {
	rows, err := database.DB.Query(`
		SELECT app_id, steam_id, playtime_forever, created_at, updated_at
		FROM game_owners
		WHERE app_id = ?
		ORDER BY playtime_forever DESC`, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game owners by app id: %w", err)
	}
	defer rows.Close()

	var owners []GameOwner
	for rows.Next() {
		var owner GameOwner
		err := rows.Scan(&owner.AppID, &owner.SteamID, &owner.PlaytimeForever, &owner.CreatedAt, &owner.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game owner row: %w", err)
		}
		owners = append(owners, owner)
	}

	return owners, nil
}

// GetSteamIDsByAppID returns just the steam IDs of owners for a specific game
func (r *GameOwnerRepository) GetSteamIDsByAppID(appID int) ([]string, error) {
	rows, err := database.DB.Query(`
		SELECT steam_id
		FROM game_owners
		WHERE app_id = ?
		ORDER BY playtime_forever DESC`, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steam ids by app id: %w", err)
	}
	defer rows.Close()

	var steamIDs []string
	for rows.Next() {
		var steamID string
		err := rows.Scan(&steamID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan steam id: %w", err)
		}
		steamIDs = append(steamIDs, steamID)
	}

	return steamIDs, nil
}

// GetOwnerCountByAppID returns the number of owners for a specific game
func (r *GameOwnerRepository) GetOwnerCountByAppID(appID int) (int, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM game_owners WHERE app_id = ?`, appID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count game owners: %w", err)
	}
	return count, nil
}

// GetGamesByUserSteamID returns all games owned by a specific user
func (r *GameOwnerRepository) GetGamesByUserSteamID(steamID string) ([]GameOwner, error) {
	rows, err := database.DB.Query(`
		SELECT app_id, steam_id, playtime_forever, created_at, updated_at
		FROM game_owners
		WHERE steam_id = ?
		ORDER BY playtime_forever DESC`, steamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games by user steam id: %w", err)
	}
	defer rows.Close()

	var games []GameOwner
	for rows.Next() {
		var game GameOwner
		err := rows.Scan(&game.AppID, &game.SteamID, &game.PlaytimeForever, &game.CreatedAt, &game.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game owner row: %w", err)
		}
		games = append(games, game)
	}

	return games, nil
}

// GetAllOwnersGroupedByAppID returns a map of appID -> []steamID for all games
func (r *GameOwnerRepository) GetAllOwnersGroupedByAppID() (map[int][]string, error) {
	rows, err := database.DB.Query(`
		SELECT app_id, steam_id
		FROM game_owners
		ORDER BY app_id, playtime_forever DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all owners grouped by app id: %w", err)
	}
	defer rows.Close()

	result := make(map[int][]string)
	for rows.Next() {
		var appID int
		var steamID string
		err := rows.Scan(&appID, &steamID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan owner row: %w", err)
		}
		result[appID] = append(result[appID], steamID)
	}

	return result, nil
}

// Upsert creates or updates a game ownership entry
func (r *GameOwnerRepository) Upsert(appID int, steamID string, playtimeForever int) error {
	if database.IsSQLite() {
		_, err := database.DB.Exec(`
			INSERT INTO game_owners (app_id, steam_id, playtime_forever, created_at, updated_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(app_id, steam_id) DO UPDATE SET
				playtime_forever = excluded.playtime_forever,
				updated_at = CURRENT_TIMESTAMP`,
			appID, steamID, playtimeForever,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert game owner: %w", err)
		}
	} else {
		// MySQL/MariaDB syntax
		_, err := database.DB.Exec(`
			INSERT INTO game_owners (app_id, steam_id, playtime_forever, created_at, updated_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON DUPLICATE KEY UPDATE
				playtime_forever = VALUES(playtime_forever),
				updated_at = CURRENT_TIMESTAMP`,
			appID, steamID, playtimeForever,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert game owner: %w", err)
		}
	}
	return nil
}

// UpsertBatch upserts multiple game ownerships for a user efficiently
func (r *GameOwnerRepository) UpsertBatch(steamID string, games []struct {
	AppID           int
	PlaytimeForever int
}) error {
	if len(games) == 0 {
		return nil
	}

	// Use a transaction for better performance
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var stmt *sql.Stmt
	if database.IsSQLite() {
		stmt, err = tx.Prepare(`
			INSERT INTO game_owners (app_id, steam_id, playtime_forever, created_at, updated_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(app_id, steam_id) DO UPDATE SET
				playtime_forever = excluded.playtime_forever,
				updated_at = CURRENT_TIMESTAMP`)
	} else {
		stmt, err = tx.Prepare(`
			INSERT INTO game_owners (app_id, steam_id, playtime_forever, created_at, updated_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON DUPLICATE KEY UPDATE
				playtime_forever = VALUES(playtime_forever),
				updated_at = CURRENT_TIMESTAMP`)
	}
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, game := range games {
		_, err = stmt.Exec(game.AppID, steamID, game.PlaytimeForever)
		if err != nil {
			return fmt.Errorf("failed to upsert game owner %d for %s: %w", game.AppID, steamID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteByUserSteamID removes all game ownerships for a specific user
func (r *GameOwnerRepository) DeleteByUserSteamID(steamID string) error {
	_, err := database.DB.Exec(`DELETE FROM game_owners WHERE steam_id = ?`, steamID)
	if err != nil {
		return fmt.Errorf("failed to delete game owners by steam id: %w", err)
	}
	return nil
}

// DeleteByAppID removes all ownership entries for a specific game
func (r *GameOwnerRepository) DeleteByAppID(appID int) error {
	_, err := database.DB.Exec(`DELETE FROM game_owners WHERE app_id = ?`, appID)
	if err != nil {
		return fmt.Errorf("failed to delete game owners by app id: %w", err)
	}
	return nil
}

// DeleteAll removes all game ownership entries
func (r *GameOwnerRepository) DeleteAll() error {
	_, err := database.DB.Exec(`DELETE FROM game_owners`)
	if err != nil {
		return fmt.Errorf("failed to delete all game owners: %w", err)
	}
	return nil
}

// Exists checks if a specific ownership entry exists
func (r *GameOwnerRepository) Exists(appID int, steamID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM game_owners WHERE app_id = ? AND steam_id = ?`, appID, steamID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check game owner existence: %w", err)
	}
	return count > 0, nil
}
