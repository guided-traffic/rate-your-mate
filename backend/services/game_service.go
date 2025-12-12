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

	"github.com/guided-traffic/rate-your-mate/backend/config"
	"github.com/guided-traffic/rate-your-mate/backend/models"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
)

const (
	steamAPIBaseURL   = "https://api.steampowered.com"
	steamStoreBaseURL = "https://store.steampowered.com/api"
	steamCDNBaseURL   = "https://steamcdn-a.akamaihd.net/steam/apps"

	// Cache settings
	gameCacheMaxAge       = 24 * time.Hour  // Refresh game data after 24 hours
	failedFetchRetryDelay = 24 * time.Hour  // Wait 24 hours before retrying failed fetches (e.g., removed games)
	rateLimitPausePeriod  = 5 * time.Minute // Pause for 5 minutes after 429 error
)

// SyncProgressCallback is called to report sync progress
type SyncProgressCallback func(phase string, currentGame string, processed, total int)

// GameService handles game-related operations
type GameService struct {
	cfg               *config.Config
	userRepo          *repository.UserRepository
	gameCacheRepo     *repository.GameCacheRepository
	imageCacheService *ImageCacheService
	httpClient        *http.Client
	cache             *gamesCache
	rateLimiter       *rateLimiter
	syncProgress      *syncProgress
}

// syncProgress tracks background sync status
type syncProgress struct {
	mu        sync.RWMutex
	isSyncing bool
	phase     string
	current   string
	processed int
	total     int
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
		cache:        &gamesCache{},
		rateLimiter:  &rateLimiter{},
		syncProgress: &syncProgress{},
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

					// Note: Image caching is deferred until after multiplayer filtering

					// Try to load categories and price from DB cache
					cached, err := s.gameCacheRepo.GetByAppID(g.AppID)
					if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
						game.Categories = cached.GetCategories()
						if cached.Name != "" {
							game.Name = cached.Name
						}
						game.IsFree = cached.IsFree
						game.PriceCents = cached.PriceCents
						game.OriginalCents = cached.OriginalCents
						game.DiscountPercent = cached.DiscountPercent
						game.PriceFormatted = cached.PriceFormatted
						game.ReviewScore = cached.ReviewScore
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
				// Check if this was a failed fetch that we should skip
				if cached.FetchFailed {
					// Skip games that failed to fetch and are still within retry delay
					if !cached.IsStale(failedFetchRetryDelay) {
						log.Printf("Skipping unavailable game %s (%d) - will retry after %v", game.Name, game.AppID, failedFetchRetryDelay)
						continue
					}
					// Retry delay expired, try fetching again
					gamesToFetch = append(gamesToFetch, game)
				} else {
					game.Categories = cached.GetCategories()
					if cached.Name != "" {
						game.Name = cached.Name
					}
					game.IsFree = cached.IsFree
					game.PriceCents = cached.PriceCents
					game.OriginalCents = cached.OriginalCents
					game.DiscountPercent = cached.DiscountPercent
					game.PriceFormatted = cached.PriceFormatted
					game.ReviewScore = cached.ReviewScore
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
			// Only cache images for multiplayer games (after filtering)
			s.imageCacheService.CacheImageAsync(game.AppID)

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
	// Skip fake users (used for development/testing)
	if strings.HasPrefix(steamID, "FAKE_") {
		log.Printf("Skipping Steam API call for fake user: %s", steamID)
		return []models.GameOwnership{}, nil
	}

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
		Name        string `json:"name"`
		HeaderImage string `json:"header_image"`
		IsFree      bool   `json:"is_free"`
		Categories  []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
		} `json:"categories"`
		PriceOverview *struct {
			Currency         string `json:"currency"`
			Initial          int    `json:"initial"`
			Final            int    `json:"final"`
			DiscountPercent  int    `json:"discount_percent"`
			InitialFormatted string `json:"initial_formatted"`
			FinalFormatted   string `json:"final_formatted"`
		} `json:"price_overview"`
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

		storeData, err := s.fetchGameCategoriesFromStore(game.AppID)
		if err != nil {
			log.Printf("Could not fetch data for %s (%d): %v", game.Name, game.AppID, err)

			// Check if this is a "game not found" error (not a rate limit or network error)
			// Cache the failure so we don't retry for 24 hours
			if strings.Contains(err.Error(), "game not found") || strings.Contains(err.Error(), "not accessible") {
				log.Printf("Game %s (%d) appears to be unavailable (removed from Steam Store?) - caching failure for %v", game.Name, game.AppID, failedFetchRetryDelay)
				if cacheErr := s.gameCacheRepo.UpsertWithStatus(game.AppID, game.Name, []string{}, nil, true); cacheErr != nil {
					log.Printf("Failed to cache failed fetch for game %d: %v", game.AppID, cacheErr)
				}
			}
			continue
		}

		game.Categories = storeData.Categories
		if storeData.Name != "" {
			game.Name = storeData.Name
		}
		game.IsFree = storeData.IsFree
		game.PriceCents = storeData.PriceCents
		game.OriginalCents = storeData.OriginalCents
		game.DiscountPercent = storeData.DiscountPercent
		game.PriceFormatted = storeData.PriceFormatted
		game.ReviewScore = storeData.ReviewScore

		// Cache image using the header_image URL from Steam API
		if storeData.HeaderImageURL != "" {
			s.imageCacheService.CacheImageFromURLAsync(game.AppID, storeData.HeaderImageURL)
		}

		// Save to DB cache
		priceInfo := &repository.GamePriceInfo{
			IsFree:          storeData.IsFree,
			PriceCents:      storeData.PriceCents,
			OriginalCents:   storeData.OriginalCents,
			DiscountPercent: storeData.DiscountPercent,
			PriceFormatted:  storeData.PriceFormatted,
			ReviewScore:     storeData.ReviewScore,
		}
		if err := s.gameCacheRepo.Upsert(game.AppID, game.Name, storeData.Categories, priceInfo); err != nil {
			log.Printf("Failed to cache game %d: %v", game.AppID, err)
		}

		time.Sleep(delayBetweenRequests)
	}
}

