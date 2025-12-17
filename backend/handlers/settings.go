package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/rate-your-mate/backend/config"
	"github.com/guided-traffic/rate-your-mate/backend/middleware"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
	"github.com/guided-traffic/rate-your-mate/backend/websocket"
)

// SettingsHandler handles admin settings endpoints
type SettingsHandler struct {
	cfg      *config.Config
	wsHub    *websocket.Hub
	userRepo *repository.UserRepository
	voteRepo *repository.VoteRepository
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(cfg *config.Config, wsHub *websocket.Hub, userRepo *repository.UserRepository, voteRepo *repository.VoteRepository) *SettingsHandler {
	return &SettingsHandler{
		cfg:      cfg,
		wsHub:    wsHub,
		userRepo: userRepo,
		voteRepo: voteRepo,
	}
}

// GetSettingsRequest represents the response for GET /settings
type GetSettingsResponse struct {
	CreditIntervalMinutes  int     `json:"credit_interval_minutes"`
	CreditMax              int     `json:"credit_max"`
	VotingPaused           bool    `json:"voting_paused"`
	VoteVisibilityMode     string  `json:"vote_visibility_mode"` // "user_choice", "all_secret", "all_public"
	MinVotesForRanking     int     `json:"min_votes_for_ranking"`
	NegativeVotingDisabled bool    `json:"negative_voting_disabled"`
	CountdownTarget        *string `json:"countdown_target,omitempty"` // RFC3339 formatted time, null if not set
}

// UpdateSettingsRequest represents the request body for PUT /settings
type UpdateSettingsRequest struct {
	CreditIntervalMinutes  *int    `json:"credit_interval_minutes"`
	CreditMax              *int    `json:"credit_max"`
	VotingPaused           *bool   `json:"voting_paused"`
	VoteVisibilityMode     *string `json:"vote_visibility_mode"` // "user_choice", "all_secret", "all_public"
	MinVotesForRanking     *int    `json:"min_votes_for_ranking"`
	NegativeVotingDisabled *bool   `json:"negative_voting_disabled"`
	CountdownTarget        *string `json:"countdown_target"` // RFC3339 formatted time, empty string to clear
}

// VotingStatusResponse represents the response for GET /voting-status
type VotingStatusResponse struct {
	VotingPaused           bool    `json:"voting_paused"`
	NegativeVotingDisabled bool    `json:"negative_voting_disabled"`
	CountdownTarget        *string `json:"countdown_target,omitempty"` // RFC3339 formatted time, null if not set
}

// CountdownResponse represents the response for GET /countdown (public endpoint)
type CountdownResponse struct {
	CountdownTarget *string `json:"countdown_target,omitempty"` // RFC3339 formatted time, null if not set
}

// GetCountdown returns only the countdown target (public endpoint for login page)
// GET /api/v1/countdown
func (h *SettingsHandler) GetCountdown(c *gin.Context) {
	response := CountdownResponse{}
	if !h.cfg.CountdownTarget.IsZero() {
		formatted := h.cfg.CountdownTarget.Format(time.RFC3339)
		response.CountdownTarget = &formatted
	}
	c.JSON(http.StatusOK, response)
}

// GetVotingStatus returns only the voting paused status (for non-admin users)
// GET /api/v1/voting-status
func (h *SettingsHandler) GetVotingStatus(c *gin.Context) {
	response := VotingStatusResponse{
		VotingPaused:           h.cfg.VotingPaused,
		NegativeVotingDisabled: h.cfg.NegativeVotingDisabled,
	}
	if !h.cfg.CountdownTarget.IsZero() {
		formatted := h.cfg.CountdownTarget.Format(time.RFC3339)
		response.CountdownTarget = &formatted
	}
	c.JSON(http.StatusOK, response)
}

// GetSettings returns the current settings
// GET /api/v1/admin/settings
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	response := GetSettingsResponse{
		CreditIntervalMinutes:  h.cfg.CreditIntervalMinutes,
		CreditMax:              h.cfg.CreditMax,
		VotingPaused:           h.cfg.VotingPaused,
		VoteVisibilityMode:     h.cfg.VoteVisibilityMode,
		MinVotesForRanking:     h.cfg.MinVotesForRanking,
		NegativeVotingDisabled: h.cfg.NegativeVotingDisabled,
	}
	if !h.cfg.CountdownTarget.IsZero() {
		formatted := h.cfg.CountdownTarget.Format(time.RFC3339)
		response.CountdownTarget = &formatted
	}
	c.JSON(http.StatusOK, response)
}

