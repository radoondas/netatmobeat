// +build !integration

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/radoondas/netatmobeat/config"
)

func newTestBeat(cfg config.Config) *Netatmobeat {
	ctx, cancel := context.WithCancel(context.Background())
	return &Netatmobeat{
		done:       make(chan struct{}),
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiBaseURL: netatmoApiUrl,
		ctx:        ctx,
		cancel:     cancel,
		creds:      ResponseOauth2Token{},
	}
}

// newTestBeatWithServer creates a test beat pointing at the given httptest server URL.
func newTestBeatWithServer(cfg config.Config, serverURL string) *Netatmobeat {
	ctx, cancel := context.WithCancel(context.Background())
	return &Netatmobeat{
		done:       make(chan struct{}),
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiBaseURL: serverURL,
		ctx:        ctx,
		cancel:     cancel,
		creds:      ResponseOauth2Token{},
	}
}

// TestInitializeTokenState_FromTokenFile verifies the full flow: load token file,
// refresh via mock server, persist rotated tokens back to disk.
func TestInitializeTokenState_FromTokenFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ResponseOauth2Token{
			Access_token:  "new-access",
			Refresh_token: "new-refresh",
			Expires_in:    10800,
			Expire_in:     10800,
			Scope:         []string{"read_station"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	dir, err := ioutil.TempDir("", "auth-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tokenPath := filepath.Join(dir, "tokens.json")
	stored := StoredToken{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresIn:    10800,
		ObtainedAt:   time.Now().UTC().Unix() - 100,
		Scope:        []string{"read_station"},
	}
	if err := SaveTokenFile(tokenPath, stored); err != nil {
		t.Fatal(err)
	}

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test-id",
		ClientSecret: "test-secret",
		TokenFile:    tokenPath,
	}, server.URL)

	err = bt.InitializeTokenState()
	if err != nil {
		t.Fatalf("InitializeTokenState failed: %v", err)
	}

	// Verify in-memory creds were updated by the refresh
	creds := bt.getCreds()
	if creds.Access_token != "new-access" {
		t.Errorf("expected access_token 'new-access', got %q", creds.Access_token)
	}
	if creds.Refresh_token != "new-refresh" {
		t.Errorf("expected refresh_token 'new-refresh', got %q", creds.Refresh_token)
	}
	if creds.LastAuthTime == 0 {
		t.Error("expected LastAuthTime to be set after refresh")
	}

	// Verify tokens were persisted to disk
	onDisk, err := LoadTokenFile(tokenPath)
	if err != nil {
		t.Fatalf("LoadTokenFile failed after refresh: %v", err)
	}
	if onDisk.RefreshToken != "new-refresh" {
		t.Errorf("persisted refresh_token: expected 'new-refresh', got %q", onDisk.RefreshToken)
	}
	if onDisk.AccessToken != "new-access" {
		t.Errorf("persisted access_token: expected 'new-access', got %q", onDisk.AccessToken)
	}
}

// TestInitializeTokenState_FromConfigRefreshToken verifies that when no token file
// exists, the config refresh_token is used and refresh succeeds via mock server.
func TestInitializeTokenState_NoTokenFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ResponseOauth2Token{
			Access_token:  "refreshed-access",
			Refresh_token: "refreshed-refresh",
			Expires_in:    10800,
			Expire_in:     10800,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test-id",
		ClientSecret: "test-secret",
		TokenFile:    "", // no token file
		RefreshToken: "config-refresh-token",
	}, server.URL)

	err := bt.InitializeTokenState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	creds := bt.getCreds()
	if creds.Access_token != "refreshed-access" {
		t.Errorf("expected 'refreshed-access', got %q", creds.Access_token)
	}
	if creds.Refresh_token != "refreshed-refresh" {
		t.Errorf("expected 'refreshed-refresh', got %q", creds.Refresh_token)
	}
}

// TestInitializeTokenState_FromConfigAccessToken verifies fallback to access_token.
func TestInitializeTokenState_AccessTokenOnly(t *testing.T) {
	bt := newTestBeat(config.Config{
		TokenFile:   "",
		AccessToken: "my-access-token",
	})

	err := bt.InitializeTokenState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	creds := bt.getCreds()
	if creds.Access_token != "my-access-token" {
		t.Errorf("expected access_token 'my-access-token', got %q", creds.Access_token)
	}
	if creds.LastAuthTime == 0 {
		t.Error("expected LastAuthTime to be set")
	}
}

