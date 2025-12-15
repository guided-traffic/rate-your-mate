package repository

import (
	"database/sql"
	"fmt"

	"github.com/guided-traffic/rate-your-mate/backend/database"
	"github.com/guided-traffic/rate-your-mate/backend/models"
)

// VoteRepository handles vote database operations
type VoteRepository struct{}

// NewVoteRepository creates a new vote repository
func NewVoteRepository() *VoteRepository {
	return &VoteRepository{}
}

// Create creates a new vote (with retry for SQLITE_BUSY)
func (r *VoteRepository) Create(vote *models.Vote) error {
	return database.WithRetry(func() error {
		result, err := database.DB.Exec(`
			INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, is_secret)
			VALUES (?, ?, ?, ?, ?)`,
			vote.FromUserID, vote.ToUserID, vote.AchievementID, vote.Points, vote.IsSecret,
		)
		if err != nil {
			return fmt.Errorf("failed to create vote: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert id: %w", err)
		}

		vote.ID = uint64(id)
		return nil
	})
}

// GetRecent returns the most recent votes for the timeline
func (r *VoteRepository) GetRecent(limit int) ([]models.VoteWithDetails, error) {
	rows, err := database.DB.Query(`
		SELECT
			v.id, v.achievement_id, v.points, v.is_secret, v.created_at,
			fu.id, fu.steam_id, fu.username, fu.avatar_url, fu.avatar_small, fu.profile_url,
			tu.id, tu.steam_id, tu.username, tu.avatar_url, tu.avatar_small, tu.profile_url
		FROM votes v
		JOIN users fu ON v.from_user_id = fu.id
		JOIN users tu ON v.to_user_id = tu.id
		ORDER BY v.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent votes: %w", err)
	}
	defer rows.Close()

	var votes []models.VoteWithDetails
	for rows.Next() {
		var v models.VoteWithDetails
		err := rows.Scan(
			&v.ID, &v.AchievementID, &v.Points, &v.IsSecret, &v.CreatedAt,
			&v.FromUser.ID, &v.FromUser.SteamID, &v.FromUser.Username, &v.FromUser.AvatarURL, &v.FromUser.AvatarSmall, &v.FromUser.ProfileURL,
			&v.ToUser.ID, &v.ToUser.SteamID, &v.ToUser.Username, &v.ToUser.AvatarURL, &v.ToUser.AvatarSmall, &v.ToUser.ProfileURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vote row: %w", err)
		}

		// Add achievement details
		if achievement, ok := models.GetAchievement(v.AchievementID); ok {
			v.Achievement = achievement
		}

		votes = append(votes, v)
	}

	return votes, nil
}

// GetByID returns a vote by ID with full details
func (r *VoteRepository) GetByID(id uint64) (*models.VoteWithDetails, error) {
	var v models.VoteWithDetails
	err := database.DB.QueryRow(`
		SELECT
			v.id, v.achievement_id, v.points, v.is_secret, v.created_at,
			fu.id, fu.steam_id, fu.username, fu.avatar_url, fu.avatar_small, fu.profile_url,
			tu.id, tu.steam_id, tu.username, tu.avatar_url, tu.avatar_small, tu.profile_url
		FROM votes v
		JOIN users fu ON v.from_user_id = fu.id
		JOIN users tu ON v.to_user_id = tu.id
		WHERE v.id = ?`, id,
	).Scan(
		&v.ID, &v.AchievementID, &v.Points, &v.IsSecret, &v.CreatedAt,
		&v.FromUser.ID, &v.FromUser.SteamID, &v.FromUser.Username, &v.FromUser.AvatarURL, &v.FromUser.AvatarSmall, &v.FromUser.ProfileURL,
		&v.ToUser.ID, &v.ToUser.SteamID, &v.ToUser.Username, &v.ToUser.AvatarURL, &v.ToUser.AvatarSmall, &v.ToUser.ProfileURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vote by id: %w", err)
	}

	// Add achievement details
	if achievement, ok := models.GetAchievement(v.AchievementID); ok {
		v.Achievement = achievement
	}

	return &v, nil
}

// LeaderboardEntry represents a user's position on the leaderboard for an achievement
type LeaderboardEntry struct {
	User       models.PublicUser `json:"user"`
	VoteCount  int               `json:"vote_count"`
	Rank       int               `json:"rank"`
}

// AchievementLeaderboard represents the leaderboard for a single achievement
type AchievementLeaderboard struct {
	Achievement models.Achievement `json:"achievement"`
	Leaders     []LeaderboardEntry `json:"leaders"`
}

// GetLeaderboard returns the top N users per achievement
func (r *VoteRepository) GetLeaderboard(topN int) ([]AchievementLeaderboard, error) {
	// Get all achievements and their top voters (sum of points)
	rows, err := database.DB.Query(`
		SELECT
			v.achievement_id,
			u.id, u.steam_id, u.username, u.avatar_url, u.avatar_small, u.profile_url,
			SUM(v.points) as vote_count
		FROM votes v
		JOIN users u ON v.to_user_id = u.id
		GROUP BY v.achievement_id, v.to_user_id
		ORDER BY v.achievement_id, vote_count DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}
	defer rows.Close()

	// Group by achievement
	achievementMap := make(map[string][]LeaderboardEntry)
	for rows.Next() {
		var achievementID string
		var user models.PublicUser
		var voteCount int

		err := rows.Scan(
			&achievementID,
			&user.ID, &user.SteamID, &user.Username, &user.AvatarURL, &user.AvatarSmall, &user.ProfileURL,
			&voteCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard row: %w", err)
		}

		// Only keep top N per achievement
		if len(achievementMap[achievementID]) < topN {
			entry := LeaderboardEntry{
				User:      user,
				VoteCount: voteCount,
				Rank:      len(achievementMap[achievementID]) + 1,
			}
			achievementMap[achievementID] = append(achievementMap[achievementID], entry)
		}
	}

	// Build result with all achievements (even those with no votes)
	var result []AchievementLeaderboard
	for _, achievement := range models.GetAllAchievements() {
		lb := AchievementLeaderboard{
			Achievement: achievement,
			Leaders:     achievementMap[achievement.ID],
		}
		if lb.Leaders == nil {
			lb.Leaders = []LeaderboardEntry{}
		}
		result = append(result, lb)
	}

	return result, nil
}