// UpdateSettings updates the settings (admin only)
// PUT /api/v1/admin/settings
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate and update settings
	updated := false

	if req.CreditIntervalMinutes != nil {
		if *req.CreditIntervalMinutes < 1 || *req.CreditIntervalMinutes > 60 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "credit_interval_minutes must be between 1 and 60",
			})
			return
		}
		h.cfg.CreditIntervalMinutes = *req.CreditIntervalMinutes
		updated = true
		log.Printf("Admin updated credit_interval_minutes to %d", *req.CreditIntervalMinutes)
	}

	if req.CreditMax != nil {
		if *req.CreditMax < 1 || *req.CreditMax > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "credit_max must be between 1 and 100",
			})
			return
		}
		h.cfg.CreditMax = *req.CreditMax
		updated = true
		log.Printf("Admin updated credit_max to %d", *req.CreditMax)
	}

	if req.VotingPaused != nil {
		wasAlreadyPaused := h.cfg.VotingPaused
		h.cfg.VotingPaused = *req.VotingPaused
		updated = true

		if *req.VotingPaused {
			// Record when voting was paused
			h.cfg.VotingPausedAt = time.Now()
			log.Printf("Admin paused voting at %v", h.cfg.VotingPausedAt)
		} else if wasAlreadyPaused && !h.cfg.VotingPausedAt.IsZero() {
			// Voting is being resumed - shift all users' last_credit_at forward
			// by the pause duration so they don't accumulate time during pause
			pauseDuration := time.Since(h.cfg.VotingPausedAt)
			log.Printf("Admin resumed voting after %v pause", pauseDuration)

			// Shift all users' last_credit_at forward by the pause duration
			if err := h.userRepo.ShiftAllLastCreditAt(pauseDuration); err != nil {
				log.Printf("Warning: Failed to shift last_credit_at times: %v", err)
			} else {
				log.Printf("Shifted all users' last_credit_at forward by %v", pauseDuration)
			}

			// Reset the paused timestamp
			h.cfg.VotingPausedAt = time.Time{}
		} else {
			log.Printf("Admin resumed voting")
		}
	}

	if req.VoteVisibilityMode != nil {
		validModes := map[string]bool{"user_choice": true, "all_secret": true, "all_public": true}
		if !validModes[*req.VoteVisibilityMode] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "vote_visibility_mode must be 'user_choice', 'all_secret', or 'all_public'",
			})
			return
		}
		h.cfg.VoteVisibilityMode = *req.VoteVisibilityMode
		updated = true
		log.Printf("Admin updated vote_visibility_mode to %s", *req.VoteVisibilityMode)
	}

	if req.MinVotesForRanking != nil {
		if *req.MinVotesForRanking < 0 || *req.MinVotesForRanking > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "min_votes_for_ranking must be between 0 and 1000",
			})
			return
		}
		h.cfg.MinVotesForRanking = *req.MinVotesForRanking
		updated = true
		log.Printf("Admin updated min_votes_for_ranking to %d", *req.MinVotesForRanking)
	}

	if req.NegativeVotingDisabled != nil {
		h.cfg.NegativeVotingDisabled = *req.NegativeVotingDisabled
		updated = true
		if *req.NegativeVotingDisabled {
			log.Printf("Admin disabled negative voting")
		} else {
			log.Printf("Admin enabled negative voting")
		}
	}

	if req.CountdownTarget != nil {
		if *req.CountdownTarget == "" {
			// Clear the countdown
			h.cfg.CountdownTarget = time.Time{}
			updated = true
			log.Printf("Admin cleared countdown target")
		} else {
			// Parse and set the countdown
			parsedTime, err := time.Parse(time.RFC3339, *req.CountdownTarget)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "countdown_target must be in RFC3339 format (e.g., 2024-12-31T18:00:00Z)",
				})
				return
			}
			h.cfg.CountdownTarget = parsedTime
			updated = true
			log.Printf("Admin set countdown target to %v", parsedTime)
		}
	}

	// Broadcast settings change to all connected clients
	if updated {
		var countdownTarget *string
		if !h.cfg.CountdownTarget.IsZero() {
			formatted := h.cfg.CountdownTarget.Format(time.RFC3339)
			countdownTarget = &formatted
		}
		h.wsHub.BroadcastSettingsUpdate(&websocket.SettingsPayload{
			CreditIntervalMinutes:  h.cfg.CreditIntervalMinutes,
			CreditMax:              h.cfg.CreditMax,
			VotingPaused:           h.cfg.VotingPaused,
			VoteVisibilityMode:     h.cfg.VoteVisibilityMode,
			NegativeVotingDisabled: h.cfg.NegativeVotingDisabled,
			CountdownTarget:        countdownTarget,
		})
	}

	response := GetSettingsResponse{
		CreditIntervalMinutes:  h.cfg.CreditIntervalMinutes,
		CreditMax:              h.cfg.CreditMax,
		VotingPaused:           h.cfg.VotingPaused,
		VoteVisibilityMode:     h.cfg.VoteVisibilityMode,
		MinVotesForRanking:     h.cfg.MinVotesForRanking,
		NegativeVotingDisabled: h.cfg.NegativeVotingDisabled,
	}
	if !h.cfg.CountdownTarget.IsZero() {
		formatted := h.cfg.CountdownTarget.Format(time.RFC3339)
		response.CountdownTarget = &formatted
	}
	c.JSON(http.StatusOK, response)
}