// fetchGameCategoriesFromStore fetches categories and price for a single game from Steam Store
// Returns GameStoreData and error. Handles 429 rate limiting.
func (s *GameService) fetchGameCategoriesFromStore(appID int) (*GameStoreData, error) {
	url := fmt.Sprintf("%s/appdetails?appids=%d&cc=de", steamStoreBaseURL, appID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call Steam Store API: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		s.setRateLimited()
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam Store API returned status %d", resp.StatusCode)
	}

	var apiResp storeAppDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Steam Store API response: %w", err)
	}

	appIDStr := fmt.Sprintf("%d", appID)
	appData, ok := apiResp[appIDStr]
	if !ok || !appData.Success {
		return nil, fmt.Errorf("game not found or not accessible")
	}

	var categories []string
	for _, cat := range appData.Data.Categories {
		categories = append(categories, cat.Description)
	}

	// Build price info
	data := &GameStoreData{
		Name:           appData.Data.Name,
		HeaderImageURL: appData.Data.HeaderImage,
		Categories:     categories,
		IsFree:         appData.Data.IsFree,
	}

	if appData.Data.IsFree {
		data.PriceFormatted = "Free"
	} else if appData.Data.PriceOverview != nil {
		data.PriceCents = appData.Data.PriceOverview.Final
		data.OriginalCents = appData.Data.PriceOverview.Initial
		data.DiscountPercent = appData.Data.PriceOverview.DiscountPercent
		data.PriceFormatted = appData.Data.PriceOverview.FinalFormatted
	}

	// Fetch review score from Steam Review API
	data.ReviewScore = s.fetchGameReviewScore(appID)

	return data, nil
}

// GameStoreData contains all data fetched from Steam Store API
type GameStoreData struct {
	Name            string
	HeaderImageURL  string
	Categories      []string
	IsFree          bool
	PriceCents      int
	OriginalCents   int
	DiscountPercent int
	PriceFormatted  string
	ReviewScore     int // Percentage of positive reviews (0-100), -1 if not enough reviews
}

