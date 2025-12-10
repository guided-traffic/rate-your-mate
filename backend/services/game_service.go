package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/lan-party-manager/backend/config"
	"github.com/guided-traffic/lan-party-manager/backend/models"
	"github.com/guided-traffic/lan-party-manager/backend/repository"
)

const (
	steamAPIBaseURL   = "https://api.steampowered.com"
	steamStoreBaseURL = "https://store.steampowered.com/api"
	steamCDNBaseURL   = "https://steamcdn-a.akamaihd.net/steam/apps"

	// Cache settings
	gameCacheMaxAge      = 24 * time.Hour // Refresh game data after 24 hours
	rateLimitPausePeriod = 5 * time.Minute // Pause for 5 minutes after 429 error
)

// GameService handles game-related operations
type GameService struct {
	cfg               *config.Config
	userRepo          *repository.UserRepository
	gameCacheRepo     *repository.GameCacheRepository
	imageCacheService *ImageCacheService
	httpClient        *http.Client
	cache             *gamesCache
	rateLimiter       *rateLimiter
}

// gamesCache caches the full response to avoid rebuilding it constantly
type gamesCache struct {
	mu        sync.RWMutex
	games     *models.GamesResponse
	expiresAt time.Time
}

// rateLimiter tracks rate limit status
type rateLimiter struct {
	mu          sync.RWMutex
	pausedUntil time.Time
	isPaused    bool
}

// NewGameService creates a new game service
func NewGameService(cfg *config.Config, userRepo *repository.UserRepository, gameCacheRepo *repository.GameCacheRepository, imageCacheService *ImageCacheService) *GameService {
	return &GameService{
		cfg:               cfg,
		userRepo:          userRepo,
		gameCacheRepo:     gameCacheRepo,
		imageCacheService: imageCacheService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache:       &gamesCache{},
		rateLimiter: &rateLimiter{},
	}
}

// GetMultiplayerGames returns all multiplayer games owned by registered players
func (s *GameService) GetMultiplayerGames() (*models.GamesResponse, error) {
	// Check in-memory cache first (5 minute TTL for response assembly)
	s.cache.mu.RLock()
	if s.cache.games != nil && time.Now().Before(s.cache.expiresAt) {
		cached := s.cache.games
		s.cache.mu.RUnlock()
		return cached, nil
	}
	s.cache.mu.RUnlock()

	// Fetch fresh data
	games, err := s.fetchMultiplayerGames()
	if err != nil {
		return nil, err
	}

	// Update in-memory cache
	s.cache.mu.Lock()
	s.cache.games = games
	s.cache.expiresAt = time.Now().Add(5 * time.Minute)
	s.cache.mu.Unlock()

	return games, nil
}

// InvalidateCache clears the in-memory games cache (DB cache remains)
func (s *GameService) InvalidateCache() {
	s.cache.mu.Lock()
	s.cache.games = nil
	s.cache.expiresAt = time.Time{}
	s.cache.mu.Unlock()
}

// isRateLimited checks if we're currently rate limited
func (s *GameService) isRateLimited() bool {
	s.rateLimiter.mu.RLock()
	defer s.rateLimiter.mu.RUnlock()
	return s.rateLimiter.isPaused && time.Now().Before(s.rateLimiter.pausedUntil)
}

// setRateLimited sets the rate limit pause
func (s *GameService) setRateLimited() {
	s.rateLimiter.mu.Lock()
	defer s.rateLimiter.mu.Unlock()
	s.rateLimiter.isPaused = true
	s.rateLimiter.pausedUntil = time.Now().Add(rateLimitPausePeriod)
	log.Printf("Steam API rate limited - pausing requests for %v", rateLimitPausePeriod)
}