// TestInitializeTokenState_NoTokenSource verifies failure with bootstrap instructions.
func TestInitializeTokenState_NoTokenSource(t *testing.T) {
	bt := newTestBeat(config.Config{
		TokenFile: "",
	})

	err := bt.InitializeTokenState()
	if err == nil {
		t.Fatal("expected error when no token source available")
	}

	expected := "no authentication tokens available"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error starting with %q, got %q", expected, err.Error())
	}
}

// TestEnsureValidToken_Fresh verifies that a recently obtained token doesn't trigger refresh.
func TestEnsureValidToken_Fresh(t *testing.T) {
	bt := newTestBeat(config.Config{})
	bt.setCreds(ResponseOauth2Token{
		Access_token: "valid-token",
		LastAuthTime: time.Now().UTC().Unix(),
	})

	// EnsureValidToken should not try to refresh a fresh token.
	// Since there's no refresh_token, if it tried to refresh it would fail.
	// A nil error means it correctly skipped the refresh.
	err := bt.EnsureValidToken()
	if err != nil {
		t.Errorf("expected no error for fresh token, got: %v", err)
	}
}

// TestEnsureValidToken_Expired verifies that an expired token triggers refresh.
func TestEnsureValidToken_Expired(t *testing.T) {
	bt := newTestBeat(config.Config{
		ClientId:     "test",
		ClientSecret: "test",
	})
	bt.setCreds(ResponseOauth2Token{
		Access_token:  "expired-token",
		Refresh_token: "some-refresh",
		LastAuthTime:  time.Now().UTC().Unix() - 20000, // well past threshold
	})

	// Will try to refresh but fail since no real server exists
	err := bt.EnsureValidToken()
	if err == nil {
		t.Error("expected error when refreshing expired token against no server")
	}
}

// TestIsAuthError verifies auth error detection.
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, true},
		{403, true},
		{404, false},
		{500, false},
	}

	for _, tc := range tests {
		got := isAuthError(tc.code)
		if got != tc.want {
			t.Errorf("isAuthError(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestOAuthErrorResponseParsing verifies JSON parsing of error responses.
func TestOAuthErrorResponseParsing(t *testing.T) {
	body := `{"error":"invalid_grant","error_description":"Token is expired"}`

	var oauthErr OAuthErrorResponse
	if err := json.Unmarshal([]byte(body), &oauthErr); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if oauthErr.Error != "invalid_grant" {
		t.Errorf("Error: got %q, want 'invalid_grant'", oauthErr.Error)
	}
	if oauthErr.Description != "Token is expired" {
		t.Errorf("Description: got %q, want 'Token is expired'", oauthErr.Description)
	}
}

// TestGetCredsSetCreds verifies thread-safe credential access.
func TestGetCredsSetCreds(t *testing.T) {
	bt := newTestBeat(config.Config{})

	creds := bt.getCreds()
	if creds.Access_token != "" {
		t.Error("expected empty access token initially")
	}

	bt.setCreds(ResponseOauth2Token{
		Access_token:  "test-access",
		Refresh_token: "test-refresh",
	})

	creds = bt.getCreds()
	if creds.Access_token != "test-access" {
		t.Errorf("expected 'test-access', got %q", creds.Access_token)
	}

	token := bt.getAccessToken()
	if token != "test-access" {
		t.Errorf("getAccessToken: expected 'test-access', got %q", token)
	}
}

// --- Phase 2 tests ---

// TestRefreshAccessToken_FullFlow verifies the full refresh cycle via mock server:
// sends correct form data, updates in-memory creds, persists to disk.
func TestRefreshAccessToken_FullFlow(t *testing.T) {
	var receivedGrantType, receivedClientID, receivedRefreshToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedGrantType = r.FormValue("grant_type")
		receivedClientID = r.FormValue("client_id")
		receivedRefreshToken = r.FormValue("refresh_token")

		resp := ResponseOauth2Token{
			Access_token:  "fresh-access",
			Refresh_token: "fresh-refresh",
			Expires_in:    10800,
			Expire_in:     10800,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	dir, err := ioutil.TempDir("", "refresh-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	tokenPath := filepath.Join(dir, "tokens.json")

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "my-client",
		ClientSecret: "my-secret",
		TokenFile:    tokenPath,
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "old-refresh",
		LastAuthTime:  time.Now().UTC().Unix() - 200, // old enough to not skip
	})

	err = bt.RefreshAccessToken()
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// Verify correct form data was sent
	if receivedGrantType != "refresh_token" {
		t.Errorf("grant_type: expected 'refresh_token', got %q", receivedGrantType)
	}
	if receivedClientID != "my-client" {
		t.Errorf("client_id: expected 'my-client', got %q", receivedClientID)
	}
	if receivedRefreshToken != "old-refresh" {
		t.Errorf("refresh_token: expected 'old-refresh', got %q", receivedRefreshToken)
	}

	// Verify in-memory creds
	creds := bt.getCreds()
	if creds.Access_token != "fresh-access" {
		t.Errorf("access_token: expected 'fresh-access', got %q", creds.Access_token)
	}
	if creds.Refresh_token != "fresh-refresh" {
		t.Errorf("refresh_token: expected 'fresh-refresh', got %q", creds.Refresh_token)
	}

	// Verify auth health metrics
	if bt.refreshFailCount != 0 {
		t.Errorf("refreshFailCount: expected 0 after success, got %d", bt.refreshFailCount)
	}
	if bt.lastRefreshSuccess == 0 {
		t.Error("lastRefreshSuccess should be set after success")
	}

	// Verify persisted to disk
	onDisk, err := LoadTokenFile(tokenPath)
	if err != nil {
		t.Fatalf("LoadTokenFile failed: %v", err)
	}
	if onDisk.RefreshToken != "fresh-refresh" {
		t.Errorf("persisted refresh_token: expected 'fresh-refresh', got %q", onDisk.RefreshToken)
	}
}

// TestRefreshAccessToken_TerminalError verifies that invalid_grant returns a terminal AuthError.
func TestRefreshAccessToken_TerminalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid_grant","error_description":"Token has been revoked"}`)
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test-id",
		ClientSecret: "test-secret",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "revoked-token",
		LastAuthTime:  time.Now().UTC().Unix() - 200,
	})

	err := bt.RefreshAccessToken()
	if err == nil {
		t.Fatal("expected error for invalid_grant")
	}

	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if !authErr.Terminal {
		t.Error("expected Terminal=true for invalid_grant")
	}
	if authErr.OAuthCode != "invalid_grant" {
		t.Errorf("OAuthCode: expected 'invalid_grant', got %q", authErr.OAuthCode)
	}
	if authErr.AttemptedRefreshToken != "revoked-token" {
		t.Errorf("AttemptedRefreshToken: expected 'revoked-token', got %q", authErr.AttemptedRefreshToken)
	}
}