// steamReviewResponse represents the Steam Review API response
type steamReviewResponse struct {
	Success      int `json:"success"`
	QuerySummary struct {
		NumReviews      int    `json:"num_reviews"`
		ReviewScore     int    `json:"review_score"`
		ReviewScoreDesc string `json:"review_score_desc"`
		TotalPositive   int    `json:"total_positive"`
		TotalNegative   int    `json:"total_negative"`
		TotalReviews    int    `json:"total_reviews"`
	} `json:"query_summary"`
}

// fetchGameReviewScore fetches the review score percentage from Steam Review API
// Returns the percentage of positive reviews (0-100), or -1 if not enough reviews
func (s *GameService) fetchGameReviewScore(appID int) int {
	url := fmt.Sprintf("https://store.steampowered.com/appreviews/%d?json=1&purchase_type=all&language=all", appID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		log.Printf("Failed to fetch reviews for game %d: %v", appID, err)
		return -1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Steam Review API returned status %d for game %d", resp.StatusCode, appID)
		return -1
	}

	var reviewResp steamReviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&reviewResp); err != nil {
		log.Printf("Failed to parse review response for game %d: %v", appID, err)
		return -1
	}

	if reviewResp.Success != 1 {
		log.Printf("Steam Review API returned unsuccessful for game %d", appID)
		return -1
	}

	totalReviews := reviewResp.QuerySummary.TotalPositive + reviewResp.QuerySummary.TotalNegative
	if totalReviews < 10 {
		// Not enough reviews for a meaningful percentage
		return -1
	}

	// Calculate percentage of positive reviews
	percentage := (reviewResp.QuerySummary.TotalPositive * 100) / totalReviews
	return percentage
}

// fetchGameDetails fetches full details for a single game (used for pinned games not in library)
// First checks DB cache, then fetches from Steam Store API if needed
func (s *GameService) fetchGameDetails(appID int) (*models.Game, error) {
	// First try DB cache
	cached, err := s.gameCacheRepo.GetByAppID(appID)
	if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
		// Check if this was a failed fetch - skip unavailable games
		if cached.FetchFailed {
			if !cached.IsStale(failedFetchRetryDelay) {
				log.Printf("Skipping unavailable pinned game (%d) - will retry after %v", appID, failedFetchRetryDelay)
				return nil, fmt.Errorf("game unavailable (cached failure)")
			}
			// Retry delay expired, continue to fetch again below
		} else {
			// For cached games, try to cache image asynchronously using old CDN URL as fallback
			s.imageCacheService.CacheImageAsync(appID)
			return &models.Game{
				AppID:           appID,
				Name:            cached.Name,
				HeaderImageURL:  s.imageCacheService.GetLocalImageURL(appID),
				CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, appID),
				Categories:      cached.GetCategories(),
				OwnerCount:      0,
				Owners:          []string{},
				IsFree:          cached.IsFree,
				PriceCents:      cached.PriceCents,
				OriginalCents:   cached.OriginalCents,
				DiscountPercent: cached.DiscountPercent,
				PriceFormatted:  cached.PriceFormatted,
				ReviewScore:     cached.ReviewScore,
			}, nil
		}
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
				IsFree:          cached.IsFree,
				PriceCents:      cached.PriceCents,
				OriginalCents:   cached.OriginalCents,
				DiscountPercent: cached.DiscountPercent,
				PriceFormatted:  cached.PriceFormatted,
				ReviewScore:     cached.ReviewScore,
			}, nil
		}
		return nil, fmt.Errorf("rate limited and no cache available")
	}

	// Fetch from Steam Store API
	storeData, err := s.fetchGameCategoriesFromStore(appID)
	if err != nil {
		// Cache the failure if it's a "game not found" error
		if strings.Contains(err.Error(), "game not found") || strings.Contains(err.Error(), "not accessible") {
			log.Printf("Pinned game (%d) appears to be unavailable - caching failure for %v", appID, failedFetchRetryDelay)
			if cacheErr := s.gameCacheRepo.UpsertWithStatus(appID, fmt.Sprintf("Unknown Game %d", appID), []string{}, nil, true); cacheErr != nil {
				log.Printf("Failed to cache failed fetch for pinned game %d: %v", appID, cacheErr)
			}
		}
		return nil, err
	}

	// Cache image using the header_image URL from Steam API
	if storeData.HeaderImageURL != "" {
		s.imageCacheService.CacheImageFromURLAsync(appID, storeData.HeaderImageURL)
	}

	// Save to DB cache
	priceInfo := &repository.GamePriceInfo{
		IsFree:          storeData.IsFree,
		PriceCents:      storeData.PriceCents,
		OriginalCents:   storeData.OriginalCents,
		DiscountPercent: storeData.DiscountPercent,
		PriceFormatted:  storeData.PriceFormatted,
		ReviewScore:     storeData.ReviewScore,
	}
	if err := s.gameCacheRepo.Upsert(appID, storeData.Name, storeData.Categories, priceInfo); err != nil {
		log.Printf("Failed to cache game %d: %v", appID, err)
	}

	return &models.Game{
		AppID:           appID,
		Name:            storeData.Name,
		HeaderImageURL:  s.imageCacheService.GetLocalImageURL(appID),
		CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, appID),
		Categories:      storeData.Categories,
		OwnerCount:      0,
		Owners:          []string{},
		IsFree:          storeData.IsFree,
		PriceCents:      storeData.PriceCents,
		OriginalCents:   storeData.OriginalCents,
		DiscountPercent: storeData.DiscountPercent,
		PriceFormatted:  storeData.PriceFormatted,
		ReviewScore:     storeData.ReviewScore,
	}, nil
}

