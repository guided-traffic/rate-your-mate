package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/lan-party-manager/backend/services"
)

// GameHandler handles game-related HTTP requests
type GameHandler struct {
	gameService       *services.GameService
	imageCacheService *services.ImageCacheService
}

// NewGameHandler creates a new game handler
func NewGameHandler(gameService *services.GameService, imageCacheService *services.ImageCacheService) *GameHandler {
	return &GameHandler{
		gameService:       gameService,
		imageCacheService: imageCacheService,
	}
}

// GetMultiplayerGames returns all multiplayer games owned by players
// GET /api/v1/games
func (h *GameHandler) GetMultiplayerGames(c *gin.Context) {
	games, err := h.gameService.GetMultiplayerGames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch games",
		})
		return
	}

	c.JSON(http.StatusOK, games)
}

// RefreshGames invalidates the cache and returns fresh game data
// POST /api/v1/games/refresh
func (h *GameHandler) RefreshGames(c *gin.Context) {
	h.gameService.InvalidateCache()

	games, err := h.gameService.GetMultiplayerGames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh games",
		})
		return
	}

	c.JSON(http.StatusOK, games)
}

// ServeGameImage serves a cached game image
// GET /api/v1/games/images/:filename
func (h *GameHandler) ServeGameImage(c *gin.Context) {
	filename := c.Param("filename")

	// Validate filename format (must be <appid>.jpg)
	if !strings.HasSuffix(filename, ".jpg") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format"})
		return
	}

	// Extract app ID from filename
	appIDStr := strings.TrimSuffix(filename, ".jpg")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Check if image exists locally
	imagePath := h.imageCacheService.GetImagePath(appID)

	// If not cached, try to cache it now
	if !h.imageCacheService.HasImage(appID) {
		if !h.imageCacheService.CacheImage(appID) {
			// Redirect to Steam CDN as fallback
			c.Redirect(http.StatusTemporaryRedirect, h.imageCacheService.GetSteamImageURL(appID))
			return
		}
	}

	// Serve the cached image
	c.Header("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	c.File(filepath.Clean(imagePath))
}
