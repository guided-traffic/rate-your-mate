package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ErrBusy is returned when SQLite is busy after all retries
var ErrBusy = errors.New("database is busy, please try again")

// initSQLite initializes a SQLite database connection
func initSQLite(dbPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection with optimized settings for concurrent access
	// _journal_mode=WAL enables Write-Ahead Logging for better concurrent writes
	// _busy_timeout=10000 waits up to 10 seconds before returning SQLITE_BUSY
	// _synchronous=NORMAL is a good balance between safety and performance
	// _cache_size=1000 increases the page cache size
	// _foreign_keys=ON enables foreign key constraints
	// _txlock=immediate ensures write transactions get the lock immediately
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=10000&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=ON&_txlock=immediate", dbPath)

	var err error
	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for SQLite with WAL mode
	// WAL mode allows multiple readers and one writer concurrently
	// We use a small pool to avoid connection overhead while allowing some concurrency
	DB.SetMaxOpenConns(5)                   // Allow multiple connections for concurrent reads
	DB.SetMaxIdleConns(2)                   // Keep some connections warm
	DB.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections periodically
	DB.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections

	// Test the connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Verify WAL mode is enabled
	var journalMode string
	if err := DB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		log.Printf("Warning: Could not verify journal mode: %v", err)
	} else {
		log.Printf("SQLite journal mode: %s", journalMode)
	}

	// Set database type
	dbType = DBTypeSQLite

	log.Printf("SQLite database initialized: %s", dbPath)
	return nil
}

// isBusyError checks if an error is a SQLite BUSY error
func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "busy") || strings.Contains(errStr, "locked")
}

// WithRetry executes a function with retry logic for SQLITE_BUSY errors
// It will retry up to maxRetries times with exponential backoff
// For MySQL, the function is executed without retry logic
func WithRetry(fn func() error) error {
	return WithRetryContext(context.Background(), fn)
}

// WithRetryContext executes a function with retry logic and context support
// For MySQL, the function is executed without retry logic
func WithRetryContext(ctx context.Context, fn func() error) error {
	// For MySQL, no retry needed - just execute the function
	if dbType == DBTypeMySQL {
		return fn()
	}

	// SQLite retry logic
	const maxRetries = 5
	baseDelay := 50 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Only retry on SQLITE_BUSY errors
		if !isBusyError(lastErr) {
			return lastErr
		}

		// Log retry attempt
		if attempt > 0 {
			log.Printf("SQLite busy, retry attempt %d/%d", attempt+1, maxRetries)
		}

		// Exponential backoff: 50ms, 100ms, 200ms, 400ms, 800ms
		delay := baseDelay * time.Duration(1<<attempt)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	log.Printf("SQLite busy after %d retries: %v", maxRetries, lastErr)
	return ErrBusy
}

// runSQLiteMigrations creates all required tables for SQLite
func runSQLiteMigrations() error {
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
			is_free INTEGER DEFAULT 0,
			price_cents INTEGER DEFAULT 0,
			original_cents INTEGER DEFAULT 0,
			discount_percent INTEGER DEFAULT 0,
			price_formatted TEXT DEFAULT '',
			fetch_failed INTEGER DEFAULT 0,
			review_score INTEGER DEFAULT -1,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index for stale game lookups
		`CREATE INDEX IF NOT EXISTS idx_game_cache_fetched ON game_cache(fetched_at)`,

		// Add price columns to existing game_cache table (migration for existing DBs)
		`ALTER TABLE game_cache ADD COLUMN is_free INTEGER DEFAULT 0`,
		`ALTER TABLE game_cache ADD COLUMN price_cents INTEGER DEFAULT 0`,
		`ALTER TABLE game_cache ADD COLUMN original_cents INTEGER DEFAULT 0`,
		`ALTER TABLE game_cache ADD COLUMN discount_percent INTEGER DEFAULT 0`,
		`ALTER TABLE game_cache ADD COLUMN price_formatted TEXT DEFAULT ''`,

		// Add fetch_failed column for games that couldn't be fetched (e.g., removed from Steam Store)
		`ALTER TABLE game_cache ADD COLUMN fetch_failed INTEGER DEFAULT 0`,

		// Add review_score column for Steam user reviews percentage
		`ALTER TABLE game_cache ADD COLUMN review_score INTEGER DEFAULT -1`,

		// Fix any NULL last_credit_at values (can happen from failed migrations)
		`UPDATE users SET last_credit_at = CURRENT_TIMESTAMP WHERE last_credit_at IS NULL`,
	}

	for _, migration := range migrations {
		_, err := DB.Exec(migration)
		if err != nil {
			// Ignore "duplicate column" errors for ALTER TABLE migrations
			errStr := err.Error()
			if containsIgnoreCase(errStr, "duplicate column") || containsIgnoreCase(errStr, "already exists") {
				continue
			}
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	log.Println("SQLite migrations completed")
	return nil
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
