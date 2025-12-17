package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	avatarsDir = "data/avatars"
)

// AvatarCacheService handles caching of Steam avatars locally
type AvatarCacheService struct {
	httpClient *http.Client
	baseDir    string
	backendURL string
}

// NewAvatarCacheService creates a new avatar cache service
func NewAvatarCacheService(backendURL string) *AvatarCacheService {
	svc := &AvatarCacheService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseDir:    avatarsDir,
		backendURL: strings.TrimSuffix(backendURL, "/"),
	}

	// Ensure the avatars directory exists at startup
	if err := svc.ensureDir(); err != nil {
		log.Printf("Warning: Could not create avatars directory: %v", err)
	}

	return svc
}

// ensureDir creates the avatars directory if it doesn't exist
func (s *AvatarCacheService) ensureDir() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// hashURL creates a deterministic filename from an avatar URL
// This allows us to detect when an avatar has changed
func (s *AvatarCacheService) hashURL(avatarURL string) string {
	hash := sha256.Sum256([]byte(avatarURL))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter filename
}

// GetAvatarFilename returns the filename for a user's avatar based on the URL hash
func (s *AvatarCacheService) GetAvatarFilename(steamID string, avatarURL string) string {
	// For SVG avatars (fallback avatars), use .svg extension
	if strings.Contains(avatarURL, "dicebear") || strings.HasSuffix(avatarURL, ".svg") {
		return fmt.Sprintf("%s_%s.svg", steamID, s.hashURL(avatarURL))
	}
	// Steam avatars are typically JPG
	return fmt.Sprintf("%s_%s.jpg", steamID, s.hashURL(avatarURL))
}

// GetAvatarPath returns the local file path for a user's avatar
func (s *AvatarCacheService) GetAvatarPath(steamID string, avatarURL string) string {
	return filepath.Join(s.baseDir, s.GetAvatarFilename(steamID, avatarURL))
}

// HasAvatar checks if an avatar is already cached locally
func (s *AvatarCacheService) HasAvatar(steamID string, avatarURL string) bool {
	path := s.GetAvatarPath(steamID, avatarURL)
	_, err := os.Stat(path)
	return err == nil
}

// GetLocalAvatarURL returns the full URL for serving the cached avatar
// This includes the backend URL so the frontend can use it directly
func (s *AvatarCacheService) GetLocalAvatarURL(steamID string, avatarURL string) string {
	filename := s.GetAvatarFilename(steamID, avatarURL)
	return fmt.Sprintf("%s/api/v1/avatars/%s", s.backendURL, filename)
}

// CacheAvatar downloads and caches a user's avatar
// Returns the local URL if successful, or the original URL as fallback
func (s *AvatarCacheService) CacheAvatar(steamID string, avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	// Skip if already cached
	if s.HasAvatar(steamID, avatarURL) {
		return s.GetLocalAvatarURL(steamID, avatarURL)
	}

	// Ensure directory exists
	if err := s.ensureDir(); err != nil {
		log.Printf("Failed to create avatar cache directory: %v", err)
		return avatarURL
	}

	// Download the avatar
	resp, err := s.httpClient.Get(avatarURL)
	if err != nil {
		log.Printf("Failed to download avatar for user %s from %s: %v", steamID, avatarURL, err)
		return avatarURL
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to download avatar for user %s from %s: HTTP %d", steamID, avatarURL, resp.StatusCode)
		return avatarURL
	}

	// Create the local file
	localPath := s.GetAvatarPath(steamID, avatarURL)
	file, err := os.Create(localPath)
	if err != nil {
		log.Printf("Failed to create avatar file for user %s: %v", steamID, err)
		return avatarURL
	}
	defer file.Close()

	// Copy the avatar data
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Failed to save avatar for user %s: %v", steamID, err)
		// Clean up partial file
		os.Remove(localPath)
		return avatarURL
	}

	log.Printf("Cached avatar for user %s", steamID)
	return s.GetLocalAvatarURL(steamID, avatarURL)
}

// CacheAvatarAsync downloads and caches a user's avatar asynchronously
func (s *AvatarCacheService) CacheAvatarAsync(steamID string, avatarURL string) {
	go func() {
		s.CacheAvatar(steamID, avatarURL)
	}()
}

// GetAvatarByFilename returns the full path to an avatar file by its filename
// Used for serving cached avatars
func (s *AvatarCacheService) GetAvatarByFilename(filename string) string {
	return filepath.Join(s.baseDir, filename)
}

// HasAvatarFile checks if an avatar file exists by filename
func (s *AvatarCacheService) HasAvatarFile(filename string) bool {
	path := s.GetAvatarByFilename(filename)
	_, err := os.Stat(path)
	return err == nil
}

// GetBaseDir returns the base directory for cached avatars
func (s *AvatarCacheService) GetBaseDir() string {
	return s.baseDir
}

// CleanupOldAvatars removes old avatar files for a user (e.g., when avatar changes)
// Keeps only the current avatar file
func (s *AvatarCacheService) CleanupOldAvatars(steamID string, currentFilename string) {
	pattern := filepath.Join(s.baseDir, steamID+"_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("Failed to find old avatars for user %s: %v", steamID, err)
		return
	}

	currentPath := filepath.Join(s.baseDir, currentFilename)
	for _, match := range matches {
		if match != currentPath {
			if err := os.Remove(match); err != nil {
				log.Printf("Failed to remove old avatar %s: %v", match, err)
			} else {
				log.Printf("Cleaned up old avatar: %s", match)
			}
		}
	}
}