// fetchMultiplayerGames fetches all games from all users and filters for multiplayer
func (s *GameService) fetchMultiplayerGames() (*models.GamesResponse, error) {
	// Get all registered users
	users, err := s.userRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		return &models.GamesResponse{
			PinnedGames: []models.Game{},
			AllGames:    []models.Game{},
		}, nil
	}

	// Collect all games from all users
	gameMap := make(map[int]*models.Game) // appID -> game
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, user := range users {
		wg.Add(1)
		go func(steamID string) {
			defer wg.Done()

			games, err := s.fetchUserGames(steamID)
			if err != nil {
				log.Printf("Failed to fetch games for user %s: %v", steamID, err)
				return
			}

			mu.Lock()
			for _, g := range games {
				if existing, ok := gameMap[g.AppID]; ok {
					// Game already exists, add this owner
					existing.OwnerCount++
					existing.Owners = append(existing.Owners, steamID)
					// Keep the higher playtime
					if g.PlaytimeForever > existing.PlaytimeForever {
						existing.PlaytimeForever = g.PlaytimeForever
					}
				} else {
					// New game - try to load from DB cache first
					game := &models.Game{
						AppID:           g.AppID,
						Name:            g.Name,
						HeaderImageURL:  s.imageCacheService.GetLocalImageURL(g.AppID),
						CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, g.AppID),
						PlaytimeForever: g.PlaytimeForever,
						OwnerCount:      1,
						Owners:          []string{steamID},
						Categories:      []string{},
					}

					// Cache the image asynchronously
					s.imageCacheService.CacheImageAsync(g.AppID)

					// Try to load categories from DB cache
					cached, err := s.gameCacheRepo.GetByAppID(g.AppID)
					if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
						game.Categories = cached.GetCategories()
						if cached.Name != "" {
							game.Name = cached.Name
						}
					}

					gameMap[g.AppID] = game
				}
			}
			mu.Unlock()
		}(user.SteamID)
	}

	wg.Wait()

	// Identify games that need their categories fetched from Steam Store API
	var gamesToFetch []*models.Game
	for _, game := range gameMap {
		if len(game.Categories) == 0 {
			// Check DB cache again (might have been populated by another goroutine)
			cached, err := s.gameCacheRepo.GetByAppID(game.AppID)
			if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
				game.Categories = cached.GetCategories()
				if cached.Name != "" {
					game.Name = cached.Name
				}
			} else {
				gamesToFetch = append(gamesToFetch, game)
			}
		}
	}

	// Sort by owner count to prioritize popular games
	sort.Slice(gamesToFetch, func(i, j int) bool {
		return gamesToFetch[i].OwnerCount > gamesToFetch[j].OwnerCount
	})

	// Fetch categories from Steam Store API (with rate limiting and DB caching)
	s.fetchGameCategories(gamesToFetch)

	// Filter for multiplayer games and build response
	var allGames []models.Game
	pinnedGameIDs := s.cfg.PinnedGameIDs

	for _, game := range gameMap {
		if game.HasMultiplayerCategory() {
			// Check if pinned
			for _, pinnedID := range pinnedGameIDs {
				if pinnedID == game.AppID {
					game.IsPinned = true
					break
				}
			}
			allGames = append(allGames, *game)
		}
	}

	// Sort all games by owner count (most owners first), then by name
	sort.Slice(allGames, func(i, j int) bool {
		if allGames[i].OwnerCount != allGames[j].OwnerCount {
			return allGames[i].OwnerCount > allGames[j].OwnerCount
		}
		return allGames[i].Name < allGames[j].Name
	})

	// Separate pinned games
	var pinnedGames []models.Game
	var unpinnedGames []models.Game

	for _, game := range allGames {
		if game.IsPinned {
			pinnedGames = append(pinnedGames, game)
		} else {
			unpinnedGames = append(unpinnedGames, game)
		}
	}

	// Also add pinned games that might not be in any user's library
	for _, pinnedID := range pinnedGameIDs {
		found := false
		for _, g := range pinnedGames {
			if g.AppID == pinnedID {
				found = true
				break
			}
		}
		if !found {
			// Fetch this pinned game's details
			game, err := s.fetchGameDetails(pinnedID)
			if err != nil {
				log.Printf("Failed to fetch pinned game %d: %v", pinnedID, err)
				continue
			}
			if game != nil {
				game.IsPinned = true
				pinnedGames = append(pinnedGames, *game)
			}
		}
	}

	// Sort pinned games by their order in the config (not alphabetically)
	sort.Slice(pinnedGames, func(i, j int) bool {
		// Find index of each game in pinnedGameIDs
		indexI := -1
		indexJ := -1
		for idx, id := range pinnedGameIDs {
			if id == pinnedGames[i].AppID {
				indexI = idx
			}
			if id == pinnedGames[j].AppID {
				indexJ = idx
			}
		}
		return indexI < indexJ
	})

	return &models.GamesResponse{
		PinnedGames: pinnedGames,
		AllGames:    unpinnedGames,
	}, nil
}

// ownedGamesResponse represents Steam API response for owned games
type ownedGamesResponse struct {
	Response struct {
		GameCount int `json:"game_count"`
		Games     []struct {
			AppID           int    `json:"appid"`
			Name            string `json:"name"`
			PlaytimeForever int    `json:"playtime_forever"`
			ImgIconURL      string `json:"img_icon_url"`
		} `json:"games"`
	} `json:"response"`
}

// fetchUserGames fetches all games owned by a user
func (s *GameService) fetchUserGames(steamID string) ([]models.GameOwnership, error) {
	if s.cfg.SteamAPIKey == "" {
		return nil, fmt.Errorf("Steam API key not configured")
	}

	url := fmt.Sprintf(
		"%s/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s&include_appinfo=true&include_played_free_games=true",
		steamAPIBaseURL,
		s.cfg.SteamAPIKey,
		steamID,
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam API returned status %d", resp.StatusCode)
	}

	var apiResp ownedGamesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Steam API response: %w", err)
	}

	var games []models.GameOwnership
	for _, g := range apiResp.Response.Games {
		games = append(games, models.GameOwnership{
			SteamID:         steamID,
			AppID:           g.AppID,
			Name:            g.Name,
			PlaytimeForever: g.PlaytimeForever,
			IconURL:         g.ImgIconURL,
		})
	}

	return games, nil
}