// GetPinnedGameIDs returns the list of pinned game IDs
func (s *GameService) GetPinnedGameIDs() []int {
	return s.cfg.PinnedGameIDs
}

// PrefetchPinnedGames fetches and caches pinned games at startup
// This runs in the background and doesn't block startup
func (s *GameService) PrefetchPinnedGames() {
	pinnedIDs := s.cfg.PinnedGameIDs
	if len(pinnedIDs) == 0 {
		log.Println("[GameSync] No pinned games configured")
		return
	}

	log.Printf("[GameSync] Prefetching %d pinned games in background...", len(pinnedIDs))

	go func() {
		const delayBetweenRequests = 300 * time.Millisecond
		fetched := 0
		skipped := 0

		for _, appID := range pinnedIDs {
			// Check if already in cache
			cached, err := s.gameCacheRepo.GetByAppID(appID)
			if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) && !cached.FetchFailed {
				log.Printf("[GameSync] Pinned game %d already cached: %s", appID, cached.Name)
				skipped++
				continue
			}

			// Check rate limit
			if s.isRateLimited() {
				log.Printf("[GameSync] Rate limited - stopping pinned game prefetch")
				break
			}

			// Fetch from Steam Store API
			storeData, err := s.fetchGameCategoriesFromStore(appID)
			if err != nil {
				log.Printf("[GameSync] Failed to prefetch pinned game %d: %v", appID, err)
				continue
			}

			// Cache the data
			priceInfo := &repository.GamePriceInfo{
				IsFree:          storeData.IsFree,
				PriceCents:      storeData.PriceCents,
				OriginalCents:   storeData.OriginalCents,
				DiscountPercent: storeData.DiscountPercent,
				PriceFormatted:  storeData.PriceFormatted,
				ReviewScore:     storeData.ReviewScore,
			}
			if err := s.gameCacheRepo.Upsert(appID, storeData.Name, storeData.Categories, priceInfo); err != nil {
				log.Printf("[GameSync] Failed to cache pinned game %d: %v", appID, err)
			}

			// Cache image
			if storeData.HeaderImageURL != "" {
				s.imageCacheService.CacheImageFromURLAsync(appID, storeData.HeaderImageURL)
			}

			log.Printf("[GameSync] Prefetched pinned game %d: %s", appID, storeData.Name)
			fetched++

			time.Sleep(delayBetweenRequests)
		}

		log.Printf("[GameSync] Pinned games prefetch complete: %d fetched, %d already cached", fetched, skipped)
	}()
}

