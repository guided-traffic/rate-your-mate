package database

import (
	"database/sql"
	"fmt"
	"log"
)

// DBType represents the type of database being used
type DBType string

const (
	// DBTypeSQLite represents SQLite database
	DBTypeSQLite DBType = "sqlite"
	// DBTypeMySQL represents MySQL database
	DBTypeMySQL DBType = "mysql"
)

// DB holds the global database connection
var DB *sql.DB

// dbType stores the current database type
var dbType DBType

// GetDBType returns the current database type
func GetDBType() DBType {
	return dbType
}

// IsSQLite returns true if the current database is SQLite
func IsSQLite() bool {
	return dbType == DBTypeSQLite
}

// IsMySQL returns true if the current database is MySQL
func IsMySQL() bool {
	return dbType == DBTypeMySQL
}

// Config holds database configuration for initialization
type Config struct {
	// Type of database: "sqlite" or "mysql"
	Type DBType

	// SQLite configuration
	SQLitePath string

	// MySQL configuration
	MySQL MySQLConfig
}

// Init initializes the database connection based on configuration
func Init(cfg Config) error {
	switch cfg.Type {
	case DBTypeSQLite:
		if cfg.SQLitePath == "" {
			return fmt.Errorf("SQLite path is required")
		}
		if err := initSQLite(cfg.SQLitePath); err != nil {
			return err
		}
		return runSQLiteMigrations()

	case DBTypeMySQL:
		if cfg.MySQL.Host == "" || cfg.MySQL.Database == "" {
			return fmt.Errorf("MySQL host and database are required")
		}
		if err := initMySQL(cfg.MySQL); err != nil {
			return err
		}
		return runMySQLMigrations()

	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// InitSQLite is a convenience function to initialize SQLite database
// This maintains backward compatibility with existing code
func InitSQLite(dbPath string) error {
	return Init(Config{
		Type:       DBTypeSQLite,
		SQLitePath: dbPath,
	})
}

// InitMySQL is a convenience function to initialize MySQL database
func InitMySQL(cfg MySQLConfig) error {
	return Init(Config{
		Type:  DBTypeMySQL,
		MySQL: cfg,
	})
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		log.Printf("Closing %s database connection", dbType)
		return DB.Close()
	}
	return nil
}

// WithTransaction executes a function within a transaction with retry support (for SQLite)
// If the function returns an error, the transaction is rolled back
// If the function succeeds, the transaction is committed
func WithTransaction(fn func(tx *sql.Tx) error) error {
	return WithRetry(func() error {
		tx, err := DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute the function
		if err := fn(tx); err != nil {
			// Attempt rollback, ignore rollback errors
			_ = tx.Rollback()
			return err
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	})
}
