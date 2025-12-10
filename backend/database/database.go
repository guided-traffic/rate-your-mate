package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB holds the database connection
var DB *sql.DB

// Init initializes the database connection and runs migrations
func Init(dbPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("Database initialized: %s", dbPath)
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// runMigrations creates all required tables
func runMigrations() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			steam_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			avatar_url TEXT,
			avatar_small TEXT,
			profile_url TEXT,
			credits INTEGER DEFAULT 0,
			last_credit_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create index for steam_id lookups
		`CREATE INDEX IF NOT EXISTS idx_users_steam_id ON users(steam_id)`,

		// Votes table
		`CREATE TABLE IF NOT EXISTS votes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_user_id INTEGER NOT NULL REFERENCES users(id),
			to_user_id INTEGER NOT NULL REFERENCES users(id),
			achievement_id TEXT NOT NULL,
			points INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			CHECK (from_user_id != to_user_id)
		)`,

		// Add points column to existing votes table (migration for existing DBs)
		`ALTER TABLE votes ADD COLUMN points INTEGER DEFAULT 1`,

		// Index for leaderboard queries
		`CREATE INDEX IF NOT EXISTS idx_votes_achievement ON votes(achievement_id, to_user_id)`,

		// Index for timeline queries
		`CREATE INDEX IF NOT EXISTS idx_votes_timeline ON votes(created_at DESC)`,

		// Chat messages table
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			message TEXT NOT NULL,
			achievements TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index for chat timeline queries
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_timeline ON chat_messages(created_at DESC)`,

		// Game cache table for Steam Store data
		`CREATE TABLE IF NOT EXISTS game_cache (
			app_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			categories TEXT DEFAULT '[]',
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index for stale game lookups
		`CREATE INDEX IF NOT EXISTS idx_game_cache_fetched ON game_cache(fetched_at)`,

		// Fix any NULL last_credit_at values (can happen from failed migrations)
		`UPDATE users SET last_credit_at = CURRENT_TIMESTAMP WHERE last_credit_at IS NULL`,
	}

	for _, migration := range migrations {
		_, err := DB.Exec(migration)
		if err != nil {
			// Ignore "duplicate column" errors for ALTER TABLE migrations
			errStr := err.Error()
			if contains(errStr, "duplicate column") || contains(errStr, "already exists") {
				continue
			}
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	log.Println("Database migrations completed")
	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
