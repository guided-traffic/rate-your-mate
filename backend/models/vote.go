package models

import "time"

// Vote represents a vote from one user to another
type Vote struct {
	ID            uint64    `json:"id"`
	FromUserID    uint64    `json:"from_user_id"`
	ToUserID      uint64    `json:"to_user_id"`
	AchievementID string    `json:"achievement_id"`
	Points        int       `json:"points"`
	IsSecret      bool      `json:"is_secret"`
	CreatedAt     time.Time `json:"created_at"`
}

// VoteWithDetails includes user information for display
type VoteWithDetails struct {
	ID            uint64      `json:"id"`
	FromUser      PublicUser  `json:"from_user"`
	ToUser        PublicUser  `json:"to_user"`
	AchievementID string      `json:"achievement_id"`
	Achievement   Achievement `json:"achievement"`
	Points        int         `json:"points"`
	IsSecret      bool        `json:"is_secret"`
	CreatedAt     time.Time   `json:"created_at"`
}

// CreateVoteRequest is the request body for creating a vote
type CreateVoteRequest struct {
	ToUserID      uint64 `json:"to_user_id" binding:"required"`
	AchievementID string `json:"achievement_id" binding:"required"`
	Points        int    `json:"points"`    // 1-3 points, defaults to 1 if not provided
	IsSecret      *bool  `json:"is_secret"` // nil = use default (negative=secret, positive=open)
}

// AnonymousUser returns an anonymous PublicUser for secret votes
func AnonymousUser() PublicUser {
	return PublicUser{
		ID:          0,
		SteamID:     "",
		Username:    "Anonym",
		AvatarURL:   "",
		AvatarSmall: "",
		ProfileURL:  "",
	}
}

// ApplyVisibilityMode applies the visibility mode to a vote
// visibilityMode can be: "user_choice", "all_secret", "all_public"
func (v *VoteWithDetails) ApplyVisibilityMode(visibilityMode string) {
	shouldAnonymize := false

	switch visibilityMode {
	case "all_secret":
		shouldAnonymize = true
	case "all_public":
		shouldAnonymize = false
	default: // "user_choice"
		shouldAnonymize = v.IsSecret
	}

	if shouldAnonymize {
		v.FromUser = AnonymousUser()
	}
}
