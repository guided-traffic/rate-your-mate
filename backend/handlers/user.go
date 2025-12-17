package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/rate-your-mate/backend/middleware"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
	"github.com/guided-traffic/rate-your-mate/backend/services"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	userRepo           *repository.UserRepository
	avatarCacheService *services.AvatarCacheService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo *repository.UserRepository, avatarCacheService *services.AvatarCacheService) *UserHandler {
	return &UserHandler{
		userRepo:           userRepo,
		avatarCacheService: avatarCacheService,
	}
}

// GetAll returns all registered users
// GET /api/v1/users
func (h *UserHandler) GetAll(c *gin.Context) {
	users, err := h.userRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load users",
		})
		return
	}

	// Convert to public user data
	publicUsers := make([]gin.H, len(users))
	for i, user := range users {
		publicUsers[i] = gin.H{
			"id":           user.ID,
			"steam_id":     user.SteamID,
			"username":     user.Username,
			"avatar_url":   user.AvatarURL,
			"avatar_small": user.AvatarSmall,
			"profile_url":  user.ProfileURL,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": publicUsers,
	})
}

// GetByID returns a single user by ID
// GET /api/v1/users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	user, err := h.userRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load user",
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"steam_id":     user.SteamID,
			"username":     user.Username,
			"avatar_url":   user.AvatarURL,
			"avatar_small": user.AvatarSmall,
			"profile_url":  user.ProfileURL,
		},
	})
}

// GetOthers returns all users except the current user (for voting)
// GET /api/v1/users/others
func (h *UserHandler) GetOthers(c *gin.Context) {
	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	users, err := h.userRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load users",
		})
		return
	}

	// Filter out current user
	publicUsers := make([]gin.H, 0)
	for _, user := range users {
		if user.ID != currentUserID {
			publicUsers = append(publicUsers, gin.H{
				"id":           user.ID,
				"steam_id":     user.SteamID,
				"username":     user.Username,
				"avatar_url":   user.AvatarURL,
				"avatar_small": user.AvatarSmall,
				"profile_url":  user.ProfileURL,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": publicUsers,
	})
}

// ServeAvatar serves a cached avatar image
// GET /api/v1/avatars/:filename
func (h *UserHandler) ServeAvatar(c *gin.Context) {
	filename := c.Param("filename")

	// Validate filename format (should contain steamID and hash)
	if !strings.Contains(filename, "_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid avatar filename"})
		return
	}

	// Check for valid extensions
	if !strings.HasSuffix(filename, ".jpg") && !strings.HasSuffix(filename, ".svg") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format"})
		return
	}

	// Check if avatar exists locally
	if !h.avatarCacheService.HasAvatarFile(filename) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Avatar not found"})
		return
	}

	// Determine content type
	contentType := "image/jpeg"
	if strings.HasSuffix(filename, ".svg") {
		contentType = "image/svg+xml"
	}

	// Serve the cached avatar
	avatarPath := h.avatarCacheService.GetAvatarByFilename(filename)
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=604800") // Cache for 7 days
	c.File(filepath.Clean(avatarPath))
}
