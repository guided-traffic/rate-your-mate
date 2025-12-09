package repository

import (
	"encoding/json"
	"fmt"

	"github.com/guided-traffic/lan-party-manager/backend/database"
	"github.com/guided-traffic/lan-party-manager/backend/models"
)

// ChatRepository handles chat message database operations
type ChatRepository struct{}

// NewChatRepository creates a new chat repository
func NewChatRepository() *ChatRepository {
	return &ChatRepository{}
}

// Create creates a new chat message with the user's current achievements
func (r *ChatRepository) Create(msg *models.ChatMessage) error {
	result, err := database.DB.Exec(`
		INSERT INTO chat_messages (user_id, message, achievements)
		VALUES (?, ?, ?)`,
		msg.UserID, msg.Message, msg.Achievements,
	)
	if err != nil {
		return fmt.Errorf("failed to create chat message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	msg.ID = uint64(id)
	return nil
}

// GetRecent returns the most recent chat messages
func (r *ChatRepository) GetRecent(limit int) ([]models.ChatMessageWithUser, error) {
	rows, err := database.DB.Query(`
		SELECT
			cm.id, cm.message, cm.achievements, cm.created_at,
			u.id, u.steam_id, u.username, u.avatar_url, u.avatar_small, u.profile_url
		FROM chat_messages cm
		JOIN users u ON cm.user_id = u.id
		ORDER BY cm.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent chat messages: %w", err)
	}
	defer rows.Close()

	var messages []models.ChatMessageWithUser
	for rows.Next() {
		var m models.ChatMessageWithUser
		var achievementsJSON string
		err := rows.Scan(
			&m.ID, &m.Message, &achievementsJSON, &m.CreatedAt,
			&m.User.ID, &m.User.SteamID, &m.User.Username, &m.User.AvatarURL, &m.User.AvatarSmall, &m.User.ProfileURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat message row: %w", err)
		}

		// Parse achievements JSON
		if achievementsJSON != "" && achievementsJSON != "[]" {
			if err := json.Unmarshal([]byte(achievementsJSON), &m.Achievements); err != nil {
				// If parsing fails, just leave it empty
				m.Achievements = []models.AchievementBadge{}
			}
		} else {
			m.Achievements = []models.AchievementBadge{}
		}

		messages = append(messages, m)
	}

	return messages, nil
}

// GetByID returns a chat message by ID with full details
func (r *ChatRepository) GetByID(id uint64) (*models.ChatMessageWithUser, error) {
	var m models.ChatMessageWithUser
	var achievementsJSON string
	err := database.DB.QueryRow(`
		SELECT
			cm.id, cm.message, cm.achievements, cm.created_at,
			u.id, u.steam_id, u.username, u.avatar_url, u.avatar_small, u.profile_url
		FROM chat_messages cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.id = ?`, id,
	).Scan(
		&m.ID, &m.Message, &achievementsJSON, &m.CreatedAt,
		&m.User.ID, &m.User.SteamID, &m.User.Username, &m.User.AvatarURL, &m.User.AvatarSmall, &m.User.ProfileURL,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get chat message: %w", err)
	}

	// Parse achievements JSON
	if achievementsJSON != "" && achievementsJSON != "[]" {
		if err := json.Unmarshal([]byte(achievementsJSON), &m.Achievements); err != nil {
			m.Achievements = []models.AchievementBadge{}
		}
	} else {
		m.Achievements = []models.AchievementBadge{}
	}

	return &m, nil
}

// GetUserAchievementBadges returns the current achievement badges for a user (aggregated votes received)
func (r *ChatRepository) GetUserAchievementBadges(userID uint64) ([]models.AchievementBadge, error) {
	rows, err := database.DB.Query(`
		SELECT achievement_id, COUNT(*) as count
		FROM votes
		WHERE to_user_id = ?
		GROUP BY achievement_id
		ORDER BY count DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user achievements: %w", err)
	}
	defer rows.Close()

	var badges []models.AchievementBadge
	for rows.Next() {
		var achievementID string
		var count int
		if err := rows.Scan(&achievementID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan achievement row: %w", err)
		}

		// Get achievement details
		if achievement, ok := models.GetAchievement(achievementID); ok {
			badges = append(badges, models.AchievementBadge{
				ID:         achievement.ID,
				Name:       achievement.Name,
				ImageURL:   achievement.ImageURL,
				IsPositive: achievement.IsPositive,
				Count:      count,
			})
		}
	}

	return badges, nil
}
