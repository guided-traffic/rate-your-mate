package handlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/lan-party-manager/backend/auth"
	"github.com/guided-traffic/lan-party-manager/backend/config"
	"github.com/guided-traffic/lan-party-manager/backend/middleware"
	"github.com/guided-traffic/lan-party-manager/backend/repository"
	"github.com/guided-traffic/lan-party-manager/backend/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	cfg           *config.Config
	steamAuth     *auth.SteamAuth
	steamAPI      *auth.SteamAPIClient
	jwtService    *auth.JWTService
	userRepo      *repository.UserRepository
	creditService *services.CreditService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.Config, userRepo *repository.UserRepository, creditService *services.CreditService) *AuthHandler {
	return &AuthHandler{
		cfg:           cfg,
		steamAuth:     auth.NewSteamAuth(cfg.BackendURL),
		steamAPI:      auth.NewSteamAPIClient(cfg.SteamAPIKey),
		jwtService:    auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationDays),
		userRepo:      userRepo,
		creditService: creditService,
	}
}

// GetJWTService returns the JWT service for use in middleware
func (h *AuthHandler) GetJWTService() *auth.JWTService {
	return h.jwtService
}

// SteamLogin initiates the Steam OpenID login flow
// GET /api/v1/auth/steam
func (h *AuthHandler) SteamLogin(c *gin.Context) {
	authURL, err := h.steamAuth.GetAuthURL()
	if err != nil {
		log.Printf("Failed to get Steam auth URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initiate Steam login",
		})
		return
	}

	// Redirect to Steam login page
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// SteamCallback handles the Steam OpenID callback
// GET /api/v1/auth/steam/callback
func (h *AuthHandler) SteamCallback(c *gin.Context) {
	// Build the full callback URL from the request
	fullURL := auth.BuildFullCallbackURL(c.Request)

	// Validate the OpenID response and extract Steam ID
	steamID, err := h.steamAuth.ValidateCallback(fullURL)
	if err != nil {
		log.Printf("Steam callback validation failed: %v", err)
		h.redirectWithError(c, "Steam authentication failed")
		return
	}

	log.Printf("Steam login successful for Steam ID: %s", steamID)

	// Fetch player profile from Steam API
	var username, avatarURL, avatarSmall, profileURL string
	if h.steamAPI.IsConfigured() {
		player, err := h.steamAPI.GetPlayerSummary(steamID)
		if err != nil {
			log.Printf("Failed to fetch Steam profile for %s: %v", steamID, err)
			// Continue with default values - we still have the Steam ID
			username = "Player_" + steamID[len(steamID)-4:]
		} else {
			username = player.PersonaName
			avatarURL = player.AvatarFull
			avatarSmall = player.Avatar
			profileURL = player.ProfileURL
			log.Printf("Fetched Steam profile: %s (%s)", username, steamID)
		}
	} else {
		log.Println("Steam API not configured, using default profile data")
		username = "Player_" + steamID[len(steamID)-4:]
	}

	// Create or update user in database
	user, isNew, err := h.userRepo.FindOrCreate(steamID, username, avatarURL, avatarSmall, profileURL)
	if err != nil {
		log.Printf("Failed to create/update user: %v", err)
		h.redirectWithError(c, "Failed to create user account")
		return
	}

	if isNew {
		log.Printf("Created new user: %s (ID: %d)", username, user.ID)
	} else {
		log.Printf("Updated existing user: %s (ID: %d)", username, user.ID)
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(steamID, user.ID, username)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		h.redirectWithError(c, "Failed to generate authentication token")
		return
	}

	// Redirect to frontend with token
	redirectURL := h.buildFrontendRedirect(token, username, avatarURL)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT is stateless - logout is handled client-side by removing the token
	// We could implement a token blacklist here if needed
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the current authenticated user's information
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	// Load user from database
	user, err := h.userRepo.GetByID(claims.UserID)
	if err != nil {
		log.Printf("Failed to load user %d: %v", claims.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load user data",
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Calculate and update credits
	credits, err := h.creditService.CalculateAndUpdateCredits(user)
	if err != nil {
		log.Printf("Failed to update credits for user %d: %v", user.ID, err)
		// Continue with existing credits
		credits = user.Credits
	}

	// Calculate time until next credit
	timeUntilNext := h.creditService.GetTimeUntilNextCredit(user)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":                     user.ID,
			"steam_id":               user.SteamID,
			"username":               user.Username,
			"avatar_url":             user.AvatarURL,
			"avatar_small":           user.AvatarSmall,
			"profile_url":            user.ProfileURL,
			"credits":                credits,
			"seconds_until_credit":   int(timeUntilNext.Seconds()),
			"credit_interval_seconds": h.cfg.CreditIntervalMinutes * 60,
			"credit_max":             h.cfg.CreditMax,
		},
	})
}

// buildFrontendRedirect creates the redirect URL to the frontend with auth data
func (h *AuthHandler) buildFrontendRedirect(token, username, avatarURL string) string {
	redirectURL, _ := url.Parse(h.cfg.FrontendURL)
	redirectURL.Path = "/auth/callback"

	query := redirectURL.Query()
	query.Set("token", token)
	query.Set("username", username)
	if avatarURL != "" {
		query.Set("avatar", avatarURL)
	}
	redirectURL.RawQuery = query.Encode()

	return redirectURL.String()
}

// redirectWithError redirects to the frontend with an error message
func (h *AuthHandler) redirectWithError(c *gin.Context, errorMsg string) {
	redirectURL, _ := url.Parse(h.cfg.FrontendURL)
	redirectURL.Path = "/auth/callback"

	query := redirectURL.Query()
	query.Set("error", errorMsg)
	redirectURL.RawQuery = query.Encode()

	c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
}
