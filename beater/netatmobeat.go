package beater

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/radoondas/netatmobeat/config"
)

const (
	netatmoApiUrl = "https://api.netatmo.com"
	authPath      = "/oauth2/token"

	cookieContentType = "application/x-www-form-urlencoded;charset=UTF-8"

	selector = "netatmobeat"

	authExpireThreshold = 10740
	authCheckPeriod     = 60

	maxBackoff = 15 * 60 // 15 minutes in seconds
	jitterPct  = 0.2     // ±20% jitter
)

type Netatmobeat struct {
	done             chan struct{}
	config           config.Config
	client           beat.Client
	httpClient       *http.Client
	apiBaseURL       string             // base URL for Netatmo API; defaults to netatmoApiUrl, injectable for tests
	ctx              context.Context    // cancelled on Stop() to abort in-flight HTTP calls
	cancel           context.CancelFunc // cancels ctx
	creds            ResponseOauth2Token
	credsMu          sync.RWMutex
	refreshMu        sync.Mutex // serializes refresh operations to prevent stampede
	persistFailCount int        // consecutive token persist failures, protected by refreshMu

	// Auth health metrics (protected by refreshMu)
	lastRefreshSuccess int64 // unix timestamp of last successful refresh
	refreshFailCount   int   // consecutive refresh failures, reset on success
}

// redactToken masks a token string for safe logging.
func redactToken(s string) string {
	if len(s) > 8 {
		return s[:4] + "***"
	}
	if len(s) > 0 {
		return "***"
	}
	return "<empty>"
}

// backoffWithJitter returns a duration with exponential backoff and ±20% jitter.
func backoffWithJitter(currentSeconds int) time.Duration {
	next := currentSeconds * 2
	if next > maxBackoff {
		next = maxBackoff
	}
	jitter := float64(next) * jitterPct
	delta := int(jitter*2) + 1
	actual := next - int(jitter) + rand.Intn(delta)
	if actual < authCheckPeriod {
		actual = authCheckPeriod
	}
	return time.Duration(actual) * time.Second
}

// getCreds returns a deep copy of the current credentials under a read lock.
func (bt *Netatmobeat) getCreds() ResponseOauth2Token {
	bt.credsMu.RLock()
	defer bt.credsMu.RUnlock()
	c := bt.creds
	if bt.creds.Scope != nil {
		c.Scope = make([]string, len(bt.creds.Scope))
		copy(c.Scope, bt.creds.Scope)
	}
	return c
}

// setCreds replaces the current credentials under a write lock.
func (bt *Netatmobeat) setCreds(c ResponseOauth2Token) {
	bt.credsMu.Lock()
	defer bt.credsMu.Unlock()
	bt.creds = c
}

// getAccessToken returns the current access token under a read lock.
func (bt *Netatmobeat) getAccessToken() string {
	bt.credsMu.RLock()
	defer bt.credsMu.RUnlock()
	return bt.creds.Access_token
}

type ResponseOauth2Token struct {
	Access_token  string   `json:"access_token"`
	Refresh_token string   `json:"refresh_token"`
	Scope         []string `json:"scope"`
	Expires_in    int      `json:"expires_in"`
	Expire_in     int      `json:"expire_in"`
	LastAuthTime  int64
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	bt := &Netatmobeat{
		done:   make(chan struct{}),
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
			},
		},
		apiBaseURL: netatmoApiUrl,
		creds:      ResponseOauth2Token{},
	}
	return bt, nil
}

func (bt *Netatmobeat) Run(b *beat.Beat) error {
	logp.NewLogger(selector).Info("Netatmobeat is running! Hit CTRL-C to stop it.")

	bt.ctx, bt.cancel = context.WithCancel(context.Background())

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	err = bt.InitializeTokenState()
	if err != nil {
		return fmt.Errorf("failed to initialize authentication: %v", err)
	}

	go func() {
		logger := logp.NewLogger(selector)
		currentBackoff := authCheckPeriod // seconds
		timer := time.NewTimer(time.Duration(currentBackoff) * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-bt.done:
				return
			case <-timer.C:
			}

			creds := bt.getCreds()
			ct := time.Now().UTC().Unix()
			threshold := refreshThreshold(creds.Expires_in)
			logger.Debug("Time difference for refresh: ", ct-creds.LastAuthTime, " threshold: ", threshold)

			if (ct - creds.LastAuthTime) >= threshold {
				if err := bt.RefreshAccessToken(); err != nil {
					if authErr, ok := err.(*AuthError); ok && authErr.Terminal {
						// Check if another goroutine already rotated the token
						currentToken := bt.getCreds().Refresh_token
						if currentToken != authErr.AttemptedRefreshToken && currentToken != "" {
							logger.Warn("Terminal auth error, but token was rotated by another goroutine. Will retry next cycle.")
							currentBackoff = authCheckPeriod
						} else {
							logger.Error("Terminal authentication failure: ", err,
								". Stopping refresh loop. Re-bootstrap required.")
							return
						}
					} else {
						// Transient error: apply exponential backoff
						wait := backoffWithJitter(currentBackoff)
						currentBackoff = int(wait.Seconds())
						logger.Error("Token refresh failed (retrying in ", wait, "): ", err)
						timer.Reset(wait)
						continue
					}
				} else {
					// Success: reset backoff
					currentBackoff = authCheckPeriod
				}
			}

			timer.Reset(time.Duration(authCheckPeriod) * time.Second)
		}
	}()

	// run only if public weather is enabled
	if bt.config.PublicWeather.Enabled {
		// for each reagion
		for _, region := range bt.config.PublicWeather.Regions {
			logp.NewLogger(selector).Info("* Region: ", region.Name, " Enabled: ", region.Enabled)
			if region.Enabled {
				go func(region config.Region) {
					ticker := time.NewTicker(bt.config.PublicWeather.Period)
					defer ticker.Stop()

					for {
						logp.NewLogger(selector).Debug("** Region: ", region.Description, " Name: ", region.Name)
						err := bt.GetRegionData(region)

						if err != nil {
							//TODO: return?
							logp.NewLogger(selector).Error(err)
						}

						select {
						case <-bt.done:
							goto GotoFinish
						case <-ticker.C:
						}
					}
				GotoFinish:
				}(region)
			}
		}
	}

	// run only if station's data are enabled
	if bt.config.WeatherStations.Enabled {
		for _, stationID := range bt.config.WeatherStations.Ids {
			stid := stationID
			go func() {
				ticker := time.NewTicker(bt.config.WeatherStations.Period)
				defer ticker.Stop()

				for {
					err := bt.GetStationsData(stid)
					if err != nil {
						//TODO: return?
						logp.NewLogger(selector).Error(err)
					}

					select {
					case <-bt.done:
						goto GotoFinish
					case <-ticker.C:
					}
				}

			GotoFinish:
			}()
		}
	} else {
		logp.NewLogger(selector).Info("Weather station data not enabled.")
	}

	<-bt.done
	return nil
}

func (bt *Netatmobeat) Stop() {
	logp.NewLogger(selector).Info("Stopping Netatmobeat.")
	if bt.cancel != nil {
		bt.cancel() // cancel in-flight HTTP requests
	}
	bt.client.Close()
	close(bt.done)
}