// storeAppDetailsResponse represents Steam Store API response
type storeAppDetailsResponse map[string]struct {
	Success bool `json:"success"`
	Data    struct {
		Name       string `json:"name"`
		Categories []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
		} `json:"categories"`
	} `json:"data"`
}

// fetchGameCategories fetches categories for multiple games from Steam Store API
// Uses DB caching and respects rate limits
func (s *GameService) fetchGameCategories(games []*models.Game) {
	if len(games) == 0 {
		return
	}

	// Check if we're rate limited
	if s.isRateLimited() {
		log.Printf("Skipping Steam Store API calls - rate limited until %v", s.rateLimiter.pausedUntil)
		return
	}

	// Process sequentially with delays to avoid rate limits
	const delayBetweenRequests = 300 * time.Millisecond

	for _, game := range games {
		// Check rate limit before each request
		if s.isRateLimited() {
			log.Printf("Rate limit hit - stopping category fetches")
			return
		}

		name, categories, err := s.fetchGameCategoriesFromStore(game.AppID)
		if err != nil {
			log.Printf("Could not fetch categories for %s (%d): %v", game.Name, game.AppID, err)
			continue
		}

		game.Categories = categories
		if name != "" {
			game.Name = name
		}

		// Save to DB cache
		if err := s.gameCacheRepo.Upsert(game.AppID, game.Name, categories); err != nil {
			log.Printf("Failed to cache game %d: %v", game.AppID, err)
		}

		time.Sleep(delayBetweenRequests)
	}
}

// fetchGameCategoriesFromStore fetches categories for a single game from Steam Store
// Returns name, categories, and error. Handles 429 rate limiting.
func (s *GameService) fetchGameCategoriesFromStore(appID int) (string, []string, error) {
	url := fmt.Sprintf("%s/appdetails?appids=%d", steamStoreBaseURL, appID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("failed to call Steam Store API: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		s.setRateLimited()
		return "", nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("Steam Store API returned status %d", resp.StatusCode)
	}

	var apiResp storeAppDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", nil, fmt.Errorf("failed to parse Steam Store API response: %w", err)
	}

	appIDStr := fmt.Sprintf("%d", appID)
	appData, ok := apiResp[appIDStr]
	if !ok || !appData.Success {
		return "", nil, fmt.Errorf("game not found or not accessible")
	}

	var categories []string
	for _, cat := range appData.Data.Categories {
		categories = append(categories, cat.Description)
	}

	return appData.Data.Name, categories, nil
}

// fetchGameDetails fetches full details for a single game (used for pinned games not in library)
// First checks DB cache, then fetches from Steam Store API if needed
func (s *GameService) fetchGameDetails(appID int) (*models.Game, error) {
	// Cache image asynchronously
	s.imageCacheService.CacheImageAsync(appID)

	// First try DB cache
	cached, err := s.gameCacheRepo.GetByAppID(appID)
	if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
		return &models.Game{
			AppID:           appID,
			Name:            cached.Name,
			HeaderImageURL:  s.imageCacheService.GetLocalImageURL(appID),
			CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, appID),
			Categories:      cached.GetCategories(),
			OwnerCount:      0,
			Owners:          []string{},
		}, nil
	}

	// Check rate limit
	if s.isRateLimited() {
		// Return partial data from stale cache if available
		if cached != nil {
			return &models.Game{
				AppID:           appID,
				Name:            cached.Name,
				HeaderImageURL:  s.imageCacheService.GetLocalImageURL(appID),
				CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, appID),
				Categories:      cached.GetCategories(),
				OwnerCount:      0,
				Owners:          []string{},
			}, nil
		}
		return nil, fmt.Errorf("rate limited and no cache available")
	}

	// Fetch from Steam Store API
	name, categories, err := s.fetchGameCategoriesFromStore(appID)
	if err != nil {
		return nil, err
	}

	// Save to DB cache
	if err := s.gameCacheRepo.Upsert(appID, name, categories); err != nil {
		log.Printf("Failed to cache game %d: %v", appID, err)
	}

	return &models.Game{
		AppID:           appID,
		Name:            name,
		HeaderImageURL:  s.imageCacheService.GetLocalImageURL(appID),
		CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, appID),
		Categories:      categories,
		OwnerCount:      0,
		Owners:          []string{},
	}, nil
}

// GetPinnedGameIDs returns the list of pinned game IDs
func (s *GameService) GetPinnedGameIDs() []int {
	return s.cfg.PinnedGameIDs
}

// Helper function to check if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