// GetVotesForUser returns all votes received by a user
func (r *VoteRepository) GetVotesForUser(userID uint64) ([]models.VoteWithDetails, error) {
	rows, err := database.DB.Query(`
		SELECT
			v.id, v.achievement_id, v.points, v.is_secret, v.created_at,
			fu.id, fu.steam_id, fu.username, fu.avatar_url, fu.avatar_small, fu.profile_url,
			tu.id, tu.steam_id, tu.username, tu.avatar_url, tu.avatar_small, tu.profile_url
		FROM votes v
		JOIN users fu ON v.from_user_id = fu.id
		JOIN users tu ON v.to_user_id = tu.id
		WHERE v.to_user_id = ?
		ORDER BY v.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes for user: %w", err)
	}
	defer rows.Close()

	var votes []models.VoteWithDetails
	for rows.Next() {
		var v models.VoteWithDetails
		err := rows.Scan(
			&v.ID, &v.AchievementID, &v.Points, &v.IsSecret, &v.CreatedAt,
			&v.FromUser.ID, &v.FromUser.SteamID, &v.FromUser.Username, &v.FromUser.AvatarURL, &v.FromUser.AvatarSmall, &v.FromUser.ProfileURL,
			&v.ToUser.ID, &v.ToUser.SteamID, &v.ToUser.Username, &v.ToUser.AvatarURL, &v.ToUser.AvatarSmall, &v.ToUser.ProfileURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vote row: %w", err)
		}

		if achievement, ok := models.GetAchievement(v.AchievementID); ok {
			v.Achievement = achievement
		}

		votes = append(votes, v)
	}

	return votes, nil
}

// Champion represents a top player in the ranking
type Champion struct {
	User       *models.PublicUser `json:"user"`
	TotalScore int                `json:"total_score"`  // Net votes + bonus points
	NetVotes   int                `json:"net_votes"`    // Positive - negative votes
	BonusPoints int               `json:"bonus_points"` // Bonus from achievement placements
	Rank       int                `json:"rank"`
}

// ChampionsResult contains the top 3 players
type ChampionsResult struct {
	King   *Champion `json:"king"`   // 1st place
	Second *Champion `json:"second"` // 2nd place
	Third  *Champion `json:"third"`  // 3rd place
}

// GetChampions calculates the top 3 players based on:
// 1. Net votes (positive - negative)
// 2. Bonus points from holding top 3 positions in positive achievements (1st: +5, 2nd: +3, 3rd: +2)
// Tie-breaking for achievement positions: first vote wins (earlier created_at)
func (r *VoteRepository) GetChampions() (*ChampionsResult, error) {
	result := &ChampionsResult{}

	// Get global rankings (already includes bonus points)
	rankings, err := r.GetGlobalRanking()
	if err != nil {
		return nil, err
	}

	// Build result with top 3
	for i, p := range rankings {
		if i >= 3 {
			break
		}
		champion := &Champion{
			User: &models.PublicUser{
				ID:          p.User.ID,
				SteamID:     p.User.SteamID,
				Username:    p.User.Username,
				AvatarURL:   p.User.AvatarURL,
				AvatarSmall: p.User.AvatarSmall,
				ProfileURL:  p.User.ProfileURL,
			},
			TotalScore:  p.TotalScore,
			NetVotes:    p.NetVotes,
			BonusPoints: p.BonusPoints,
			Rank:        p.Rank,
		}

		switch i {
		case 0:
			result.King = champion
		case 1:
			result.Second = champion
		case 2:
			result.Third = champion
		}
	}

	return result, nil
}

// DeleteAll deletes all votes from the database (admin only)
func (r *VoteRepository) DeleteAll() (int64, error) {
	var rowsAffected int64
	err := database.WithRetry(func() error {
		result, err := database.DB.Exec(`DELETE FROM votes`)
		if err != nil {
			return fmt.Errorf("failed to delete all votes: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		return nil
	})

	return rowsAffected, err
}

// PlayerRanking represents a user's global ranking based on total score (net votes + bonus points)
type PlayerRanking struct {
	User        models.PublicUser `json:"user"`
	TotalScore  int               `json:"total_score"`  // net votes + bonus points
	NetVotes    int               `json:"net_votes"`    // positive votes - negative votes
	BonusPoints int               `json:"bonus_points"` // bonus from achievement placements
	Rank        int               `json:"rank"`
}

// GlobalRankingResult contains the global ranking data
type GlobalRankingResult struct {
	Rankings       []PlayerRanking `json:"rankings"`
	TotalVotes     int             `json:"total_votes"`
	MinVotesNeeded int             `json:"min_votes_needed"`
}

// GetTotalVoteCount returns the total number of votes in the database
func (r *VoteRepository) GetTotalVoteCount() (int, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM votes`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total vote count: %w", err)
	}
	return count, nil
}

