package beater

import (
	"fmt"
	"log"
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
)

type Netatmobeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
	creds  ResponseOauth2Token
}

//{"access_token":"599163c4f5459521648b4586|d56b25668a4fcb4c956c12073942fdb5","refresh_token":"599163c4f5459521648b4586|938350d65da251ec562ef8f52ee6f294","scope":["read_station"],"expires_in":10800,"expire_in":10800}
type ResponseOauth2Token struct {
	Access_token  string   `json:"access_token"`
	Refresh_token string   `json:"refresh_token"`
	Scope         []string `json:"scope"`
	Expires_in    int      `json:"expires_in"`
	Expire_in     int      `json:"expire_in"`
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
	logp.Info("netatmobeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(bt.config.Period)
	counter := 1

	err = bt.GetAccessToken()
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = bt.GetStationsData()
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = bt.GetPublicData()
	if err != nil {
		log.Fatal(err)
		return err
	}

	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    b.Info.Name,
				"counter": counter,
			},
		}
		bt.client.Publish(event)
		//bt.client.PublishEvents(event)
		logp.Info("Event sent")
		counter++
	}
}

func (bt *Netatmobeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
