package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// MessageTypeVoteReceived is sent when a user receives a vote
	MessageTypeVoteReceived MessageType = "vote_received"
	// MessageTypeNewVote is sent to all clients when any vote is created (for timeline)
	MessageTypeNewVote MessageType = "new_vote"
	// MessageTypeUserJoined is sent when a new user joins
	MessageTypeUserJoined MessageType = "user_joined"
	// MessageTypeSettingsUpdate is sent when admin changes settings
	MessageTypeSettingsUpdate MessageType = "settings_update"
	// MessageTypeCreditsReset is sent when admin resets all credits
	MessageTypeCreditsReset MessageType = "credits_reset"
	// MessageTypeCreditsGiven is sent when admin gives everyone a credit
	MessageTypeCreditsGiven MessageType = "credits_given"
	// MessageTypeVotesReset is sent when admin deletes all votes
	MessageTypeVotesReset MessageType = "votes_reset"
	// MessageTypeChatMessage is sent when a new chat message is posted
	MessageTypeChatMessage MessageType = "chat_message"
	// MessageTypeNewKing is sent when the king changes
	MessageTypeNewKing MessageType = "new_king"
	// MessageTypeGamesSyncProgress is sent during background game library sync
	MessageTypeGamesSyncProgress MessageType = "games_sync_progress"
	// MessageTypeGamesSyncComplete is sent when game sync is finished
	MessageTypeGamesSyncComplete MessageType = "games_sync_complete"
	// MessageTypeUserKicked is sent when a user is kicked
	MessageTypeUserKicked MessageType = "user_kicked"
	// MessageTypeUserBanned is sent when a user is banned
	MessageTypeUserBanned MessageType = "user_banned"
	// MessageTypeVoteInvalidation is sent when a vote's invalidation status changes
	MessageTypeVoteInvalidation MessageType = "vote_invalidation"
	// MessageTypeError is sent when an error occurs
	MessageTypeError MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// VotePayload contains vote information for notifications
type VotePayload struct {
	VoteID        uint64 `json:"vote_id"`
	FromUserID    uint64 `json:"from_user_id"`
	FromUsername  string `json:"from_username"`
	FromAvatar    string `json:"from_avatar"`
	ToUserID      uint64 `json:"to_user_id"`
	ToUsername    string `json:"to_username"`
	ToAvatar      string `json:"to_avatar"`
	AchievementID string `json:"achievement_id"`
	Achievement   string `json:"achievement_name"`
	IsPositive    bool   `json:"is_positive"`
	IsSecret      bool   `json:"is_secret"`
	CreatedAt     string `json:"created_at"`
	Points        int    `json:"points,omitempty"` // Number of points awarded (1-3)
}

// SettingsPayload contains settings information for broadcasts
type SettingsPayload struct {
	CreditIntervalMinutes  int     `json:"credit_interval_minutes"`
	CreditMax              int     `json:"credit_max"`
	VotingPaused           bool    `json:"voting_paused"`
	VoteVisibilityMode     string  `json:"vote_visibility_mode"`     // "user_choice", "all_secret", "all_public"
	NegativeVotingDisabled bool    `json:"negative_voting_disabled"` // When true, negative achievements cannot be voted
	CountdownTarget        *string `json:"countdown_target,omitempty"` // RFC3339 formatted time, null if not set
}

// ChatMessagePayload contains chat message information for broadcasts
type ChatMessagePayload struct {
	ID           uint64        `json:"id"`
	UserID       uint64        `json:"user_id"`
	Username     string        `json:"username"`
	SteamID      string        `json:"steam_id"`
	AvatarSmall  string        `json:"avatar_small"`
	Message      string        `json:"message"`
	Achievements interface{}   `json:"achievements"` // Achievement badges at time of message
	CreatedAt    string        `json:"created_at"`
}

// Client represents a connected WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint64
	steamID  string
	username string
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[uint64]*Client

	// All clients for broadcast
	allClients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast to all clients
	broadcast chan []byte

	// Send to specific user
	sendToUser chan *UserMessage

	mutex sync.RWMutex
}

// UserMessage is a message targeted at a specific user
type UserMessage struct {
	UserID  uint64
	Message []byte
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint64]*Client),
		allClients: make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		sendToUser: make(chan *UserMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.userID] = client
			h.allClients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket: Client connected - User %d (%s)", client.userID, client.username)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.allClients[client]; ok {
				delete(h.allClients, client)
				delete(h.clients, client.userID)
				close(client.send)
				log.Printf("WebSocket: Client disconnected - User %d (%s)", client.userID, client.username)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.allClients {
				select {
				case client.send <- message:
				default:
					// Client send buffer full, close connection
					close(client.send)
					delete(h.allClients, client)
					delete(h.clients, client.userID)
				}
			}
			h.mutex.RUnlock()

		case userMsg := <-h.sendToUser:
			h.mutex.RLock()
			if client, ok := h.clients[userMsg.UserID]; ok {
				select {
				case client.send <- userMsg.Message:
				default:
					// Client send buffer full
					close(client.send)
					delete(h.allClients, client)
					delete(h.clients, client.userID)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastVote sends a new vote notification to all clients
func (h *Hub) BroadcastVote(payload *VotePayload) {
	msg := Message{
		Type:    MessageTypeNewVote,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal broadcast message: %v", err)
		return
	}

	log.Printf("WebSocket: Broadcasting new_vote to %d clients", h.GetConnectedUserCount())
	h.broadcast <- data
}

// NotifyVoteReceived sends a notification to the user who received a vote
func (h *Hub) NotifyVoteReceived(toUserID uint64, payload *VotePayload) {
	msg := Message{
		Type:    MessageTypeVoteReceived,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal notification message: %v", err)
		return
	}

	log.Printf("WebSocket: Sending vote_received notification to user %d (connected: %v)", toUserID, h.IsUserConnected(toUserID))
	h.sendToUser <- &UserMessage{
		UserID:  toUserID,
		Message: data,
	}
}

// GetConnectedUserCount returns the number of connected users
func (h *Hub) GetConnectedUserCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.allClients)
}

