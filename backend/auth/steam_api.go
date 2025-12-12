package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	steamAPIBaseURL = "https://api.steampowered.com"
)

// SteamPlayer represents a Steam player's profile data
type SteamPlayer struct {
	SteamID         string `json:"steamid"`
	PersonaName     string `json:"personaname"`
	ProfileURL      string `json:"profileurl"`
	Avatar          string `json:"avatar"`           // 32x32
	AvatarMedium    string `json:"avatarmedium"`     // 64x64
	AvatarFull      string `json:"avatarfull"`       // 184x184
	PersonaState    int    `json:"personastate"`     // 0=Offline, 1=Online, etc.
	CommunityVisibilityState int `json:"communityvisibilitystate"`
	ProfileState    int    `json:"profilestate"`
	LastLogoff      int64  `json:"lastlogoff"`
	RealName        string `json:"realname,omitempty"`
	TimeCreated     int64  `json:"timecreated,omitempty"`
	LocCountryCode  string `json:"loccountrycode,omitempty"`
}

// steamAPIResponse represents the API response structure
type steamAPIResponse struct {
	Response struct {
		Players []SteamPlayer `json:"players"`
	} `json:"response"`
}

// SteamAPIClient handles communication with the Steam Web API
type SteamAPIClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewSteamAPIClient creates a new Steam API client
func NewSteamAPIClient(apiKey string) *SteamAPIClient {
	return &SteamAPIClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPlayerSummary fetches a single player's profile data
func (c *SteamAPIClient) GetPlayerSummary(steamID string) (*SteamPlayer, error) {
	// Skip fake users (used for development/testing)
	if strings.HasPrefix(steamID, "FAKE_") {
		return nil, fmt.Errorf("fake user: %s", steamID)
	}

	players, err := c.GetPlayerSummaries([]string{steamID})
	if err != nil {
		return nil, err
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("player not found: %s", steamID)
	}

	return &players[0], nil
}

// GetPlayerSummaries fetches profile data for multiple players (max 100)
func (c *SteamAPIClient) GetPlayerSummaries(steamIDs []string) ([]SteamPlayer, error) {
	if len(steamIDs) == 0 {
		return nil, fmt.Errorf("no Steam IDs provided")
	}

	// Filter out fake users (used for development/testing)
	realSteamIDs := make([]string, 0, len(steamIDs))
	for _, id := range steamIDs {
		if !strings.HasPrefix(id, "FAKE_") {
			realSteamIDs = append(realSteamIDs, id)
		}
	}

	if len(realSteamIDs) == 0 {
		return []SteamPlayer{}, nil
	}

	if len(realSteamIDs) > 100 {
		return nil, fmt.Errorf("maximum 100 Steam IDs allowed per request")
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf("Steam API key not configured")
	}

	// Build the API URL
	url := fmt.Sprintf(
		"%s/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s",
		steamAPIBaseURL,
		c.apiKey,
		strings.Join(realSteamIDs, ","),
	)

	// Make the request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var apiResp steamAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Steam API response: %w", err)
	}

	return apiResp.Response.Players, nil
}

// IsConfigured returns true if the API client has a valid API key
func (c *SteamAPIClient) IsConfigured() bool {
	return c.apiKey != ""
}

// CheckConnectivity verifies that the Steam API endpoints are reachable
// Returns nil if all checks pass, otherwise returns an error describing the issue
func (c *SteamAPIClient) CheckConnectivity() error {
	// Check Steam Community (OpenID endpoint)
	steamCommunityURL := "https://steamcommunity.com/openid"
	resp, err := c.httpClient.Head(steamCommunityURL)
	if err != nil {
		return fmt.Errorf("cannot reach Steam Community (%s): %w", steamCommunityURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Steam Community returned status %d", resp.StatusCode)
	}

	// Check Steam Web API (only if API key is configured)
	if c.apiKey != "" {
		// Use a simple API call to verify connectivity and API key validity
		// We use GetPlayerSummaries with Valve's test Steam ID
		testURL := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=76561197960435530", steamAPIBaseURL, c.apiKey)
		resp, err := c.httpClient.Get(testURL)
		if err != nil {
			return fmt.Errorf("cannot reach Steam Web API (%s): %w", steamAPIBaseURL, err)
		}
		resp.Body.Close()
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return fmt.Errorf("Steam API key is invalid or unauthorized (status %d)", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("Steam Web API returned status %d", resp.StatusCode)
		}
	}

	return nil
}
