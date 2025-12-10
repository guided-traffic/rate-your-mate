package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/guided-traffic/lan-party-manager/backend/database"
	"github.com/guided-traffic/lan-party-manager/backend/models"
)

// UserRepository handles user database operations
type UserRepository struct{}

// NewUserRepository creates a new user repository
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create creates a new user in the database (with retry for SQLITE_BUSY)
func (r *UserRepository) Create(user *models.User) error {
	return database.WithRetry(func() error {
		result, err := database.DB.Exec(`
			INSERT INTO users (steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			user.SteamID, user.Username, user.AvatarURL, user.AvatarSmall, user.ProfileURL, user.Credits, user.LastCreditAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert id: %w", err)
		}

		user.ID = uint64(id)
		return nil
	})
}

// GetByID finds a user by ID
func (r *UserRepository) GetByID(id uint64) (*models.User, error) {
	user := &models.User{}
	err := database.DB.QueryRow(`
		SELECT id, steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.SteamID, &user.Username, &user.AvatarURL, &user.AvatarSmall, &user.ProfileURL,
		&user.Credits, &user.LastCreditAt, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// GetBySteamID finds a user by Steam ID
func (r *UserRepository) GetBySteamID(steamID string) (*models.User, error) {
	user := &models.User{}
	err := database.DB.QueryRow(`
		SELECT id, steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at, created_at, updated_at
		FROM users WHERE steam_id = ?`, steamID,
	).Scan(&user.ID, &user.SteamID, &user.Username, &user.AvatarURL, &user.AvatarSmall, &user.ProfileURL,
		&user.Credits, &user.LastCreditAt, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by steam id: %w", err)
	}

	return user, nil
}

// GetAll returns all users
func (r *UserRepository) GetAll() ([]models.User, error) {
	rows, err := database.DB.Query(`
		SELECT id, steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at, created_at, updated_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.SteamID, &user.Username, &user.AvatarURL, &user.AvatarSmall, &user.ProfileURL,
			&user.Credits, &user.LastCreditAt, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// Update updates a user's profile information (with retry for SQLITE_BUSY)
func (r *UserRepository) Update(user *models.User) error {
	return database.WithRetry(func() error {
		_, err := database.DB.Exec(`
			UPDATE users
			SET username = ?, avatar_url = ?, avatar_small = ?, profile_url = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			user.Username, user.AvatarURL, user.AvatarSmall, user.ProfileURL, user.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		return nil
	})
}

// UpdateCredits updates a user's credits (with retry for SQLITE_BUSY)
func (r *UserRepository) UpdateCredits(userID uint64, credits int, lastCreditAt time.Time) error {
	return database.WithRetry(func() error {
		_, err := database.DB.Exec(`
			UPDATE users
			SET credits = ?, last_credit_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			credits, lastCreditAt, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to update credits: %w", err)
		}
		return nil
	})
}

// DeductCredit deducts one credit from a user (atomic operation)
func (r *UserRepository) DeductCredit(userID uint64) error {
	return r.DeductCredits(userID, 1)
}

// DeductCredits deducts a specified amount of credits from a user (atomic operation with retry)
func (r *UserRepository) DeductCredits(userID uint64, amount int) error {
	var rowsAffected int64

	err := database.WithRetry(func() error {
		result, err := database.DB.Exec(`
			UPDATE users
			SET credits = credits - ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND credits >= ?`,
			amount, userID, amount,
		)
		if err != nil {
			return fmt.Errorf("failed to deduct credits: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check rows affected: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("insufficient credits")
	}

	return nil
}

// ResetAllCredits sets all users' credits to 0 and resets the time until next credit (with retry for SQLITE_BUSY)
func (r *UserRepository) ResetAllCredits() (int64, error) {
	var rowsAffected int64

	err := database.WithRetry(func() error {
		result, err := database.DB.Exec(`
			UPDATE users
			SET credits = 0, last_credit_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP`)
		if err != nil {
			return fmt.Errorf("failed to reset all credits: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		return nil
	})

	return rowsAffected, err
}

// GiveEveryoneCredit gives each user 1 credit (respecting max credits, with retry for SQLITE_BUSY)
func (r *UserRepository) GiveEveryoneCredit(maxCredits int) (int64, error) {
	var rowsAffected int64

	err := database.WithRetry(func() error {
		result, err := database.DB.Exec(`
			UPDATE users
			SET credits = MIN(credits + 1, ?), updated_at = CURRENT_TIMESTAMP
			WHERE credits < ?`,
			maxCredits, maxCredits)
		if err != nil {
			return fmt.Errorf("failed to give everyone credit: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		return nil
	})

	return rowsAffected, err
}

// ShiftAllLastCreditAt shifts all users' last_credit_at forward by the given duration
// This is used when voting is resumed after a pause to prevent users from accumulating
// credit time during the pause
func (r *UserRepository) ShiftAllLastCreditAt(duration time.Duration) error {
	// Add the duration (in seconds) to all last_credit_at timestamps
	// We calculate the new timestamp in Go and update directly
	newTime := time.Now()

	// Get all users and update their last_credit_at by adding the pause duration
	rows, err := database.DB.Query(`SELECT id, last_credit_at FROM users`)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uint64
		var lastCreditAt time.Time
		if err := rows.Scan(&userID, &lastCreditAt); err != nil {
			// If last_credit_at is NULL, use current time
			continue
		}

		// Shift the timestamp forward by the pause duration
		newLastCreditAt := lastCreditAt.Add(duration)

		// Don't set it to the future
		if newLastCreditAt.After(newTime) {
			newLastCreditAt = newTime
		}

		// Update this user
		_, err := database.DB.Exec(`
			UPDATE users
			SET last_credit_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			newLastCreditAt, userID)
		if err != nil {
			return fmt.Errorf("failed to update user %d: %w", userID, err)
		}
	}

	return rows.Err()
}

// FindOrCreate finds a user by Steam ID or creates a new one
func (r *UserRepository) FindOrCreate(steamID, username, avatarURL, avatarSmall, profileURL string) (*models.User, bool, error) {
	// Try to find existing user
	user, err := r.GetBySteamID(steamID)
	if err != nil {
		return nil, false, err
	}

	if user != nil {
		// Update profile data if it changed
		if user.Username != username || user.AvatarURL != avatarURL {
			user.Username = username
			user.AvatarURL = avatarURL
			user.AvatarSmall = avatarSmall
			user.ProfileURL = profileURL
			if err := r.Update(user); err != nil {
				return nil, false, err
			}
		}
		return user, false, nil // false = existing user
	}

	// Create new user
	user = &models.User{
		SteamID:      steamID,
		Username:     username,
		AvatarURL:    avatarURL,
		AvatarSmall:  avatarSmall,
		ProfileURL:   profileURL,
		Credits:      0,
		LastCreditAt: time.Now(),
	}

	if err := r.Create(user); err != nil {
		return nil, false, err
	}

	return user, true, nil // true = new user created
}
