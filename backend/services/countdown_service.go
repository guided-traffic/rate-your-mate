package services

import (
	"log"
	"time"

	"github.com/guided-traffic/rate-your-mate/backend/config"
	"github.com/guided-traffic/rate-your-mate/backend/repository"
	"github.com/guided-traffic/rate-your-mate/backend/websocket"
)

// CountdownService handles countdown expiration and automatic voting pause lift
type CountdownService struct {
	cfg      *config.Config
	wsHub    *websocket.Hub
	userRepo *repository.UserRepository
	ticker   *time.Ticker
	done     chan bool
}

// NewCountdownService creates a new countdown service
func NewCountdownService(cfg *config.Config, wsHub *websocket.Hub, userRepo *repository.UserRepository) *CountdownService {
	return &CountdownService{
		cfg:      cfg,
		wsHub:    wsHub,
		userRepo: userRepo,
		done:     make(chan bool),
	}
}

// Start begins the countdown watcher
func (s *CountdownService) Start() {
	// Check every second for countdown expiration
	s.ticker = time.NewTicker(1 * time.Second)
	go s.watch()
	log.Println("Countdown service started")
}

// Stop stops the countdown watcher
func (s *CountdownService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("Countdown service stopped")
}

// watch continuously checks if the countdown has expired
func (s *CountdownService) watch() {
	for {
		select {
		case <-s.done:
			return
		case <-s.ticker.C:
			s.checkCountdown()
		}
	}
}

// checkCountdown checks if the countdown has expired and lifts voting pause
func (s *CountdownService) checkCountdown() {
	// Skip if no countdown is set
	if s.cfg.CountdownTarget.IsZero() {
		return
	}

	// Check if countdown has expired
	if time.Now().After(s.cfg.CountdownTarget) {
		log.Printf("Countdown expired at %v - lifting voting pause", s.cfg.CountdownTarget)

		// Lift voting pause if it was set
		if s.cfg.VotingPaused {
			wasAlreadyPaused := s.cfg.VotingPaused
			s.cfg.VotingPaused = false

			// Shift credit timers if voting was paused
			if wasAlreadyPaused && !s.cfg.VotingPausedAt.IsZero() {
				pauseDuration := time.Since(s.cfg.VotingPausedAt)
				log.Printf("Automatically resumed voting after %v pause (countdown expired)", pauseDuration)

				// Shift all users' last_credit_at forward by the pause duration
				if err := s.userRepo.ShiftAllLastCreditAt(pauseDuration); err != nil {
					log.Printf("Warning: Failed to shift last_credit_at times: %v", err)
				} else {
					log.Printf("Shifted all users' last_credit_at forward by %v", pauseDuration)
				}

				// Reset the paused timestamp
				s.cfg.VotingPausedAt = time.Time{}
			}

			// Broadcast settings update
			s.wsHub.BroadcastSettingsUpdate(&websocket.SettingsPayload{
				CreditIntervalMinutes:  s.cfg.CreditIntervalMinutes,
				CreditMax:              s.cfg.CreditMax,
				VotingPaused:           s.cfg.VotingPaused,
				VoteVisibilityMode:     s.cfg.VoteVisibilityMode,
				NegativeVotingDisabled: s.cfg.NegativeVotingDisabled,
				CountdownTarget:        nil, // Countdown has expired
			})
		}

		// Clear the countdown target
		s.cfg.CountdownTarget = time.Time{}
		log.Println("Countdown target cleared")
	}
}
