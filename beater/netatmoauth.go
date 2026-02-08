package beater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// OAuthErrorResponse represents the error payload returned by Netatmo's OAuth endpoint.
type OAuthErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// AuthError represents an authentication error with context for retry/terminal decisions.
type AuthError struct {
	Terminal              bool   // true = non-recoverable (invalid_grant, revoked app)
	OAuthCode             string // "invalid_grant", "invalid_client", etc.
	Message               string
	AttemptedRefreshToken string // the refresh token we tried to use
}

func (e *AuthError) Error() string { return e.Message }

// isTerminalOAuthError returns true for OAuth error codes that cannot be recovered by retrying.
func isTerminalOAuthError(code string) bool {
	switch code {
	case "invalid_grant", "invalid_client", "unauthorized_client":
		return true
	}
	return false
}

// InitializeTokenState sets up credentials at startup by trying sources in order:
// 1. Token file on disk
// 2. refresh_token from config
// 3. access_token from config (short-lived, no auto-recovery)
// 4. Fail with bootstrap instructions
func (bt *Netatmobeat) InitializeTokenState() error {
	logger := logp.NewLogger(selector)

	// Validate required credentials for refresh-based flows
	needsRefresh := bt.config.RefreshToken != "" || bt.config.TokenFile != ""
	if needsRefresh {
		if bt.config.ClientId == "" {
			return fmt.Errorf("client_id is required when using refresh_token or token_file authentication")
		}
		if bt.config.ClientSecret == "" {
			return fmt.Errorf("client_secret is required when using refresh_token or token_file authentication")
		}
	}

	// Validate token file path is writable before any token operations
	if bt.config.TokenFile != "" {
		if err := ValidateTokenFilePath(bt.config.TokenFile); err != nil {
			return fmt.Errorf("token file path validation failed: %v", err)
		}
	}

	// Warn about deprecated fields
	if bt.config.Username != "" || bt.config.Password != "" {
		logger.Warn("username/password config fields are deprecated and ignored. " +
			"Netatmo removed password-grant authentication in July 2023. " +
			"Use refresh_token or token_file instead.")
	}

	// 1. Try loading from token file
	if bt.config.TokenFile != "" {
		stored, err := LoadTokenFile(bt.config.TokenFile)
		if err == nil && stored.RefreshToken != "" {
			logger.Info("Loaded tokens from file: ", bt.config.TokenFile)
			bt.setCreds(ResponseOauth2Token{
				Access_token:  stored.AccessToken,
				Refresh_token: stored.RefreshToken,
				Expires_in:    stored.ExpiresIn,
				Expire_in:     stored.ExpiresIn,
				Scope:         stored.Scope,
				LastAuthTime:  stored.ObtainedAt,
			})
			// Refresh to get a fresh access token
			if err := bt.RefreshAccessToken(); err != nil {
				return fmt.Errorf("token file loaded but refresh failed: %v", err)
			}
			return nil
		} else if err != nil && !os.IsNotExist(err) {
			logger.Warn("Could not load token file: ", err)
		}
	}

	// 2. Try refresh_token from config
	if bt.config.RefreshToken != "" {
		logger.Info("Using refresh_token from config for initial authentication.")
		bt.setCreds(ResponseOauth2Token{
			Refresh_token: bt.config.RefreshToken,
		})
		if err := bt.RefreshAccessToken(); err != nil {
			return fmt.Errorf("config refresh_token provided but refresh failed: %v", err)
		}
		return nil
	}

	// 3. Try access_token from config (short-lived fallback)
	if bt.config.AccessToken != "" {
		logger.Warn("Using access_token from config. This token is short-lived (~3 hours) " +
			"and cannot be refreshed without a refresh_token. Provide a refresh_token for unattended operation.")
		bt.setCreds(ResponseOauth2Token{
			Access_token: bt.config.AccessToken,
			LastAuthTime: time.Now().UTC().Unix(),
		})
		return nil
	}

	// 4. No token source available
	return fmt.Errorf("no authentication tokens available. To bootstrap:\n" +
		"  1. Go to https://dev.netatmo.com/apps/ and select your app\n" +
		"  2. Use the Token Generator with scope 'read_station'\n" +
		"  3. Copy the refresh_token into netatmobeat.yml")
}

// refreshThreshold returns the number of seconds after which a token should be refreshed.
// Uses the dynamic expires_in from the token response, refreshing 60s before actual expiry.
// Falls back to the hardcoded authExpireThreshold if expires_in is not available.
func refreshThreshold(expiresIn int) int64 {
	if expiresIn > 120 {
		return int64(expiresIn - 60)
	}
	return int64(authExpireThreshold)
}

// EnsureValidToken checks if the current access token is still within its validity window.
// If expired or about to expire, triggers a refresh.
func (bt *Netatmobeat) EnsureValidToken() error {
	creds := bt.getCreds()
	ct := time.Now().UTC().Unix()
	threshold := refreshThreshold(creds.Expires_in)
	if creds.LastAuthTime == 0 || (ct-creds.LastAuthTime) >= threshold {
		return bt.RefreshAccessToken()
	}
	return nil
}