// getAchievementBonusPoints calculates bonus points for each user based on their achievement positions
// Only positive achievements count for bonus: 1st place = 5, 2nd = 3, 3rd = 2 points
func (r *VoteRepository) getAchievementBonusPoints() (map[uint64]int, error) {
	rows, err := database.DB.Query(`
		SELECT
			v.achievement_id,
			v.to_user_id,
			SUM(v.points) as vote_count,
			MIN(v.created_at) as first_vote
		FROM votes v
		WHERE v.achievement_id IN ('pro-player', 'teamplayer', 'clutch-king', 'support-hero', 'stratege', 'good-sport')
		GROUP BY v.achievement_id, v.to_user_id
		ORDER BY v.achievement_id, vote_count DESC, first_vote ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get achievement rankings: %w", err)
	}
	defer rows.Close()

	bonusPoints := make(map[uint64]int)
	currentAchievement := ""
	positionInAchievement := 0

	for rows.Next() {
		var achievementID string
		var userID uint64
		var voteCount int
		var firstVote interface{}

		err := rows.Scan(&achievementID, &userID, &voteCount, &firstVote)
		if err != nil {
			return nil, fmt.Errorf("failed to scan achievement ranking row: %w", err)
		}

		if achievementID != currentAchievement {
			currentAchievement = achievementID
			positionInAchievement = 0
		}

		positionInAchievement++

		switch positionInAchievement {
		case 1:
			bonusPoints[userID] += 5
		case 2:
			bonusPoints[userID] += 3
		case 3:
			bonusPoints[userID] += 2
		}
	}

	return bonusPoints, nil
}

// GetGlobalRanking calculates the global ranking based on total score (net votes + bonus points)
// Users with the same total score share the same rank
func (r *VoteRepository) GetGlobalRanking() ([]PlayerRanking, error) {
	// Step 1: Get bonus points from achievement positions
	bonusPoints, err := r.getAchievementBonusPoints()
	if err != nil {
		return nil, err
	}

	// Step 2: Calculate net votes per user
	rows, err := database.DB.Query(`
		SELECT
			u.id, u.steam_id, u.username, u.avatar_url, u.avatar_small, u.profile_url,
			COALESCE(SUM(CASE
				WHEN v.achievement_id IN ('pro-player', 'teamplayer', 'clutch-king', 'support-hero', 'stratege', 'good-sport')
				THEN v.points
				ELSE 0
			END), 0) -
			COALESCE(SUM(CASE
				WHEN v.achievement_id IN ('rage-quitter', 'toxic', 'friendly-fire-expert')
				THEN v.points
				ELSE 0
			END), 0) as net_votes
		FROM users u
		LEFT JOIN votes v ON v.to_user_id = u.id
		WHERE NOT EXISTS (SELECT 1 FROM banned_users b WHERE b.steam_id = u.steam_id)
		GROUP BY u.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get global ranking: %w", err)
	}
	defer rows.Close()

	var rankings []PlayerRanking
	for rows.Next() {
		var user models.PublicUser
		var netVotes int

		err := rows.Scan(
			&user.ID, &user.SteamID, &user.Username, &user.AvatarURL, &user.AvatarSmall, &user.ProfileURL,
			&netVotes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ranking row: %w", err)
		}

		bonus := bonusPoints[user.ID]
		rankings = append(rankings, PlayerRanking{
			User:        user,
			TotalScore:  netVotes + bonus,
			NetVotes:    netVotes,
			BonusPoints: bonus,
		})
	}

	// Sort by total score descending, then by username
	for i := 0; i < len(rankings); i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].TotalScore > rankings[i].TotalScore ||
				(rankings[j].TotalScore == rankings[i].TotalScore && rankings[j].User.Username < rankings[i].User.Username) {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	// Assign ranks - users with the same total score share the same rank
	currentRank := 1
	for i := range rankings {
		if i > 0 && rankings[i].TotalScore < rankings[i-1].TotalScore {
			currentRank = i + 1
		}
		rankings[i].Rank = currentRank
	}

	return rankings, nil
}

// GetUserRank returns the rank for a specific user
func (r *VoteRepository) GetUserRank(userID uint64) (*PlayerRanking, error) {
	rankings, err := r.GetGlobalRanking()
	if err != nil {
		return nil, err
	}

	for _, ranking := range rankings {
		if ranking.User.ID == userID {
			return &ranking, nil
		}
	}

	return nil, nil
}
