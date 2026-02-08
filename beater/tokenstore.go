package beater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StoredToken represents the token data persisted to disk.
type StoredToken struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	ObtainedAt   int64    `json:"obtained_at_unix"`
	Scope        []string `json:"scope"`
}

// ValidateTokenFilePath checks that the directory for the token file exists and is writable.
// Creates the directory if it doesn't exist. Returns an error if the path is not usable.
func ValidateTokenFilePath(path string) error {
	dir := filepath.Dir(path)

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("token file directory %s is not accessible: %w", dir, err)
	}

	// Probe-write a temp file to verify writability
	tmpFile, err := os.CreateTemp(dir, ".netatmobeat-probe-*.tmp")
	if err != nil {
		return fmt.Errorf("token file directory %s is not writable: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath)

	return nil
}

// LoadTokenFile reads a stored token from a JSON file at the given path.
// Returns an error if the file does not exist or cannot be parsed.
func LoadTokenFile(path string) (*StoredToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file %s: %w", path, err)
	}

	var token StoredToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file %s: %w", path, err)
	}

	return &token, nil
}

// SaveTokenFile writes the token data as JSON to the given path.
// Uses atomic write (temp file + rename) to prevent corruption on crash.
// File is created with 0600 permissions (owner read/write only).
func SaveTokenFile(path string, token StoredToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".netatmobeat-token-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", dir, err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write token temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close token temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set token file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename token file into place: %w", err)
	}

	return nil
}