// isAuthError returns true if the HTTP status code indicates an authentication failure.
func isAuthError(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

// RefreshAccessToken exchanges the current refresh token for a new token pair.
// On success, updates in-memory credentials and persists the rotated tokens to disk.
// Uses refreshMu to serialize refresh operations — if another goroutine just refreshed,
// skips the refresh and returns nil.
func (bt *Netatmobeat) RefreshAccessToken() error {
	logger := logp.NewLogger(selector)

	bt.refreshMu.Lock()
	defer bt.refreshMu.Unlock()

	// Re-check after acquiring lock: if another goroutine just refreshed
	// (LastAuthTime is recent), skip this refresh.
	creds := bt.getCreds()
	if creds.LastAuthTime > 0 {
		elapsed := time.Now().UTC().Unix() - creds.LastAuthTime
		if elapsed < authCheckPeriod {
			logger.Debug("Skipping refresh — another goroutine refreshed ", elapsed, "s ago.")
			return nil
		}
	}

	currentCreds := bt.getCreds()
	logger.Debug("Refreshing token. Current refresh_token: ", redactToken(currentCreds.Refresh_token))

	data := url.Values{}
	data.Add("grant_type", "refresh_token")
	data.Add("client_id", bt.config.ClientId)
	data.Add("client_secret", bt.config.ClientSecret)
	data.Add("refresh_token", currentCreds.Refresh_token)

	u, _ := url.ParseRequestURI(bt.apiBaseURL)
	u.Path = authPath
	urlStr := u.String()

	encoded := data.Encode()

	r, err := http.NewRequestWithContext(bt.ctx, http.MethodPost, urlStr, strings.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %v", err)
	}
	r.Header.Add("Content-Type", cookieContentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(encoded)))

	resp, err := bt.httpClient.Do(r)
	if err != nil {
		bt.refreshFailCount++
		return fmt.Errorf("refresh request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		bt.refreshFailCount++
		return fmt.Errorf("failed to read refresh response body: %v", err)
	}

	// Handle non-200 responses with OAuth error parsing
	if resp.StatusCode != http.StatusOK {
		bt.refreshFailCount++
		var oauthErr OAuthErrorResponse
		if jsonErr := json.Unmarshal(body, &oauthErr); jsonErr == nil && oauthErr.Error != "" {
			terminal := isTerminalOAuthError(oauthErr.Error)
			msg := fmt.Sprintf("OAuth error during refresh: %s - %s", oauthErr.Error, oauthErr.Description)
			if terminal {
				msg = fmt.Sprintf("refresh token %s is invalid or expired (%s). "+
					"Re-authorization required: obtain a new token from https://dev.netatmo.com/apps/ "+
					"(error: %s)", redactToken(currentCreds.Refresh_token), oauthErr.Error, oauthErr.Description)
			}
			return &AuthError{
				Terminal:              terminal,
				OAuthCode:             oauthErr.Error,
				Message:               msg,
				AttemptedRefreshToken: currentCreds.Refresh_token,
			}
		}
		return fmt.Errorf("token refresh failed with HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var newCreds ResponseOauth2Token
	if err := json.Unmarshal(body, &newCreds); err != nil {
		bt.refreshFailCount++
		return fmt.Errorf("failed to parse refresh response: %v", err)
	}

	newCreds.LastAuthTime = time.Now().UTC().Unix()
	bt.setCreds(newCreds)

	// Update auth health metrics
	bt.lastRefreshSuccess = newCreds.LastAuthTime
	bt.refreshFailCount = 0

	logger.Info("Token refreshed successfully. Expires in: ", newCreds.Expire_in, "s")

	// Persist rotated tokens to disk
	if bt.config.TokenFile != "" {
		stored := StoredToken{
			AccessToken:  newCreds.Access_token,
			RefreshToken: newCreds.Refresh_token,
			ExpiresIn:    newCreds.Expires_in,
			ObtainedAt:   newCreds.LastAuthTime,
			Scope:        newCreds.Scope,
		}
		if err := SaveTokenFile(bt.config.TokenFile, stored); err != nil {
			bt.persistFailCount++
			if bt.persistFailCount >= 3 {
				logger.Error("Token persistence has failed ", bt.persistFailCount,
					" consecutive times for ", bt.config.TokenFile,
					". If the process restarts, re-authorization will be required. "+
						"Check that the token file path is writable. Error: ", err)
			} else {
				logger.Warn("Failed to persist rotated tokens to ", bt.config.TokenFile,
					" (attempt ", bt.persistFailCount, "): ", err)
			}
		} else {
			if bt.persistFailCount > 0 {
				logger.Info("Token persistence recovered after ", bt.persistFailCount, " failures.")
			}
			bt.persistFailCount = 0
			logger.Debug("Persisted rotated tokens to: ", bt.config.TokenFile)
		}
	}

	return nil
}
