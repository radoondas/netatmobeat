package beater

import (
	"fmt"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/radoondas/netatmobeat/config"
	"log"
	"time"
)

const (
	netatmoApiUrl = "https://api.netatmo.com"
	authPath      = "/oauth2/token"

	cookieContentType = "application/x-www-form-urlencoded;charset=UTF-8"

	selector = "netatmobeat"

	authExpireThreshold = 10740
	authCheckPeriod     = 60
)

type Netatmobeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
	creds  ResponseOauth2Token
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
		creds:  ResponseOauth2Token{},
	}
	return bt, nil
}

func (bt *Netatmobeat) Run(b *beat.Beat) error {
	logp.NewLogger(selector).Info("Netatmobeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	err = bt.GetAccessToken()
	if err != nil {
		log.Fatal(err)
		return err
	}

	go func() {
		// Hardcoded check period
		ticker := time.NewTicker(authCheckPeriod * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-bt.done:
				goto GotoFinish
			case <-ticker.C:
			}

			ct := time.Now().UTC().Unix()
			logp.NewLogger(selector).Debug("Time difference for refresh: ", ct-bt.creds.LastAuthTime)
			if (ct - bt.creds.LastAuthTime) >= authExpireThreshold {
				bt.RefreshAccessToken()
			}
		}
	GotoFinish:
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
						select {
						case <-bt.done:
							goto GotoFinish
						case <-ticker.C:
						}

						logp.NewLogger(selector).Debug("** Region: ", region.Description, " Name: ", region.Name)
						err := bt.GetRegionData(region)

						if err != nil {
							//TODO: return?
							logp.NewLogger(selector).Error(err)
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
			go func() {
				ticker := time.NewTicker(bt.config.WeatherStations.Period)
				defer ticker.Stop()

				for {
					select {
					case <-bt.done:
						goto GotoFinish
					case <-ticker.C:
					}

					err := bt.GetStationsData(stationID)
					if err != nil {
						//TODO: return?
						logp.NewLogger(selector).Error(err)
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
	bt.client.Close()
	close(bt.done)
}
