package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	gameImagesDir = "data/game_images"
	steamCDNURL   = "https://steamcdn-a.akamaihd.net/steam/apps"
)

// ImageCacheService handles caching of game images locally
type ImageCacheService struct {
	httpClient *http.Client
	baseDir    string
}

// NewImageCacheService creates a new image cache service
func NewImageCacheService() *ImageCacheService {
	svc := &ImageCacheService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseDir: gameImagesDir,
	}

	// Ensure the images directory exists at startup
	if err := svc.ensureDir(); err != nil {
		log.Printf("Warning: Could not create game images directory: %v", err)
	}

	return svc
}

// ensureDir creates the game images directory if it doesn't exist
func (s *ImageCacheService) ensureDir() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// GetImagePath returns the local file path for a game's header image
func (s *ImageCacheService) GetImagePath(appID int) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%d.jpg", appID))
}

// HasImage checks if an image is already cached locally
func (s *ImageCacheService) HasImage(appID int) bool {
	path := s.GetImagePath(appID)
	_, err := os.Stat(path)
	return err == nil
}

// GetLocalImageURL returns the URL path for serving the cached image
// This is the path that will be used by the frontend
func (s *ImageCacheService) GetLocalImageURL(appID int) string {
	return fmt.Sprintf("/api/v1/games/images/%d.jpg", appID)
}

// GetSteamImageURL returns the original Steam CDN URL for a game's header image
func (s *ImageCacheService) GetSteamImageURL(appID int) string {
	return fmt.Sprintf("%s/%d/header.jpg", steamCDNURL, appID)
}

// CacheImage downloads and caches a game's header image
// Returns true if the image was successfully cached, false otherwise
func (s *ImageCacheService) CacheImage(appID int) bool {
	// Skip if already cached
	if s.HasImage(appID) {
		return true
	}

	// Ensure directory exists
	if err := s.ensureDir(); err != nil {
		log.Printf("Failed to create image cache directory: %v", err)
		return false
	}

	// Download from Steam CDN
	imageURL := s.GetSteamImageURL(appID)
	resp, err := s.httpClient.Get(imageURL)
	if err != nil {
		log.Printf("Failed to download image for game %d: %v", appID, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to download image for game %d: HTTP %d", appID, resp.StatusCode)
		return false
	}

	// Create the local file
	localPath := s.GetImagePath(appID)
	file, err := os.Create(localPath)
	if err != nil {
		log.Printf("Failed to create image file for game %d: %v", appID, err)
		return false
	}
	defer file.Close()

	// Copy the image data
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Failed to save image for game %d: %v", appID, err)
		// Clean up partial file
		os.Remove(localPath)
		return false
	}

	log.Printf("Cached image for game %d", appID)
	return true
}

// CacheImageAsync downloads and caches a game's header image asynchronously
func (s *ImageCacheService) CacheImageAsync(appID int) {
	go func() {
		s.CacheImage(appID)
	}()
}

// GetBaseDir returns the base directory for cached images
func (s *ImageCacheService) GetBaseDir() string {
	return s.baseDir
}