// TestRefreshAccessToken_TransientError verifies that HTTP 500 returns a regular error, not terminal.
func TestRefreshAccessToken_TransientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test-id",
		ClientSecret: "test-secret",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "valid-token",
		LastAuthTime:  time.Now().UTC().Unix() - 200,
	})

	err := bt.RefreshAccessToken()
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}

	// Should NOT be a terminal AuthError
	if authErr, ok := err.(*AuthError); ok && authErr.Terminal {
		t.Error("HTTP 500 should not be a terminal error")
	}

	// Verify failure count incremented
	if bt.refreshFailCount != 1 {
		t.Errorf("refreshFailCount: expected 1, got %d", bt.refreshFailCount)
	}
}

// TestRefreshAccessToken_ConcurrentStampede verifies that concurrent refresh calls
// result in only one actual HTTP request to the server.
func TestRefreshAccessToken_ConcurrentStampede(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Small delay to ensure concurrent goroutines pile up on the mutex
		time.Sleep(50 * time.Millisecond)

		resp := ResponseOauth2Token{
			Access_token:  "stamped-access",
			Refresh_token: "stamped-refresh",
			Expires_in:    10800,
			Expire_in:     10800,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test-id",
		ClientSecret: "test-secret",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "old-token",
		LastAuthTime:  time.Now().UTC().Unix() - 200, // old enough to trigger refresh
	})

	// Launch 5 concurrent refresh attempts
	var wg sync.WaitGroup
	errCh := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- bt.RefreshAccessToken()
		}()
	}
	wg.Wait()
	close(errCh)

	// All should succeed (either refreshed or skipped due to mutex re-check)
	for err := range errCh {
		if err != nil {
			t.Errorf("concurrent refresh returned error: %v", err)
		}
	}

	// Only 1 request should have reached the server (the rest skip via re-check)
	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected 1 server request, got %d (stampede not prevented)", count)
	}

	// All goroutines should see the new token
	creds := bt.getCreds()
	if creds.Access_token != "stamped-access" {
		t.Errorf("expected 'stamped-access', got %q", creds.Access_token)
	}
}

