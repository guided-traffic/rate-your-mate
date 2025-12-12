package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guided-traffic/rate-your-mate/backend/config"
	"github.com/guided-traffic/rate-your-mate/backend/middleware"
	"github.com/guided-traffic/rate-your-mate/backend/models"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
	"github.com/guided-traffic/rate-your-mate/backend/services"
	"github.com/guided-traffic/rate-your-mate/backend/websocket"
)

// VoteHandler handles vote-related endpoints
type VoteHandler struct {
	voteRepo      *repository.VoteRepository
	userRepo      *repository.UserRepository
	creditService *services.CreditService
	wsHub         *websocket.Hub
	cfg           *config.Config
}

// NewVoteHandler creates a new vote handler
func NewVoteHandler(voteRepo *repository.VoteRepository, userRepo *repository.UserRepository, creditService *services.CreditService, wsHub *websocket.Hub, cfg *config.Config) *VoteHandler {
	return &VoteHandler{
		voteRepo:      voteRepo,
		userRepo:      userRepo,
		creditService: creditService,
		wsHub:         wsHub,
		cfg:           cfg,
	}
}

// Create creates a new vote
// POST /api/v1/votes
func (h *VoteHandler) Create(c *gin.Context) {
	// Check if voting is paused
	if h.cfg.VotingPaused {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Voting is currently paused by admin",
		})
		return
	}

	// Get current user
	fromUserID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	// Parse request body
	var req models.CreateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Default to 1 point if not specified
	points := req.Points
	if points == 0 {
		points = 1
	}

	// Validate points (1-3)
	if points < 1 || points > 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Points must be between 1 and 3",
		})
		return
	}

	// Validate achievement
	if !models.IsValidAchievement(req.AchievementID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid achievement ID",
		})
		return
	}

	// Can't vote for yourself
	if fromUserID == req.ToUserID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot vote for yourself",
		})
		return
	}

	// Check if target user exists
	toUser, err := h.userRepo.GetByID(req.ToUserID)
	if err != nil {
		log.Printf("Failed to check target user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process vote",
		})
		return
	}
	if toUser == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Target user not found",
		})
		return
	}

	// Check and update credits for current user
	fromUser, err := h.userRepo.GetByID(fromUserID)
	if err != nil {
		log.Printf("Failed to load current user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process vote",
		})
		return
	}

	// Calculate current credits
	_, err = h.creditService.CalculateAndUpdateCredits(fromUser)
	if err != nil {
		log.Printf("Failed to calculate credits: %v", err)
	}

	// Reload user to get updated credits
	fromUser, _ = h.userRepo.GetByID(fromUserID)

	// Check if user has enough credits for the requested points
	if !h.creditService.CanAffordVoteWithPoints(fromUser, points) {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Insufficient credits",
			"credits": fromUser.Credits,
		})
		return
	}

	// Deduct credits based on points
	if err := h.creditService.DeductVoteCostWithPoints(fromUserID, points); err != nil {
		log.Printf("Failed to deduct credits: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process vote",
		})
		return
	}

	// Get the current king before creating votes (only for positive achievements)
	var previousKingID uint64
	achievement, _ := models.GetAchievement(req.AchievementID)
	if achievement.IsPositive {
		champsBefore, _ := h.voteRepo.GetChampions()
		if champsBefore != nil && champsBefore.King != nil {
			previousKingID = champsBefore.King.User.ID
		}
	}

	// Determine if vote is secret:
	// - If is_secret is explicitly set in request, use that value
	// - Otherwise: negative achievements default to secret, positive to open
	isSecret := !achievement.IsPositive // default: negative=secret, positive=open
	if req.IsSecret != nil {
		isSecret = *req.IsSecret
	}

	// Create a single vote with points value
	vote := &models.Vote{
		FromUserID:    fromUserID,
		ToUserID:      req.ToUserID,
		AchievementID: req.AchievementID,
		Points:        points,
		IsSecret:      isSecret,
	}

	if err := h.voteRepo.Create(vote); err != nil {
		log.Printf("Failed to create vote: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create vote",
		})
		return
	}

	// Get full vote details for response
	voteDetails, err := h.voteRepo.GetByID(vote.ID)
	if err != nil {
		log.Printf("Failed to get vote details: %v", err)
	}

	// Broadcast vote to all WebSocket clients (once, with points info)
	if voteDetails != nil && h.wsHub != nil {
		achievement, _ := models.GetAchievement(voteDetails.AchievementID)

		// Determine if sender should be anonymized based on visibility mode
		shouldAnonymize := false
		switch h.cfg.VoteVisibilityMode {
		case "all_secret":
			shouldAnonymize = true
		case "all_public":
			shouldAnonymize = false
		default: // "user_choice"
			shouldAnonymize = isSecret
		}

		// Prepare payload - anonymize sender if needed
		fromUserID := voteDetails.FromUser.ID
		fromUsername := voteDetails.FromUser.Username
		fromAvatar := voteDetails.FromUser.AvatarSmall
		if shouldAnonymize {
			fromUserID = 0
			fromUsername = "Anonym"
			fromAvatar = ""
		}

		payload := &websocket.VotePayload{
			VoteID:        voteDetails.ID,
			FromUserID:    fromUserID,
			FromUsername:  fromUsername,
			FromAvatar:    fromAvatar,
			ToUserID:      voteDetails.ToUser.ID,
			ToUsername:    voteDetails.ToUser.Username,
			ToAvatar:      voteDetails.ToUser.AvatarSmall,
			AchievementID: voteDetails.AchievementID,
			Achievement:   achievement.Name,
			IsPositive:    achievement.IsPositive,
			IsSecret:      shouldAnonymize,
			CreatedAt:     voteDetails.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Points:        points,
		}

		// Broadcast to all clients - frontend decides who shows notification popup
		h.wsHub.BroadcastVote(payload)

		// Check if the king has changed (only for positive achievements)
		if achievement.IsPositive {
			champsAfter, _ := h.voteRepo.GetChampions()
			if champsAfter != nil && champsAfter.King != nil {
				newKingID := champsAfter.King.User.ID
				// If king changed, broadcast the new king notification
				if newKingID != previousKingID {
					h.wsHub.BroadcastNewKing(
						newKingID,
						champsAfter.King.User.Username,
						champsAfter.King.User.AvatarURL,
					)
				}
			}
		}
	}

	// Return updated credits
	fromUser, _ = h.userRepo.GetByID(fromUserID)

	c.JSON(http.StatusCreated, gin.H{
		"vote":    voteDetails,
		"credits": fromUser.Credits,
	})
}

