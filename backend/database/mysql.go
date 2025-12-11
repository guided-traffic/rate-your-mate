package database

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

// MySQLConfig holds MySQL connection configuration
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	// TLS configuration
	TLSEnabled    bool
	TLSSkipVerify bool
	TLSCACert     string // Path to CA certificate file

	// Connection pool configuration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultMySQLConfig returns a MySQLConfig with sensible defaults
func DefaultMySQLConfig() MySQLConfig {
	return MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// initMySQL initializes a MySQL database connection
func initMySQL(cfg MySQLConfig) error {
	// First, try to create the database if it doesn't exist
	if err := ensureMySQLDatabaseExists(cfg); err != nil {
		return fmt.Errorf("failed to ensure database exists: %w", err)
	}

	// Build MySQL DSN
	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = cfg.User
	mysqlCfg.Passwd = cfg.Password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mysqlCfg.DBName = cfg.Database
	mysqlCfg.ParseTime = true
	mysqlCfg.Loc = time.UTC
	mysqlCfg.MultiStatements = true
	mysqlCfg.InterpolateParams = true

	// Configure TLS if enabled
	if cfg.TLSEnabled {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}

		// Register the TLS config with a unique name
		tlsConfigName := "custom"
		if err := mysql.RegisterTLSConfig(tlsConfigName, tlsConfig); err != nil {
			return fmt.Errorf("failed to register TLS config: %w", err)
		}
		mysqlCfg.TLSConfig = tlsConfigName
	}

	// Build DSN and open connection
	dsn := mysqlCfg.FormatDSN()
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(cfg.MaxOpenConns)
	DB.SetMaxIdleConns(cfg.MaxIdleConns)
	DB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	DB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test the connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	// Set database type
	dbType = DBTypeMySQL

	// Log connection info (without password)
	log.Printf("MySQL database initialized: %s@%s:%d/%s (TLS: %v)",
		cfg.User, cfg.Host, cfg.Port, cfg.Database, cfg.TLSEnabled)

	return nil
}

// ensureMySQLDatabaseExists connects without a database and creates it if necessary
func ensureMySQLDatabaseExists(cfg MySQLConfig) error {
	// Build MySQL DSN without database name
	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = cfg.User
	mysqlCfg.Passwd = cfg.Password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mysqlCfg.ParseTime = true
	mysqlCfg.Loc = time.UTC
	mysqlCfg.MultiStatements = true

	// Configure TLS if enabled
	if cfg.TLSEnabled {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}

		tlsConfigName := "custom-init"
		if err := mysql.RegisterTLSConfig(tlsConfigName, tlsConfig); err != nil {
			// Ignore error if already registered
			if err.Error() != "tls: failed to find any PEM data in certificate input" {
				// Try to use existing config
			}
		}
		mysqlCfg.TLSConfig = tlsConfigName
	}

	dsn := mysqlCfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL server: %w", err)
	}

	// Create database if it doesn't exist
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Database)
	_, err = db.Exec(createDBSQL)
	if err != nil {
		return fmt.Errorf("failed to create database '%s': %w", cfg.Database, err)
	}

	log.Printf("Ensured MySQL database '%s' exists", cfg.Database)
	return nil
}

// buildTLSConfig creates a TLS configuration for MySQL
func buildTLSConfig(cfg MySQLConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.TLSSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// If a CA certificate is provided, load and use it
	if cfg.TLSCACert != "" {
		caCert, err := os.ReadFile(cfg.TLSCACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
		// When using a custom CA, we still want to verify the server cert
		// but allow skip verify for hostname only if configured
		if !cfg.TLSSkipVerify {
			tlsConfig.InsecureSkipVerify = false
		}
	}

	return tlsConfig, nil
}

// runMySQLMigrations creates all required tables for MySQL
func runMySQLMigrations() error {
	migrations := []string{
		// Users table (MySQL-compatible)
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			steam_id VARCHAR(20) UNIQUE NOT NULL,
			username VARCHAR(255) NOT NULL,
			avatar_url TEXT,
			avatar_small TEXT,
			profile_url TEXT,
			credits INT DEFAULT 0,
			last_credit_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_steam_id (steam_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// Votes table (MySQL-compatible)
		`CREATE TABLE IF NOT EXISTS votes (
			id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			from_user_id BIGINT UNSIGNED NOT NULL,
			to_user_id BIGINT UNSIGNED NOT NULL,
			achievement_id VARCHAR(50) NOT NULL,
			points INT DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_votes_achievement (achievement_id, to_user_id),
			INDEX idx_votes_timeline (created_at DESC),
			CONSTRAINT chk_no_self_vote CHECK (from_user_id != to_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// Chat messages table (MySQL-compatible)
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT UNSIGNED NOT NULL,
			message TEXT NOT NULL,
			achievements TEXT DEFAULT ('[]'),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_chat_messages_timeline (created_at DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// Game cache table for Steam Store data (MySQL-compatible)
		`CREATE TABLE IF NOT EXISTS game_cache (
			app_id BIGINT UNSIGNED PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			categories TEXT DEFAULT ('[]'),
			is_free TINYINT(1) DEFAULT 0,
			price_cents INT DEFAULT 0,
			original_cents INT DEFAULT 0,
			discount_percent INT DEFAULT 0,
			price_formatted VARCHAR(50) DEFAULT '',
			fetch_failed TINYINT(1) DEFAULT 0,
			review_score INT DEFAULT -1,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_game_cache_fetched (fetched_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, migration := range migrations {
		_, err := DB.Exec(migration)
		if err != nil {
			// Ignore "table already exists" errors
			if containsIgnoreCase(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("MySQL migration failed: %w\nSQL: %s", err, migration)
		}
	}

	// Run ALTER TABLE migrations separately to handle existing databases
	alterMigrations := []struct {
		sql    string
		column string
	}{
		{`ALTER TABLE votes ADD COLUMN points INT DEFAULT 1`, "points"},
		{`ALTER TABLE game_cache ADD COLUMN is_free TINYINT(1) DEFAULT 0`, "is_free"},
		{`ALTER TABLE game_cache ADD COLUMN price_cents INT DEFAULT 0`, "price_cents"},
		{`ALTER TABLE game_cache ADD COLUMN original_cents INT DEFAULT 0`, "original_cents"},
		{`ALTER TABLE game_cache ADD COLUMN discount_percent INT DEFAULT 0`, "discount_percent"},
		{`ALTER TABLE game_cache ADD COLUMN price_formatted VARCHAR(50) DEFAULT ''`, "price_formatted"},
		{`ALTER TABLE game_cache ADD COLUMN fetch_failed TINYINT(1) DEFAULT 0`, "fetch_failed"},
		{`ALTER TABLE game_cache ADD COLUMN review_score INT DEFAULT -1`, "review_score"},
	}

	for _, m := range alterMigrations {
		_, err := DB.Exec(m.sql)
		if err != nil {
			// Ignore "duplicate column" errors
			if containsIgnoreCase(err.Error(), "duplicate column") {
				continue
			}
			// Ignore MySQL error 1060: Duplicate column name
			if containsIgnoreCase(err.Error(), "1060") {
				continue
			}
			log.Printf("Warning: ALTER TABLE migration for column '%s' failed: %v", m.column, err)
		}
	}

	// Fix any NULL last_credit_at values
	_, err := DB.Exec(`UPDATE users SET last_credit_at = CURRENT_TIMESTAMP WHERE last_credit_at IS NULL`)
	if err != nil {
		log.Printf("Warning: Failed to fix NULL last_credit_at values: %v", err)
	}

	log.Println("MySQL migrations completed")
	return nil
}