// GetSyncStatus returns the current sync status
func (s *GameService) GetSyncStatus() (isSyncing bool, phase string, current string, processed, total int) {
	s.syncProgress.mu.RLock()
	defer s.syncProgress.mu.RUnlock()
	return s.syncProgress.isSyncing, s.syncProgress.phase, s.syncProgress.current, s.syncProgress.processed, s.syncProgress.total
}

// IsSyncing returns whether a background sync is in progress
func (s *GameService) IsSyncing() bool {
	s.syncProgress.mu.RLock()
	defer s.syncProgress.mu.RUnlock()
	return s.syncProgress.isSyncing
}

// setSyncProgress updates the sync progress
func (s *GameService) setSyncProgress(isSyncing bool, phase, current string, processed, total int) {
	s.syncProgress.mu.Lock()
	s.syncProgress.isSyncing = isSyncing
	s.syncProgress.phase = phase
	s.syncProgress.current = current
	s.syncProgress.processed = processed
	s.syncProgress.total = total
	s.syncProgress.mu.Unlock()
}

// GetMultiplayerGamesCached returns only cached games without triggering a sync
// This is fast and returns immediately
func (s *GameService) GetMultiplayerGamesCached() (*models.GamesResponse, bool, error) {
	// Check in-memory cache first
	s.cache.mu.RLock()
	if s.cache.games != nil && time.Now().Before(s.cache.expiresAt) {
		cached := s.cache.games
		s.cache.mu.RUnlock()
		return cached, false, nil // false = no sync needed
	}
	s.cache.mu.RUnlock()

	// Try to build response from DB cache only (no Steam API calls)
	games, needsSync, err := s.buildGamesFromCache()
	if err != nil {
		return nil, needsSync, err
	}

	// Update in-memory cache
	s.cache.mu.Lock()
	s.cache.games = games
	s.cache.expiresAt = time.Now().Add(5 * time.Minute)
	s.cache.mu.Unlock()

	return games, needsSync, nil
}