// TestRefreshAccessToken_SkipsRecentRefresh verifies that if another goroutine just
// refreshed, a second call skips the refresh.
func TestRefreshAccessToken_SkipsRecentRefresh(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		resp := ResponseOauth2Token{
			Access_token:  "first-access",
			Refresh_token: "first-refresh",
			Expires_in:    10800,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test",
		ClientSecret: "test",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "old",
		LastAuthTime:  time.Now().UTC().Unix() - 200,
	})

	// First refresh should go through
	if err := bt.RefreshAccessToken(); err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	// Second refresh immediately after should skip (LastAuthTime is recent)
	if err := bt.RefreshAccessToken(); err != nil {
		t.Fatalf("second refresh failed: %v", err)
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected 1 server request, got %d", count)
	}
}

// TestRefreshAccessToken_PersistFailureDegraded verifies that persist failure
// tracks degraded state but doesn't fail the refresh itself.
func TestRefreshAccessToken_PersistFailureDegraded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ResponseOauth2Token{
			Access_token:  "access",
			Refresh_token: "refresh",
			Expires_in:    10800,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Point token file at a non-writable path (directory that doesn't exist)
	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test",
		ClientSecret: "test",
		TokenFile:    "/nonexistent-dir/subdir/tokens.json",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "old",
		LastAuthTime:  time.Now().UTC().Unix() - 200,
	})

	// Refresh should succeed (in-memory) even though persist fails
	err := bt.RefreshAccessToken()
	if err != nil {
		t.Fatalf("refresh should succeed even with persist failure: %v", err)
	}

	// Verify in-memory creds are updated
	creds := bt.getCreds()
	if creds.Access_token != "access" {
		t.Errorf("expected 'access', got %q", creds.Access_token)
	}

	// Verify persist failure was tracked
	if bt.persistFailCount != 1 {
		t.Errorf("persistFailCount: expected 1, got %d", bt.persistFailCount)
	}
}

// TestRefreshThreshold verifies the dynamic threshold calculation.
func TestRefreshThreshold(t *testing.T) {
	tests := []struct {
		expiresIn int
		want      int64
	}{
		{10800, 10740},                    // 3 hours -> refresh 60s before
		{3600, 3540},                      // 1 hour -> refresh 60s before
		{121, 61},                         // just above 120 -> 61s
		{120, int64(authExpireThreshold)}, // at boundary -> fallback
		{60, int64(authExpireThreshold)},  // below boundary -> fallback
		{0, int64(authExpireThreshold)},   // zero -> fallback
	}

	for _, tc := range tests {
		got := refreshThreshold(tc.expiresIn)
		if got != tc.want {
			t.Errorf("refreshThreshold(%d) = %d, want %d", tc.expiresIn, got, tc.want)
		}
	}
}

