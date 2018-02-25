package beater

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (bt *Netatmobeat) GetAccessToken() error {

	data := url.Values{}
	data.Add("grant_type", "password")
	data.Add("client_id", bt.config.Client_id)
	data.Add("client_secret", bt.config.Client_secret)
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

	//rt := ResponseOauth2Token{}
	err = json.Unmarshal([]byte(body), &bt.creds)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Access_token: %v", bt.creds.Access_token)
	fmt.Printf("Refresh_token: %v", bt.creds.Refresh_token)

	//fmt.Printf(string(bodyBytes))

	return nil
}

//Endpoint: https://api.netatmo.com/oauth2/token
func (bt *Netatmobeat) RefreshAccessToken() error {

	data := url.Values{}
	data.Add("grant_type", "refresh_token")
	data.Add("client_id", bt.config.Client_id)
	data.Add("client_secret", bt.config.Client_secret)
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

	//rt := ResponseOauth2Token{}
	err = json.Unmarshal([]byte(body), &bt.creds)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Access_token: %v", bt.creds.Access_token)
	fmt.Printf("Refresh_token: %v", bt.creds.Refresh_token)

	//fmt.Printf(string(bodyBytes))

	return nil
}