// buildGamesFromCache builds the games response using only DB-cached data
func (s *GameService) buildGamesFromCache() (*models.GamesResponse, bool, error) {
	users, err := s.userRepo.GetAll()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get users: %w", err)
	}

	pinnedGameIDs := s.cfg.PinnedGameIDs
	needsSync := false

	// Even with no users, we still want to show pinned games
	if len(users) == 0 {
		log.Printf("[GameSync] No users, loading pinned games only")
		pinnedGames := s.loadPinnedGamesFromCache(&needsSync)
		return &models.GamesResponse{
			PinnedGames: pinnedGames,
			AllGames:    []models.Game{},
		}, needsSync, nil
	}

	log.Printf("[GameSync] Building games from cache for %d users", len(users))

	// Collect all games from all users (this is fast - just Steam API call)
	gameMap := make(map[int]*models.Game)
	var mu sync.Mutex
	var wg sync.WaitGroup
	needsSync = false

	for _, user := range users {
		wg.Add(1)
		go func(steamID string) {
			defer wg.Done()

			log.Printf("[GameSync] Fetching owned games for user %s", steamID)
			games, err := s.fetchUserGames(steamID)
			if err != nil {
				log.Printf("[GameSync] Failed to fetch games for user %s: %v", steamID, err)
				return
			}
			log.Printf("[GameSync] User %s has %d games", steamID, len(games))

			mu.Lock()
			for _, g := range games {
				if existing, ok := gameMap[g.AppID]; ok {
					existing.OwnerCount++
					existing.Owners = append(existing.Owners, steamID)
					if g.PlaytimeForever > existing.PlaytimeForever {
						existing.PlaytimeForever = g.PlaytimeForever
					}
				} else {
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

					// Try to load from DB cache
					cached, err := s.gameCacheRepo.GetByAppID(g.AppID)
					if err == nil && cached != nil && !cached.IsStale(gameCacheMaxAge) {
						game.Categories = cached.GetCategories()
						if cached.Name != "" {
							game.Name = cached.Name
						}
						game.IsFree = cached.IsFree
						game.PriceCents = cached.PriceCents
						game.OriginalCents = cached.OriginalCents
						game.DiscountPercent = cached.DiscountPercent
						game.PriceFormatted = cached.PriceFormatted
						game.ReviewScore = cached.ReviewScore
					} else {
						needsSync = true // At least one game needs category fetch
					}

					gameMap[g.AppID] = game
				}
			}
			mu.Unlock()
		}(user.SteamID)
	}

	wg.Wait()

	log.Printf("[GameSync] Total unique games from all users: %d, needsSync: %v", len(gameMap), needsSync)

	// Filter for multiplayer games and build response
	var allGames []models.Game

	for _, game := range gameMap {
		if game.HasMultiplayerCategory() {
			s.imageCacheService.CacheImageAsync(game.AppID)
			for _, pinnedID := range pinnedGameIDs {
				if pinnedID == game.AppID {
					game.IsPinned = true
					break
				}
			}
			allGames = append(allGames, *game)
		}
	}

	log.Printf("[GameSync] After multiplayer filter: %d games", len(allGames))

	// Sort all games by owner count, then by name
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

	// Add pinned games that might not be in any user's library
	for _, pinnedID := range pinnedGameIDs {
		found := false
		for _, g := range pinnedGames {
			if g.AppID == pinnedID {
				found = true
				break
			}
		}
		if !found {
			// Try to load from cache first
			cached, err := s.gameCacheRepo.GetByAppID(pinnedID)
			if err == nil && cached != nil && !cached.FetchFailed {
				game := models.Game{
					AppID:           pinnedID,
					Name:            cached.Name,
					HeaderImageURL:  s.imageCacheService.GetLocalImageURL(pinnedID),
					CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, pinnedID),
					Categories:      cached.GetCategories(),
					OwnerCount:      0,
					Owners:          []string{},
					IsPinned:        true,
					IsFree:          cached.IsFree,
					PriceCents:      cached.PriceCents,
					OriginalCents:   cached.OriginalCents,
					DiscountPercent: cached.DiscountPercent,
					PriceFormatted:  cached.PriceFormatted,
					ReviewScore:     cached.ReviewScore,
				}
				pinnedGames = append(pinnedGames, game)
			} else {
				needsSync = true // Pinned game needs to be fetched
			}
		}
	}

	// Sort pinned games by config order
	sort.Slice(pinnedGames, func(i, j int) bool {
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
	}, needsSync, nil
}

// loadPinnedGamesFromCache loads pinned games from DB cache
func (s *GameService) loadPinnedGamesFromCache(needsSync *bool) []models.Game {
	pinnedGameIDs := s.cfg.PinnedGameIDs
	var pinnedGames []models.Game

	for _, pinnedID := range pinnedGameIDs {
		cached, err := s.gameCacheRepo.GetByAppID(pinnedID)
		if err == nil && cached != nil && !cached.FetchFailed {
			game := models.Game{
				AppID:           pinnedID,
				Name:            cached.Name,
				HeaderImageURL:  s.imageCacheService.GetLocalImageURL(pinnedID),
				CapsuleImageURL: fmt.Sprintf("%s/%d/capsule_231x87.jpg", steamCDNBaseURL, pinnedID),
				Categories:      cached.GetCategories(),
				OwnerCount:      0,
				Owners:          []string{},
				IsPinned:        true,
				IsFree:          cached.IsFree,
				PriceCents:      cached.PriceCents,
				OriginalCents:   cached.OriginalCents,
				DiscountPercent: cached.DiscountPercent,
				PriceFormatted:  cached.PriceFormatted,
				ReviewScore:     cached.ReviewScore,
			}
			pinnedGames = append(pinnedGames, game)
			log.Printf("[GameSync] Loaded pinned game from cache: %s (%d)", cached.Name, pinnedID)
		} else {
			log.Printf("[GameSync] Pinned game %d not in cache, needs sync", pinnedID)
			*needsSync = true
		}
	}

	return pinnedGames
}

