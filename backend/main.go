package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/lan-party-manager/backend/config"
	"github.com/guided-traffic/lan-party-manager/backend/database"
	"github.com/guided-traffic/lan-party-manager/backend/handlers"
	"github.com/guided-traffic/lan-party-manager/backend/middleware"
	"github.com/guided-traffic/lan-party-manager/backend/repository"
	"github.com/guided-traffic/lan-party-manager/backend/services"
	"github.com/guided-traffic/lan-party-manager/backend/websocket"
)

// Global config
var cfg *config.Config

func main() {
	// Load configuration
	cfg = config.Load()
	log.Printf("Configuration loaded - Frontend: %s, Backend: %s", cfg.FrontendURL, cfg.BackendURL)

	// Initialize database
	if err := database.Init("data/lan-party.db"); err != nil {
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

	// Initialize services
	creditService := services.NewCreditService(cfg, userRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg, userRepo, creditService)
	userHandler := handlers.NewUserHandler(userRepo)
	achievementHandler := handlers.NewAchievementHandler()
	voteHandler := handlers.NewVoteHandler(voteRepo, userRepo, creditService, wsHub, cfg)
	wsHandler := handlers.NewWebSocketHandler(wsHub, authHandler.GetJWTService())
	settingsHandler := handlers.NewSettingsHandler(cfg, wsHub, userRepo)
	chatHandler := handlers.NewChatHandler(chatRepo, userRepo, wsHub)

	r := gin.Default()

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{cfg.FrontendURL}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
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

		// WebSocket endpoint (token passed as query param)
		api.GET("/ws", wsHandler.HandleConnection)
		api.GET("/ws/status", wsHandler.GetStatus)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authHandler.GetJWTService()))
		{
			// Auth
			protected.GET("/auth/me", authHandler.Me)

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

			// Voting status (public for authenticated users)
			protected.GET("/voting-status", settingsHandler.GetVotingStatus)

			// Leaderboard
			protected.GET("/leaderboard", voteHandler.GetLeaderboard)

			// Admin routes (require admin privileges)
			admin := protected.Group("/admin")
			admin.Use(settingsHandler.AdminMiddleware())
			{
				admin.GET("/settings", settingsHandler.GetSettings)
				admin.PUT("/settings", settingsHandler.UpdateSettings)
				admin.POST("/credits/reset", settingsHandler.ResetAllCredits)
				admin.POST("/credits/give", settingsHandler.GiveEveryoneCredit)
			}
		}
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
