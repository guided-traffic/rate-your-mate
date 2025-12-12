package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port        string
	FrontendURL string
	BackendURL  string

	// Database
	DBType     string // "sqlite" or "mysql"
	DBPath     string // SQLite database path

	// MySQL
	MySQLHost            string
	MySQLPort            int
	MySQLUser            string
	MySQLPassword        string
	MySQLDatabase        string
	MySQLTLSEnabled      bool
	MySQLTLSSkipVerify   bool
	MySQLTLSCACert       string // Path to CA certificate
	MySQLMaxOpenConns    int
	MySQLMaxIdleConns    int
	MySQLConnMaxLifetime time.Duration
	MySQLConnMaxIdleTime time.Duration

	// Steam
	SteamAPIKey string

	// JWT
	JWTSecret         string
	JWTExpirationDays int

	// Credits
	CreditIntervalMinutes int
	CreditMax             int

	// Voting
	VotingPaused       bool
	VotingPausedAt     time.Time // Timestamp when voting was paused (for freezing credit generation)
	VoteVisibilityMode string    // "user_choice", "all_secret", "all_public" - Default: user_choice

	// Admin
	AdminSteamIDs []string
	AdminPassword string // Optional password for additional admin panel security

	// Games
	PinnedGameIDs []int // App IDs of pinned/featured games
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists (for local development)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		// Server
		Port:        getEnv("PORT", "8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:4200"),
		BackendURL:  getEnv("BACKEND_URL", "http://localhost:8080"),

		// Database
		DBType: getEnv("DB_TYPE", "sqlite"),
		DBPath: getEnv("DB_PATH", "data/rate-your-mate.db"),

		// MySQL
		MySQLHost:            getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:            getEnvAsInt("MYSQL_PORT", 3306),
		MySQLUser:            getEnv("MYSQL_USER", ""),
		MySQLPassword:        getEnv("MYSQL_PASSWORD", ""),
		MySQLDatabase:        getEnv("MYSQL_DATABASE", "rate_your_mate"),
		MySQLTLSEnabled:      getEnvAsBool("MYSQL_TLS_ENABLED", false),
		MySQLTLSSkipVerify:   getEnvAsBool("MYSQL_TLS_SKIP_VERIFY", false),
		MySQLTLSCACert:       getEnv("MYSQL_TLS_CA_CERT", ""),
		MySQLMaxOpenConns:    getEnvAsInt("MYSQL_MAX_OPEN_CONNS", 25),
		MySQLMaxIdleConns:    getEnvAsInt("MYSQL_MAX_IDLE_CONNS", 5),
		MySQLConnMaxLifetime: getEnvAsDuration("MYSQL_CONN_MAX_LIFETIME", 5*time.Minute),
		MySQLConnMaxIdleTime: getEnvAsDuration("MYSQL_CONN_MAX_IDLE_TIME", 1*time.Minute),

		// Steam & Auth
		SteamAPIKey:       getEnv("STEAM_API_KEY", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTExpirationDays: getEnvAsInt("JWT_EXPIRATION_DAYS", 7),

		// Credits
		CreditIntervalMinutes: getEnvAsInt("CREDIT_INTERVAL_MINUTES", 10),
		CreditMax:             getEnvAsInt("CREDIT_MAX", 10),

		// Voting visibility - default to user choice
		VoteVisibilityMode: getEnv("VOTE_VISIBILITY_MODE", "user_choice"),

		// Admin
		AdminSteamIDs: getEnvAsStringSlice("ADMIN_STEAM_IDS", []string{}),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		PinnedGameIDs: getEnvAsIntSlice("PINNED_GAME_IDS", []int{}),
	}

	// Validate required configuration
	cfg.validate()

	return cfg
}

// validate checks that all required configuration is present
func (c *Config) validate() {
	if c.SteamAPIKey == "" {
		log.Println("WARNING: STEAM_API_KEY is not set - Steam profile data will not be available")
	}
	if c.JWTSecret == "" {
		log.Fatal("FATAL: JWT_SECRET must be set")
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt reads an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool reads an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration reads an environment variable as duration or returns a default value
// Supports formats like "5m", "1h", "30s"
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvAsStringSlice reads an environment variable as a comma-separated list of strings
func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

// getEnvAsIntSlice reads an environment variable as a comma-separated list of integers
func getEnvAsIntSlice(key string, defaultValue []int) []int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		parts := strings.Split(value, ",")
		result := make([]int, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				if intValue, err := strconv.Atoi(trimmed); err == nil {
					result = append(result, intValue)
				}
			}
		}
		return result
	}
	return defaultValue
}

// IsAdmin checks if the given Steam ID is in the admin list
func (c *Config) IsAdmin(steamID string) bool {
	for _, adminID := range c.AdminSteamIDs {
		if adminID == steamID {
			return true
		}
	}
	return false
}