// ResetAllCreditsResponse represents the response for POST /admin/credits/reset
type ResetAllCreditsResponse struct {
	Message       string `json:"message"`
	UsersAffected int64  `json:"users_affected"`
}

// GiveEveryoneCreditResponse represents the response for POST /admin/credits/give
type GiveEveryoneCreditResponse struct {
	Message       string `json:"message"`
	UsersAffected int64  `json:"users_affected"`
}

// ResetAllCredits sets all users' credits to 0
// POST /api/v1/admin/credits/reset
func (h *SettingsHandler) ResetAllCredits(c *gin.Context) {
	usersAffected, err := h.userRepo.ResetAllCredits()
	if err != nil {
		log.Printf("Error resetting all credits: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reset credits",
		})
		return
	}

	log.Printf("Admin reset all credits - %d users affected", usersAffected)

	// Broadcast credit reset to all connected clients
	h.wsHub.BroadcastCreditsReset()

	c.JSON(http.StatusOK, ResetAllCreditsResponse{
		Message:       "Alle Credits wurden auf 0 gesetzt",
		UsersAffected: usersAffected,
	})
}

// GiveEveryoneCredit gives each user 1 credit
// POST /api/v1/admin/credits/give
func (h *SettingsHandler) GiveEveryoneCredit(c *gin.Context) {
	usersAffected, err := h.userRepo.GiveEveryoneCredit(h.cfg.CreditMax)
	if err != nil {
		log.Printf("Error giving everyone credit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to give credits",
		})
		return
	}

	log.Printf("Admin gave everyone a credit - %d users affected", usersAffected)

	// Broadcast credit update to all connected clients
	h.wsHub.BroadcastCreditsGiven()

	c.JSON(http.StatusOK, GiveEveryoneCreditResponse{
		Message:       "Jedem Spieler wurde 1 Credit gegeben",
		UsersAffected: usersAffected,
	})
}

// AdminMiddleware checks if the current user is an admin
func (h *SettingsHandler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := middleware.GetClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Not authenticated",
			})
			c.Abort()
			return
		}

		if !h.cfg.IsAdmin(claims.SteamID) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// VerifyAdminPasswordRequest represents the request body for POST /admin/verify-password
type VerifyAdminPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// VerifyAdminPassword checks if the provided password matches the admin password
// POST /api/v1/admin/verify-password
func (h *SettingsHandler) VerifyAdminPassword(c *gin.Context) {
	// If no admin password is configured, always allow access
	if h.cfg.AdminPassword == "" {
		c.JSON(http.StatusOK, gin.H{
			"valid":             true,
			"password_required": false,
		})
		return
	}

	var req VerifyAdminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password is required",
		})
		return
	}

	if req.Password == h.cfg.AdminPassword {
		log.Printf("Admin password verified successfully")
		c.JSON(http.StatusOK, gin.H{
			"valid":             true,
			"password_required": true,
		})
	} else {
		log.Printf("Invalid admin password attempt")
		c.JSON(http.StatusForbidden, gin.H{
			"valid":             false,
			"password_required": true,
			"error":             "Invalid password",
		})
	}
}

// CheckAdminPasswordRequired checks if an admin password is configured
// GET /api/v1/admin/password-required
func (h *SettingsHandler) CheckAdminPasswordRequired(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"password_required": h.cfg.AdminPassword != "",
	})
}

// DeleteAllVotesResponse represents the response for POST /admin/votes/delete-all
type DeleteAllVotesResponse struct {
	Message       string `json:"message"`
	VotesDeleted  int64  `json:"votes_deleted"`
}