// TestIsTerminalOAuthError verifies terminal error classification.
func TestIsTerminalOAuthError(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"invalid_grant", true},
		{"invalid_client", true},
		{"unauthorized_client", true},
		{"invalid_request", false},
		{"server_error", false},
		{"", false},
	}

	for _, tc := range tests {
		got := isTerminalOAuthError(tc.code)
		if got != tc.want {
			t.Errorf("isTerminalOAuthError(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestRedactToken verifies token masking for log safety.
func TestRedactToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdefghijklmnop", "abcd***"},
		{"abcdefghi", "abcd***"},
		{"abcdefgh", "***"}, // exactly 8 chars, not > 8
		{"short", "***"},
		{"a", "***"},
		{"", "<empty>"},
	}

	for _, tc := range tests {
		got := redactToken(tc.input)
		if got != tc.want {
			t.Errorf("redactToken(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestBackoffWithJitter verifies exponential backoff bounds and cap.
func TestBackoffWithJitter(t *testing.T) {
	// Test doubling
	d := backoffWithJitter(60)
	secs := int(d.Seconds())
	// 60*2 = 120, with ±20% jitter → [96, 144], but capped at min authCheckPeriod (60)
	if secs < 96 || secs > 144 {
		t.Errorf("backoff(60) = %ds, expected [96, 144]", secs)
	}

	// Test cap at maxBackoff (15 min = 900s)
	d = backoffWithJitter(600)
	secs = int(d.Seconds())
	// 600*2 = 1200, capped to 900, with ±20% jitter → [720, 1080]
	if secs < 720 || secs > 1080 {
		t.Errorf("backoff(600) = %ds, expected [720, 1080]", secs)
	}

	// Test minimum floor (authCheckPeriod = 60)
	d = backoffWithJitter(10)
	secs = int(d.Seconds())
	// 10*2 = 20, with jitter could go as low as 16, but floor is 60
	if secs < authCheckPeriod {
		t.Errorf("backoff(10) = %ds, should not go below %d", secs, authCheckPeriod)
	}
}

// TestGetCreds_DeepCopiesScope verifies that getCreds returns an independent slice.
func TestGetCreds_DeepCopiesScope(t *testing.T) {
	bt := newTestBeat(config.Config{})
	bt.setCreds(ResponseOauth2Token{
		Access_token: "test",
		Scope:        []string{"read_station", "read_thermostat"},
	})

	creds1 := bt.getCreds()
	creds2 := bt.getCreds()

	// Mutate creds1's scope — should not affect creds2 or the original
	creds1.Scope[0] = "mutated"

	if creds2.Scope[0] == "mutated" {
		t.Error("getCreds returned a shared slice — deep copy not working")
	}

	original := bt.getCreds()
	if original.Scope[0] == "mutated" {
		t.Error("mutation of returned creds affected the original")
	}
}

// TestGetCreds_NilScope verifies getCreds handles nil Scope gracefully.
func TestGetCreds_NilScope(t *testing.T) {
	bt := newTestBeat(config.Config{})
	bt.setCreds(ResponseOauth2Token{
		Access_token: "test",
		Scope:        nil,
	})

	creds := bt.getCreds()
	if creds.Scope != nil {
		t.Errorf("expected nil Scope, got %v", creds.Scope)
	}
}

// TestInitializeTokenState_MissingClientId verifies config validation.
func TestInitializeTokenState_MissingClientId(t *testing.T) {
	bt := newTestBeat(config.Config{
		ClientId:     "",
		ClientSecret: "secret",
		RefreshToken: "some-token",
	})

	err := bt.InitializeTokenState()
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}
	expected := "client_id is required"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// TestInitializeTokenState_MissingClientSecret verifies config validation.
func TestInitializeTokenState_MissingClientSecret(t *testing.T) {
	bt := newTestBeat(config.Config{
		ClientId:     "id",
		ClientSecret: "",
		RefreshToken: "some-token",
	})

	err := bt.InitializeTokenState()
	if err == nil {
		t.Fatal("expected error for missing client_secret")
	}
	expected := "client_secret is required"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// TestInitializeTokenState_NonWritableTokenFile verifies startup validation.
func TestInitializeTokenState_NonWritableTokenFile(t *testing.T) {
	bt := newTestBeat(config.Config{
		ClientId:     "id",
		ClientSecret: "secret",
		TokenFile:    "/nonexistent-readonly-path/subdir/tokens.json",
		RefreshToken: "some-token",
	})

	err := bt.InitializeTokenState()
	if err == nil {
		t.Fatal("expected error for non-writable token file path")
	}
	expected := "token file path validation failed"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// TestContextCancellation verifies that cancelling the context aborts HTTP requests.
func TestContextCancellation(t *testing.T) {
	// Server that blocks for a long time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer server.Close()

	bt := newTestBeatWithServer(config.Config{
		ClientId:     "test",
		ClientSecret: "test",
	}, server.URL)
	bt.setCreds(ResponseOauth2Token{
		Refresh_token: "test",
		LastAuthTime:  time.Now().UTC().Unix() - 200,
	})

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		bt.cancel()
	}()

	start := time.Now()
	err := bt.RefreshAccessToken()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}

	// Should complete well before the 10s server delay and 5s client timeout
	if elapsed > 2*time.Second {
		t.Errorf("cancellation took too long: %v (should be <2s)", elapsed)
	}
}

// TestGetCredsSetCreds_Concurrent verifies no race conditions under concurrent access.
func TestGetCredsSetCreds_Concurrent(t *testing.T) {
	bt := newTestBeat(config.Config{})

	var wg sync.WaitGroup
	// 10 writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bt.setCreds(ResponseOauth2Token{
				Access_token:  fmt.Sprintf("access-%d", n),
				Refresh_token: fmt.Sprintf("refresh-%d", n),
				Scope:         []string{"read_station"},
			})
		}(i)
	}
	// 10 readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			creds := bt.getCreds()
			_ = creds.Access_token
			_ = bt.getAccessToken()
		}()
	}
	wg.Wait()

	// If we get here without a race detector panic, the test passes.
	// Just verify we can still read creds.
	creds := bt.getCreds()
	if creds.Access_token == "" {
		t.Error("expected non-empty access token after concurrent writes")
	}
}
