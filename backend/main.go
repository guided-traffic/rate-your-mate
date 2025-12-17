package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/rate-your-mate/backend/auth"
	"github.com/guided-traffic/rate-your-mate/backend/config"
	"github.com/guided-traffic/rate-your-mate/backend/database"
	"github.com/guided-traffic/rate-your-mate/backend/handlers"
	"github.com/guided-traffic/rate-your-mate/backend/middleware"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
	"github.com/guided-traffic/rate-your-mate/backend/services"
	"github.com/guided-traffic/rate-your-mate/backend/websocket"
)

// Version information - set via ldflags during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Global config
var cfg *config.Config

func main() {
	// Load configuration
	cfg = config.Load()
	log.Printf("Configuration loaded - Frontend: %s, Backend: %s", cfg.FrontendURL, cfg.BackendURL)

	// Check Steam connectivity at startup
	steamAPIClient := auth.NewSteamAPIClient(cfg.SteamAPIKey)
	if err := steamAPIClient.CheckConnectivity(); err != nil {
		log.Fatalf("Steam connectivity check failed: %v", err)
	}
	log.Println("Steam endpoints are reachable")

	// Initialize database based on configuration
	dbCfg := database.Config{
		Type:       database.DBType(cfg.DBType),
		SQLitePath: cfg.DBPath,
		MySQL: database.MySQLConfig{
			Host:            cfg.MySQLHost,
			Port:            cfg.MySQLPort,
			User:            cfg.MySQLUser,
			Password:        cfg.MySQLPassword,
			Database:        cfg.MySQLDatabase,
			TLSEnabled:      cfg.MySQLTLSEnabled,
			TLSSkipVerify:   cfg.MySQLTLSSkipVerify,
			TLSCACert:       cfg.MySQLTLSCACert,
			MaxOpenConns:    cfg.MySQLMaxOpenConns,
			MaxIdleConns:    cfg.MySQLMaxIdleConns,
			ConnMaxLifetime: cfg.MySQLConnMaxLifetime,
			ConnMaxIdleTime: cfg.MySQLConnMaxIdleTime,
		},
	}
	if err := database.Init(dbCfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	voteRepo := repository.NewVoteRepository()
	chatRepo := repository.NewChatRepository()
	gameCacheRepo := repository.NewGameCacheRepository()
	gameOwnerRepo := repository.NewGameOwnerRepository()

	// Initialize services
	creditService := services.NewCreditService(cfg, userRepo)
	imageCacheService := services.NewImageCacheService()
	avatarCacheService := services.NewAvatarCacheService(cfg.BackendURL)
	gameService := services.NewGameService(cfg, userRepo, gameCacheRepo, gameOwnerRepo, imageCacheService)
	countdownService := services.NewCountdownService(cfg, wsHub, userRepo)

	// Start countdown watcher
	countdownService.Start()
	defer countdownService.Stop()

	// Prefetch pinned games in background at startup
	gameService.PrefetchPinnedGames()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg, userRepo, creditService, gameService, avatarCacheService, wsHub)
	userHandler := handlers.NewUserHandler(userRepo, avatarCacheService)
	achievementHandler := handlers.NewAchievementHandler()
	voteHandler := handlers.NewVoteHandler(voteRepo, userRepo, creditService, wsHub, cfg)
	wsHandler := handlers.NewWebSocketHandler(wsHub, authHandler.GetJWTService())
	settingsHandler := handlers.NewSettingsHandler(cfg, wsHub, userRepo, voteRepo)
	chatHandler := handlers.NewChatHandler(chatRepo, userRepo, wsHub)
	gameHandler := handlers.NewGameHandler(gameService, imageCacheService, gameCacheRepo, userRepo, cfg, wsHub)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{cfg.FrontendURL}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Health check endpoint with version info
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"version":   Version,
			"buildTime": BuildTime,
			"gitCommit": GitCommit,
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Health check endpoint (also available at /health for backwards compatibility)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"version":   Version,
				"buildTime": BuildTime,
				"gitCommit": GitCommit,
			})
		})

		// Auth endpoints (public)
		auth := api.Group("/auth")
		{
			auth.GET("/steam", authHandler.SteamLogin)
			auth.GET("/steam/callback", authHandler.SteamCallback)
			auth.POST("/logout", authHandler.Logout)
		}

		// Achievements (public)
		api.GET("/achievements", achievementHandler.GetAll)
		api.GET("/achievements/:id", achievementHandler.GetByID)

		// Game images (public - allows caching by browsers/CDNs)
		api.GET("/games/images/:filename", gameHandler.ServeGameImage)

		// Avatar images (public - allows caching by browsers/CDNs)
		api.GET("/avatars/:filename", userHandler.ServeAvatar)

		// Public countdown endpoint (for login page)
		api.GET("/countdown", settingsHandler.GetCountdown)

		// WebSocket endpoint (token passed as query param, validates internally)
		api.GET("/ws", wsHandler.HandleConnection)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authHandler.GetJWTService()))
		{
			// Auth
			protected.GET("/auth/me", authHandler.Me)

			// WebSocket status (requires authentication)
			protected.GET("/ws/status", wsHandler.GetStatus)

			// Users
			protected.GET("/users", userHandler.GetAll)
			protected.GET("/users/others", userHandler.GetOthers)
			protected.GET("/users/:id", userHandler.GetByID)

			// Votes
			protected.POST("/votes", voteHandler.Create)
			protected.GET("/votes", voteHandler.GetTimeline)

			// Chat
			protected.GET("/chat", chatHandler.GetMessages)
			protected.POST("/chat", chatHandler.Create)

			// Voting status (for authenticated users)
			protected.GET("/voting-status", settingsHandler.GetVotingStatus)

			// Leaderboard
			protected.GET("/leaderboard", voteHandler.GetLeaderboard)
			protected.GET("/champions", voteHandler.GetChampions)

			// Global Ranking
			protected.GET("/ranking", voteHandler.GetGlobalRanking)
			protected.GET("/ranking/me", voteHandler.GetMyRanking)

			// Games
			protected.GET("/games", gameHandler.GetMultiplayerGames)
			protected.POST("/games/refresh", gameHandler.RefreshGames)
			protected.POST("/games/refresh-my-games", gameHandler.RefreshMyGames)
			protected.POST("/games/sync", gameHandler.StartBackgroundSync)
			protected.GET("/games/sync/status", gameHandler.GetSyncStatus)

			// Admin routes (require admin privileges)
			admin := protected.Group("/admin")
			admin.Use(settingsHandler.AdminMiddleware())
			{
				admin.GET("/password-required", settingsHandler.CheckAdminPasswordRequired)
				admin.POST("/verify-password", settingsHandler.VerifyAdminPassword)
				admin.GET("/settings", settingsHandler.GetSettings)
				admin.PUT("/settings", settingsHandler.UpdateSettings)
				admin.POST("/credits/reset", settingsHandler.ResetAllCredits)
				admin.POST("/credits/give", settingsHandler.GiveEveryoneCredit)
				admin.POST("/votes/delete-all", settingsHandler.DeleteAllVotes)
				admin.POST("/games/invalidate-cache", gameHandler.InvalidateDBCache)
				// Vote management
				admin.PUT("/votes/:id/invalidate", voteHandler.ToggleInvalidation)
				// User management
				admin.GET("/users", settingsHandler.GetAllUsersForAdmin)
				admin.GET("/users/banned", settingsHandler.GetAllBannedUsers)
				admin.POST("/users/:id/kick", settingsHandler.KickUser)
				admin.POST("/users/:id/ban", settingsHandler.BanUser)
				admin.POST("/users/unban/:steam_id", settingsHandler.UnbanUser)
			}
		}
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