// DeleteAllVotes deletes all votes from the database
// POST /api/v1/admin/votes/delete-all
func (h *SettingsHandler) DeleteAllVotes(c *gin.Context) {
	votesDeleted, err := h.voteRepo.DeleteAll()
	if err != nil {
		log.Printf("Error deleting all votes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete votes",
		})
		return
	}

	log.Printf("Admin deleted all votes - %d votes deleted", votesDeleted)

	// Broadcast votes reset to all connected clients
	h.wsHub.BroadcastVotesReset()

	c.JSON(http.StatusOK, DeleteAllVotesResponse{
		Message:      "Alle Votes wurden gelöscht",
		VotesDeleted: votesDeleted,
	})
}

// KickUserRequest represents the request body for POST /admin/users/:id/kick
type KickUserRequest struct {
	Reason string `json:"reason"`
}

// BanUserRequest represents the request body for POST /admin/users/:id/ban
type BanUserRequest struct {
	Reason string `json:"reason"`
}

// GetAllUsersForAdmin returns all users for admin management
// GET /api/v1/admin/users
func (h *SettingsHandler) GetAllUsersForAdmin(c *gin.Context) {
	users, err := h.userRepo.GetAllForAdmin()
	if err != nil {
		log.Printf("Error getting users for admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// GetAllBannedUsers returns all banned users
// GET /api/v1/admin/users/banned
func (h *SettingsHandler) GetAllBannedUsers(c *gin.Context) {
	users, err := h.userRepo.GetAllBannedUsers()
	if err != nil {
		log.Printf("Error getting banned users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get banned users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"banned_users": users,
	})
}

// KickUser removes a user and all their data
// POST /api/v1/admin/users/:id/kick
func (h *SettingsHandler) KickUser(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	userID := c.Param("id")

	// Get user to kick
	var id uint64
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userRepo.GetByID(id)
	if err != nil {
		log.Printf("Error getting user for kick: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete the user (cascade will handle votes and chat messages)
	if err := h.userRepo.DeleteByID(id); err != nil {
		log.Printf("Error kicking user %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to kick user"})
		return
	}

	log.Printf("Admin %s kicked user %s (%s)", claims.SteamID, user.Username, user.SteamID)

	// Broadcast user kicked to all connected clients
	h.wsHub.BroadcastUserKicked(user.ID, user.Username)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Spieler wurde gekickt",
		"username": user.Username,
	})
}

// BanUser bans a user (removes them and prevents re-login)
// POST /api/v1/admin/users/:id/ban
func (h *SettingsHandler) BanUser(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	userID := c.Param("id")

	var req BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Reason is optional
		req.Reason = ""
	}

	// Get user to ban
	var id uint64
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userRepo.GetByID(id)
	if err != nil {
		log.Printf("Error getting user for ban: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent admin from banning themselves
	if user.SteamID == claims.SteamID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Du kannst dich nicht selbst bannen"})
		return
	}

	// Add to ban list
	if err := h.userRepo.BanUser(user.SteamID, user.Username, req.Reason, claims.SteamID); err != nil {
		log.Printf("Error banning user %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}

	// Delete the user (cascade will handle votes and chat messages)
	if err := h.userRepo.DeleteByID(id); err != nil {
		log.Printf("Error deleting banned user %d: %v", id, err)
		// Don't return error - user is already banned
	}

	log.Printf("Admin %s banned user %s (%s) - Reason: %s", claims.SteamID, user.Username, user.SteamID, req.Reason)

	// Broadcast user banned to all connected clients
	h.wsHub.BroadcastUserBanned(user.ID, user.Username)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Spieler wurde gebannt",
		"username": user.Username,
	})
}

// UnbanUser removes a user from the ban list
// POST /api/v1/admin/users/unban/:steam_id
func (h *SettingsHandler) UnbanUser(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	steamID := c.Param("steam_id")

	// Check if user is actually banned
	banned, err := h.userRepo.GetBannedUser(steamID)
	if err != nil {
		log.Printf("Error getting banned user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ban info"})
		return
	}
	if banned == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not banned"})
		return
	}

	// Remove from ban list
	if err := h.userRepo.UnbanUser(steamID); err != nil {
		log.Printf("Error unbanning user %s: %v", steamID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unban user"})
		return
	}

	log.Printf("Admin %s unbanned user %s (%s)", claims.SteamID, banned.Username, steamID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Spieler wurde entbannt",
		"username": banned.Username,
	})
}
