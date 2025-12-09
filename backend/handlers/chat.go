package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/lan-party-manager/backend/middleware"
	"github.com/guided-traffic/lan-party-manager/backend/models"
	"github.com/guided-traffic/lan-party-manager/backend/repository"
	"github.com/guided-traffic/lan-party-manager/backend/websocket"
)

// ChatHandler handles chat-related requests
type ChatHandler struct {
	chatRepo *repository.ChatRepository
	userRepo *repository.UserRepository
	wsHub    *websocket.Hub
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatRepo *repository.ChatRepository, userRepo *repository.UserRepository, wsHub *websocket.Hub) *ChatHandler {
	return &ChatHandler{
		chatRepo: chatRepo,
		userRepo: userRepo,
		wsHub:    wsHub,
	}
}

// GetMessages returns recent chat messages
// GET /api/v1/chat
func (h *ChatHandler) GetMessages(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	messages, err := h.chatRepo.GetRecent(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get chat messages",
		})
		return
	}

	// Reverse order so oldest is first (for display)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

// Create creates a new chat message
// POST /api/v1/chat
func (h *ChatHandler) Create(c *gin.Context) {
	// Get user from context (set by auth middleware)
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userID := claims.UserID
	username := claims.Username
	steamID := claims.SteamID

	// Parse request
	var req models.CreateChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Sanitize message
	message := strings.TrimSpace(req.Message)
	if len(message) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Message cannot be empty",
		})
		return
	}
	if len(message) > 500 {
		message = message[:500]
	}

	// Get user's current achievements
	achievements, err := h.chatRepo.GetUserAchievementBadges(userID)
	if err != nil {
		achievements = []models.AchievementBadge{}
	}

	// Convert achievements to JSON for storage
	achievementsJSON, err := json.Marshal(achievements)
	if err != nil {
		achievementsJSON = []byte("[]")
	}

	// Create chat message
	chatMsg := &models.ChatMessage{
		UserID:       userID,
		Message:      message,
		Achievements: string(achievementsJSON),
	}

	if err := h.chatRepo.Create(chatMsg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create chat message",
		})
		return
	}

	// Get the full message with user info
	fullMsg, err := h.chatRepo.GetByID(chatMsg.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chat message",
		})
		return
	}

	// Get user avatar info for WebSocket broadcast
	user, _ := h.userRepo.GetByID(userID)
	avatarSmall := ""
	if user != nil {
		avatarSmall = user.AvatarSmall
	}

	// Broadcast to all connected clients
	h.wsHub.BroadcastChatMessage(&websocket.ChatMessagePayload{
		ID:           fullMsg.ID,
		UserID:       userID,
		Username:     username,
		SteamID:      steamID,
		AvatarSmall:  avatarSmall,
		Message:      fullMsg.Message,
		Achievements: achievements,
		CreatedAt:    fullMsg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": fullMsg,
	})
}