// SyncGames triggers a sync for all games that need updating
// This can be called at any time - it checks the DB for games with stale/missing data
func (s *GameService) SyncGames(progressCallback SyncProgressCallback) {
	s.TriggerSyncIfNeeded(progressCallback)
}

// RegisterUserGames records a user's games in the cache and triggers sync if needed
// This is called when a new user registers - their games are added to the DB
// and a sync is triggered to fetch missing data
func (s *GameService) RegisterUserGames(steamID string, progressCallback SyncProgressCallback) {
	go func() {
		log.Printf("GameService: Registering games for new user %s", steamID)

		// Fetch new user's game library from Steam
		userGames, err := s.fetchUserGames(steamID)
		if err != nil {
			log.Printf("GameService: Failed to fetch games for new user %s: %v", steamID, err)
			return
		}

		if len(userGames) == 0 {
			log.Printf("GameService: New user %s has no games", steamID)
			return
		}

		log.Printf("GameService: New user %s has %d games, inserting into cache", steamID, len(userGames))

		// Insert all games into cache (without overwriting existing data)
		// Games that already exist will be skipped
		newGames := 0
		for _, g := range userGames {
			err := s.gameCacheRepo.InsertIfNotExists(g.AppID, g.Name)
			if err != nil {
				log.Printf("GameService: Failed to insert game %d: %v", g.AppID, err)
			} else {
				newGames++
			}
		}

		log.Printf("GameService: Inserted %d games for user %s", newGames, steamID)

		// Invalidate response cache so new user's ownership is reflected
		s.InvalidateCache()

		// Now trigger a sync to fetch missing data
		s.TriggerSyncIfNeeded(progressCallback)
	}()
}

// TriggerSyncIfNeeded checks if there are games that need syncing and starts a sync
func (s *GameService) TriggerSyncIfNeeded(progressCallback SyncProgressCallback) {
	// Check if sync is already running
	s.syncProgress.mu.RLock()
	isSyncing := s.syncProgress.isSyncing
	s.syncProgress.mu.RUnlock()

	if isSyncing {
		log.Println("GameService: Sync already in progress, skipping")
		return
	}

	// Check if there are games needing sync
	count, err := s.gameCacheRepo.CountGamesNeedingSync(gameCacheMaxAge, failedFetchRetryDelay)
	if err != nil {
		log.Printf("GameService: Failed to count games needing sync: %v", err)
		return
	}

	if count == 0 {
		log.Println("GameService: No games need syncing")
		if progressCallback != nil {
			progressCallback("complete", "", 0, 0)
		}
		return
	}

	log.Printf("GameService: %d games need syncing, starting sync", count)
	s.runSync(progressCallback)
}

