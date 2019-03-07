package beater

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

func (bt *Netatmobeat) GetAccessToken() error {

	logp.NewLogger(selector).Debug("Authenticating.")

	data := url.Values{}
	data.Add("grant_type", "password")
	data.Add("client_id", bt.config.ClientId)
	data.Add("client_secret", bt.config.ClientSecret)
	data.Add("username", bt.config.Username)
	data.Add("password", bt.config.Password)
	data.Add("scope", "read_station")

	u, _ := url.ParseRequestURI(netatmoApiUrl)
	u.Path = authPath
	urlStr := u.String()

	client := &http.Client{}

	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode())) // <-- URL-encoded payload
	r.Header.Add("Content-Type", cookieContentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(r)

	if err != nil {
		log.Fatal(err)
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &bt.creds)
	if err != nil {
		panic(err)
	}

	// set last auth time
	bt.creds.LastAuthTime = time.Now().UTC().Unix()

	logp.NewLogger(selector).Debug("Access_token: ", bt.creds.Access_token)
	logp.NewLogger(selector).Debug("Refresh_token: ", bt.creds.Refresh_token)
	logp.NewLogger(selector).Debug("Expires in: ", bt.creds.Expire_in)

	return nil
}

//Endpoint: https://api.netatmo.com/oauth2/token
func (bt *Netatmobeat) RefreshAccessToken() error {

	logp.NewLogger(selector).Debug("Refreshing Token.")

	data := url.Values{}
	data.Add("grant_type", "refresh_token")
	data.Add("client_id", bt.config.ClientId)
	data.Add("client_secret", bt.config.ClientSecret)
	data.Add("refresh_token", bt.creds.Refresh_token)

	u, _ := url.ParseRequestURI(netatmoApiUrl)
	u.Path = authPath
	urlStr := u.String()

	client := &http.Client{}

	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode())) // <-- URL-encoded payload
	r.Header.Add("Content-Type", cookieContentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(r)

	if err != nil {
		log.Fatal(err)
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &bt.creds)
	if err != nil {
		panic(err)
	}

	// set last auth time
	bt.creds.LastAuthTime = time.Now().UTC().Unix()

	logp.NewLogger(selector).Debug("Access_token: ", bt.creds.Access_token)
	logp.NewLogger(selector).Debug("Refresh_token: ", bt.creds.Refresh_token)
	logp.NewLogger(selector).Debug("Expires in: ", bt.creds.Expire_in)

	return nil
}