// GetTimeline returns recent votes for the timeline
// GET /api/v1/votes
func (h *VoteHandler) GetTimeline(c *gin.Context) {
	votes, err := h.voteRepo.GetRecent(100)
	if err != nil {
		log.Printf("Failed to get timeline: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load timeline",
		})
		return
	}

	if votes == nil {
		votes = []models.VoteWithDetails{}
	}

	// Apply visibility mode to all votes
	for i := range votes {
		votes[i].ApplyVisibilityMode(h.cfg.VoteVisibilityMode)
	}

	c.JSON(http.StatusOK, gin.H{
		"votes": votes,
	})
}

// GetLeaderboard returns the leaderboard (top 3 per achievement)
// GET /api/v1/leaderboard
func (h *VoteHandler) GetLeaderboard(c *gin.Context) {
	leaderboard, err := h.voteRepo.GetLeaderboard(3)
	if err != nil {
		log.Printf("Failed to get leaderboard: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load leaderboard",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": leaderboard,
	})
}

// GetChampions returns the king (winner) and brother of the king (loser)
// GET /api/v1/champions
func (h *VoteHandler) GetChampions(c *gin.Context) {
	champions, err := h.voteRepo.GetChampions()
	if err != nil {
		log.Printf("Failed to get champions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load champions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"champions": champions,
	})
}

// GlobalRankingResponse represents the response for GET /api/v1/ranking
type GlobalRankingResponse struct {
	Rankings           []repository.PlayerRanking `json:"rankings"`
	TotalVotes         int                        `json:"total_votes"`
	MinVotesForRanking int                        `json:"min_votes_for_ranking"`
	RankingActive      bool                       `json:"ranking_active"`
}

// GetGlobalRanking returns the global ranking based on net votes
// GET /api/v1/ranking
func (h *VoteHandler) GetGlobalRanking(c *gin.Context) {
	rankings, err := h.voteRepo.GetGlobalRanking()
	if err != nil {
		log.Printf("Failed to get global ranking: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load ranking",
		})
		return
	}

	totalVotes, err := h.voteRepo.GetTotalVoteCount()
	if err != nil {
		log.Printf("Failed to get total vote count: %v", err)
		totalVotes = 0
	}

	c.JSON(http.StatusOK, GlobalRankingResponse{
		Rankings:           rankings,
		TotalVotes:         totalVotes,
		MinVotesForRanking: h.cfg.MinVotesForRanking,
		RankingActive:      totalVotes >= h.cfg.MinVotesForRanking,
	})
}

// GetMyRanking returns the current user's rank
// GET /api/v1/ranking/me
func (h *VoteHandler) GetMyRanking(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	totalVotes, err := h.voteRepo.GetTotalVoteCount()
	if err != nil {
		log.Printf("Failed to get total vote count: %v", err)
		totalVotes = 0
	}

	rankingActive := totalVotes >= h.cfg.MinVotesForRanking

	// If ranking is not active yet, return early
	if !rankingActive {
		c.JSON(http.StatusOK, gin.H{
			"rank":               nil,
			"total_votes":        totalVotes,
			"min_votes_for_ranking": h.cfg.MinVotesForRanking,
			"ranking_active":     false,
		})
		return
	}

	ranking, err := h.voteRepo.GetUserRank(userID)
	if err != nil {
		log.Printf("Failed to get user rank: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load ranking",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rank":               ranking,
		"total_votes":        totalVotes,
		"min_votes_for_ranking": h.cfg.MinVotesForRanking,
		"ranking_active":     true,
	})
}