// runSync performs the actual sync work
func (s *GameService) runSync(progressCallback SyncProgressCallback) {
	// Set syncing state
	s.syncProgress.mu.Lock()
	if s.syncProgress.isSyncing {
		s.syncProgress.mu.Unlock()
		log.Println("GameService: Sync already in progress, skipping")
		return
	}
	s.syncProgress.isSyncing = true
	s.syncProgress.mu.Unlock()

	go func() {
		defer func() {
			s.setSyncProgress(false, "", "", 0, 0)
		}()

		log.Println("GameService: Starting sync")

		// Get all games that need syncing
		gamesToSync, err := s.gameCacheRepo.GetGamesNeedingSync(gameCacheMaxAge, failedFetchRetryDelay)
		if err != nil {
			log.Printf("GameService: Failed to get games needing sync: %v", err)
			return
		}

		if len(gamesToSync) == 0 {
			log.Println("GameService: No games to sync")
			if progressCallback != nil {
				progressCallback("complete", "", 0, 0)
			}
			return
		}

		// Convert to models.Game for the fetch function
		var games []*models.Game
		for _, g := range gamesToSync {
			games = append(games, &models.Game{
				AppID: g.AppID,
				Name:  g.Name,
			})
		}

		totalToFetch := len(games)
		log.Printf("GameService: Syncing %d games", totalToFetch)

		s.setSyncProgress(true, "fetching_categories", "", 0, totalToFetch)
		if progressCallback != nil {
			progressCallback("fetching_categories", "", 0, totalToFetch)
		}

		// Fetch game data with progress reporting
		s.fetchGameCategoriesWithProgress(games, func(processed int, currentGame string) {
			s.setSyncProgress(true, "fetching_categories", currentGame, processed, totalToFetch)
			if progressCallback != nil {
				progressCallback("fetching_categories", currentGame, processed, totalToFetch)
			}
		})

		// Invalidate response cache
		s.InvalidateCache()

		// Count multiplayer games
		multiplayerCount := 0
		for _, game := range games {
			if game.HasMultiplayerCategory() {
				multiplayerCount++
			}
		}

		log.Printf("GameService: Sync batch complete. Synced %d games (%d multiplayer)", totalToFetch, multiplayerCount)

		// Check if there are more games to sync (new users may have joined during sync)
		remainingCount, err := s.gameCacheRepo.CountGamesNeedingSync(gameCacheMaxAge, failedFetchRetryDelay)
		if err != nil {
			log.Printf("GameService: Failed to count remaining games: %v", err)
		} else if remainingCount > 0 {
			log.Printf("GameService: %d more games need syncing, continuing...", remainingCount)
			// Reset sync state and continue
			s.syncProgress.mu.Lock()
			s.syncProgress.isSyncing = false
			s.syncProgress.mu.Unlock()
			// Recursive call to sync remaining games
			s.runSync(progressCallback)
			return
		}

		log.Println("GameService: All games synced")
		if progressCallback != nil {
			progressCallback("complete", "", multiplayerCount, totalToFetch)
		}
	}()
}

// fetchGameCategoriesWithProgress fetches categories with progress callback
func (s *GameService) fetchGameCategoriesWithProgress(games []*models.Game, progressCallback func(processed int, currentGame string)) {
	if len(games) == 0 {
		return
	}

	if s.isRateLimited() {
		log.Printf("Skipping Steam Store API calls - rate limited until %v", s.rateLimiter.pausedUntil)
		return
	}

	const delayBetweenRequests = 300 * time.Millisecond

	for i, game := range games {
		if s.isRateLimited() {
			log.Printf("Rate limit hit - stopping category fetches")
			return
		}

		if progressCallback != nil {
			progressCallback(i, game.Name)
		}

		storeData, err := s.fetchGameCategoriesFromStore(game.AppID)
		if err != nil {
			log.Printf("Could not fetch data for %s (%d): %v", game.Name, game.AppID, err)

			if strings.Contains(err.Error(), "game not found") || strings.Contains(err.Error(), "not accessible") {
				log.Printf("Game %s (%d) appears to be unavailable - caching failure", game.Name, game.AppID)
				if cacheErr := s.gameCacheRepo.UpsertWithStatus(game.AppID, game.Name, []string{}, nil, true); cacheErr != nil {
					log.Printf("Failed to cache failed fetch for game %d: %v", game.AppID, cacheErr)
				}
			}
			continue
		}

		game.Categories = storeData.Categories
		if storeData.Name != "" {
			game.Name = storeData.Name
		}
		game.IsFree = storeData.IsFree
		game.PriceCents = storeData.PriceCents
		game.OriginalCents = storeData.OriginalCents
		game.DiscountPercent = storeData.DiscountPercent
		game.PriceFormatted = storeData.PriceFormatted
		game.ReviewScore = storeData.ReviewScore

		// Save to DB cache
		priceInfo := &repository.GamePriceInfo{
			IsFree:          storeData.IsFree,
			PriceCents:      storeData.PriceCents,
			OriginalCents:   storeData.OriginalCents,
			DiscountPercent: storeData.DiscountPercent,
			PriceFormatted:  storeData.PriceFormatted,
			ReviewScore:     storeData.ReviewScore,
		}
		if err := s.gameCacheRepo.Upsert(game.AppID, game.Name, storeData.Categories, priceInfo); err != nil {
			log.Printf("Failed to cache game %d: %v", game.AppID, err)
		}

		time.Sleep(delayBetweenRequests)
	}

	if progressCallback != nil {
		progressCallback(len(games), "")
	}
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
