// netatmo-token is a standalone CLI for obtaining and verifying Netatmo OAuth2 tokens.
//
// Usage:
//
//   netatmo-token generate --client-id=ID --client-secret=SECRET [--port=9876] [--save=tokens.json]
//   netatmo-token verify   --client-id=ID --client-secret=SECRET --refresh-token=TOKEN [--save=tokens.json]
//   netatmo-token verify   --client-id=ID --client-secret=SECRET --token-file=path    [--save=tokens.json]
//
// The generate command runs the full OAuth2 authorization code flow:
//   1. Starts a local HTTP server for the callback
//   2. Opens (or prints) the Netatmo authorization URL
//   3. Exchanges the authorization code for tokens
//   4. Prints and optionally saves the tokens
//
// The verify command tests an existing refresh token by exchanging it for a new
// token pair. WARNING: Netatmo rotates refresh tokens on every use, so verifying
// invalidates the old token. Always save the new one.
//
// Build (requires Go 1.21+ for Apple Silicon; pure stdlib, no vendor deps):
//
//   # From a temp directory (avoids GOPATH cache conflicts):
//   mkdir /tmp/netatmo-token-build && cp main.go /tmp/netatmo-token-build/
//   cd /tmp/netatmo-token-build && go mod init netatmo-token
//   CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o netatmo-token .
//   cp netatmo-token /path/to/netatmobeat/ && rm -rf /tmp/netatmo-token-build
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	netatmoAuthURL  = "https://api.netatmo.com/oauth2/authorize"
	netatmoTokenURL = "https://api.netatmo.com/oauth2/token"
	defaultScope    = "read_station"
)

// tokenResponse represents the Netatmo OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	Scope        []string `json:"scope"`
}

// oauthError represents an error response from the Netatmo token endpoint.
type oauthError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// storedToken matches the format used by netatmobeat's token file (tokenstore.go).
type storedToken struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	ObtainedAt   int64    `json:"obtained_at_unix"`
	Scope        []string `json:"scope"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "verify":
		cmdVerify(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Netatmo Token Manager — obtain and verify OAuth2 tokens for netatmobeat

Usage:
  netatmo-token generate  Obtain new tokens via browser-based OAuth2 flow
  netatmo-token verify    Validate an existing refresh token (rotates it!)

Run 'netatmo-token <command> -h' for command-specific options.
`)
}

