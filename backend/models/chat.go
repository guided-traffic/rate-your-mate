package models

import "time"

// ChatMessage represents a chat message in the system
type ChatMessage struct {
	ID           uint64    `json:"id"`
	UserID       uint64    `json:"user_id"`
	Message      string    `json:"message"`
	Achievements string    `json:"achievements"` // JSON array of achievement IDs at time of message
	CreatedAt    time.Time `json:"created_at"`
}

// ChatMessageWithUser includes user information for display
type ChatMessageWithUser struct {
	ID           uint64           `json:"id"`
	User         PublicUser       `json:"user"`
	Message      string           `json:"message"`
	Achievements []AchievementBadge `json:"achievements"` // Achievement badges at time of message
	CreatedAt    time.Time        `json:"created_at"`
}

// AchievementBadge represents a simplified achievement for display as badge
type AchievementBadge struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ImageURL   string `json:"image_url"`
	IsPositive bool   `json:"is_positive"`
	Count      int    `json:"count"`
}

// CreateChatMessageRequest is the request body for creating a chat message
type CreateChatMessageRequest struct {
	Message string `json:"message" binding:"required,min=1,max=500"`
}