// IsUserConnected checks if a specific user is connected
func (h *Hub) IsUserConnected(userID uint64) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// BroadcastVoteInvalidation sends vote invalidation update to all clients
func (h *Hub) BroadcastVoteInvalidation(voteID uint64, isInvalidated bool) {
	msg := Message{
		Type: MessageTypeVoteInvalidation,
		Payload: map[string]interface{}{
			"vote_id":        voteID,
			"is_invalidated": isInvalidated,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal vote invalidation message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted vote invalidation (vote %d, invalidated: %v) to all clients", voteID, isInvalidated)
}

// BroadcastSettingsUpdate sends settings update to all connected clients
func (h *Hub) BroadcastSettingsUpdate(payload *SettingsPayload) {
	msg := Message{
		Type:    MessageTypeSettingsUpdate,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal settings message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted settings update to all clients")
}

// BroadcastCreditsReset notifies all clients that credits have been reset
func (h *Hub) BroadcastCreditsReset() {
	msg := Message{
		Type:    MessageTypeCreditsReset,
		Payload: map[string]string{"message": "Alle Credits wurden zurückgesetzt"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal credits reset message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted credits reset to all clients")
}

// BroadcastCreditsGiven notifies all clients that they received a credit
func (h *Hub) BroadcastCreditsGiven() {
	msg := Message{
		Type:    MessageTypeCreditsGiven,
		Payload: map[string]string{"message": "Du hast 1 Credit erhalten"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal credits given message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted credits given to all clients")
}

// BroadcastVotesReset notifies all clients that all votes have been deleted
func (h *Hub) BroadcastVotesReset() {
	msg := Message{
		Type:    MessageTypeVotesReset,
		Payload: map[string]string{"message": "Alle Votes wurden gelöscht"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal votes reset message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted votes reset to all clients")
}

// BroadcastChatMessage sends a new chat message to all clients
func (h *Hub) BroadcastChatMessage(payload *ChatMessagePayload) {
	msg := Message{
		Type:    MessageTypeChatMessage,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal chat message: %v", err)
		return
	}

	log.Printf("WebSocket: Broadcasting chat_message to %d clients", h.GetConnectedUserCount())
	h.broadcast <- data
}

// NewKingPayload contains info about the new king
type NewKingPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// BroadcastNewKing notifies all clients that there is a new king
func (h *Hub) BroadcastNewKing(userID uint64, username string, avatar string) {
	msg := Message{
		Type: MessageTypeNewKing,
		Payload: NewKingPayload{
			UserID:   userID,
			Username: username,
			Avatar:   avatar,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal new king message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted new king notification for user %s", username)
}

// GamesSyncProgressPayload contains progress info for game library sync
type GamesSyncProgressPayload struct {
	Phase         string `json:"phase"`          // "fetching_users", "fetching_categories", "complete"
	CurrentGame   string `json:"current_game"`   // Name of current game being processed
	ProcessedCount int   `json:"processed_count"` // Number of games processed so far
	TotalCount    int    `json:"total_count"`    // Total games to process
	Percentage    int    `json:"percentage"`     // 0-100
}

// BroadcastGamesSyncProgress notifies all clients about game sync progress
func (h *Hub) BroadcastGamesSyncProgress(payload *GamesSyncProgressPayload) {
	msg := Message{
		Type:    MessageTypeGamesSyncProgress,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal games sync progress message: %v", err)
		return
	}

	h.broadcast <- data
}

// BroadcastGamesSyncComplete notifies all clients that game sync is complete
func (h *Hub) BroadcastGamesSyncComplete(totalGames int) {
	msg := Message{
		Type: MessageTypeGamesSyncComplete,
		Payload: map[string]interface{}{
			"message":     "Spielebibliothek aktualisiert",
			"total_games": totalGames,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal games sync complete message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted games sync complete with %d games", totalGames)
}

// UserActionPayload contains info about a user kick/ban
type UserActionPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
}

// BroadcastUserKicked notifies all clients that a user was kicked
func (h *Hub) BroadcastUserKicked(userID uint64, username string) {
	msg := Message{
		Type: MessageTypeUserKicked,
		Payload: UserActionPayload{
			UserID:   userID,
			Username: username,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal user kicked message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted user kicked notification for %s", username)
}

// BroadcastUserBanned notifies all clients that a user was banned
func (h *Hub) BroadcastUserBanned(userID uint64, username string) {
	msg := Message{
		Type: MessageTypeUserBanned,
		Payload: UserActionPayload{
			UserID:   userID,
			Username: username,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal user banned message: %v", err)
		return
	}

	h.broadcast <- data
	log.Printf("WebSocket: Broadcasted user banned notification for %s", username)
}