// cmdGenerate runs the OAuth2 authorization code flow via a local callback server.
func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	clientID := fs.String("client-id", "", "Netatmo app client ID (required)")
	clientSecret := fs.String("client-secret", "", "Netatmo app client secret (required)")
	port := fs.Int("port", 9876, "Local port for OAuth2 callback")
	scope := fs.String("scope", defaultScope, "OAuth2 scope(s), space-separated")
	saveFile := fs.String("save", "", "Save tokens to file (netatmobeat token file format)")
	fs.Parse(args)

	if *clientID == "" || *clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Error: --client-id and --client-secret are required")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
		os.Exit(1)
	}

	state := randomState()
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", *port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local callback server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch in OAuth callback")
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("Authorization denied: %s — %s", errMsg, desc), http.StatusBadRequest)
			errCh <- fmt.Errorf("authorization denied: %s — %s", errMsg, desc)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code in callback")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2>"+
			"<p>You can close this tab and return to the terminal.</p></body></html>")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot listen on port %d: %v\n", *port, err)
		os.Exit(1)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	// Build authorization URL
	params := url.Values{
		"client_id":     {*clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {*scope},
		"state":         {state},
		"response_type": {"code"},
	}
	authURL := netatmoAuthURL + "?" + params.Encode()

	fmt.Println("Open this URL in your browser to authorize:")
	fmt.Println()
	fmt.Println("  ", authURL)
	fmt.Println()
	fmt.Println("Waiting for authorization callback on", redirectURI, "...")

	openBrowser(authURL)

	// Wait for callback or timeout
	var code string
	select {
	case code = <-codeCh:
		// success
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	case <-time.After(5 * time.Minute):
		fmt.Fprintln(os.Stderr, "\nError: timed out waiting for authorization (5 minutes)")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Authorization code received. Exchanging for tokens...")

	tokens, err := exchangeCode(*clientID, *clientSecret, code, redirectURI, *scope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error exchanging authorization code: %v\n", err)
		os.Exit(1)
	}

	printTokens(tokens)

	if *saveFile != "" {
		if err := saveTokens(*saveFile, tokens); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving token file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Tokens saved to: %s\n", *saveFile)
	}
}

// cmdVerify tests an existing refresh token by exchanging it with Netatmo.
func cmdVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	clientID := fs.String("client-id", "", "Netatmo app client ID (required)")
	clientSecret := fs.String("client-secret", "", "Netatmo app client secret (required)")
	refreshToken := fs.String("refresh-token", "", "Refresh token to verify")
	tokenFile := fs.String("token-file", "", "Path to netatmobeat token file (alternative to --refresh-token)")
	saveFile := fs.String("save", "", "Save rotated tokens to file (default: overwrites --token-file if used)")
	fs.Parse(args)

	if *clientID == "" || *clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Error: --client-id and --client-secret are required")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
		os.Exit(1)
	}

	token := *refreshToken

	// Load from token file if --refresh-token not provided
	if token == "" && *tokenFile != "" {
		data, err := ioutil.ReadFile(*tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading token file: %v\n", err)
			os.Exit(1)
		}
		var stored storedToken
		if err := json.Unmarshal(data, &stored); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing token file: %v\n", err)
			os.Exit(1)
		}
		token = stored.RefreshToken
		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: token file does not contain a refresh_token")
			os.Exit(1)
		}
		fmt.Printf("Loaded refresh token from: %s\n", *tokenFile)
	}

	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: provide --refresh-token or --token-file")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("WARNING: Netatmo rotates refresh tokens on every use.")
	fmt.Println("Verifying will INVALIDATE the current token and issue a new one.")
	fmt.Println("Make sure to save the new token (use --save or copy from output).")
	fmt.Println()

	redacted := token
	if len(redacted) > 8 {
		redacted = redacted[:8] + "***"
	}
	fmt.Printf("Verifying refresh token: %s\n\n", redacted)

	tokens, err := doRefresh(*clientID, *clientSecret, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS — token is valid!")
	fmt.Println()
	fmt.Printf("  new refresh_token: %s\n", tokens.RefreshToken)
	fmt.Printf("  expires_in:        %ds\n", tokens.ExpiresIn)
	fmt.Printf("  scope:             %s\n", strings.Join(tokens.Scope, " "))
	fmt.Println()
	fmt.Println("IMPORTANT: Your old refresh token is now invalidated.")
	fmt.Println("Update your netatmobeat.yml:")
	fmt.Println()
	fmt.Printf("  refresh_token: \"%s\"\n", tokens.RefreshToken)
	fmt.Println()

	// Determine save destination
	saveDest := *saveFile
	if saveDest == "" && *tokenFile != "" {
		saveDest = *tokenFile // overwrite the source file by default
	}
	if saveDest != "" {
		if err := saveTokens(saveDest, tokens); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving token file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Rotated tokens saved to: %s\n", saveDest)
	}
}

// --- Token endpoint helpers ---

func exchangeCode(clientID, clientSecret, code, redirectURI, scope string) (*tokenResponse, error) {
	return doTokenRequest(url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"scope":         {scope},
	})
}

func doRefresh(clientID, clientSecret, refreshToken string) (*tokenResponse, error) {
	return doTokenRequest(url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	})
}

func doTokenRequest(data url.Values) (*tokenResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(netatmoTokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oErr oauthError
		if json.Unmarshal(body, &oErr) == nil && oErr.Error != "" {
			return nil, fmt.Errorf("%s: %s", oErr.Error, oErr.Description)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	return &tokens, nil
}

// --- Helpers ---

func printTokens(tokens *tokenResponse) {
	fmt.Println()
	fmt.Println("=== Tokens obtained successfully ===")
	fmt.Println()
	if len(tokens.AccessToken) > 12 {
		fmt.Printf("  access_token:  %s...%s\n", tokens.AccessToken[:8], tokens.AccessToken[len(tokens.AccessToken)-4:])
	} else {
		fmt.Printf("  access_token:  %s\n", tokens.AccessToken)
	}
	fmt.Printf("  refresh_token: %s\n", tokens.RefreshToken)
	fmt.Printf("  expires_in:    %ds\n", tokens.ExpiresIn)
	fmt.Printf("  scope:         %s\n", strings.Join(tokens.Scope, " "))
	fmt.Println()
	fmt.Println("Add this to your netatmobeat.yml:")
	fmt.Println()
	fmt.Printf("  refresh_token: \"%s\"\n", tokens.RefreshToken)
	fmt.Println()
}

func saveTokens(path string, tokens *tokenResponse) error {
	stored := storedToken{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		ObtainedAt:   time.Now().UTC().Unix(),
		Scope:        tokens.Scope,
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, append(data, '\n'), 0600)
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}
