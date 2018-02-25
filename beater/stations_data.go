/*
 * API documentation: https://dev.netatmo.com/resources/technical/reference/weatherstation/getstationsdata
 */
package beater

import (
	"encoding/json"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	URI_PATH_STATION = "/api/getstationsdata"
)

type StationsData struct {
	Body struct {
		Devices     []Device `json:"devices"`
		User        User     `json:"user"`
		Status      string   `json:"status"`
		Time_exec   float32  `json:"time_exec"`
		Time_server int      `json:"time_server"`
	}
}

type Device struct {
	Device_id         string         `json:"_id"`
	Cipher_id         string         `json:"cipher_id"`
	Last_status_store int            `json:"last_status_store"`
	Modules           []Module       `json:"modules"`
	Place             Place          `json:"place"`
	Station_name      string         `json:"station_name"`
	Type              string         `json:"type"`
	Dashboard_data    Dashboard_data `json:"dashboard_data"`
	Data_type         []string       `json:"data_type"`
	Co2_calibrating   bool           `json:"co_2_calibrating"`
	Date_setup        int            `json:"date_setup"`
	Last_setup        int            `json:"last_setup"`
	Module_name       string         `json:"module_name"`
	Firmware          int            `json:"firmware"`
	Last_upgrade      int            `json:"last_upgrade"`
	Wifi_status       int            `json:"wifi_status"`
}

type Module struct {
	Module_id            string               `json:"_id"`
	Type                 string               `json:"type"`
	Last_message         int                  `json:"last_message"`
	Last_seen            int                  `json:"last_seen"`
	ModuleDashboard_data ModuleDashboard_data `json:"dashboard_data"`
	Data_type            []string             `json:"data_type"`
	Module_name          string               `json:"module_name"`
	Last_setup           int                  `json:"last_setup"`
	Battery_vp           int                  `json:"battery_vp"`
	Battery_percent      int                  `json:"battery_percent"`
	Rf_status            int                  `json:"rf_status"`
	Firmware             int                  `json:"firmware"`
}

type ModuleDashboard_data struct {
	Time_utc      int     `json:"time_utc"`
	Temperature   float32 `json:"Temperature"`
	Temp_trend    string  `json:"temp_trend"`
	Humidity      float32 `json:"Humidity"`
	Date_max_temp int     `json:"date_max_temp"`
	Date_min_temp int     `json:"date_min_temp"`
	Min_temp      float32 `json:"min_temp"`
	Max_temp      float32 `json:"max_temp"`
}

type Place struct {
	Altitude float32   `json:"altitude"`
	City     string    `json:"city"`
	Country  string    `json:"country"`
	Timezone string    `json:"timezone"`
	Location []float32 `json:"location"`
}

type Dashboard_data struct {
	AbsolutePressure float32 `json:"AbsolutePressure"`
	Time_utc         int     `json:"time_utc"`
	Noise            float32 `json:"Noise"`
	Temperature      float32 `json:"Temperature"`
	Temp_trend       string  `json:"temp_trend"`
	Humidity         float32 `json:"Humidity"`
	Pressure         float32 `json:"Pressure"`
	Pressure_trend   string  `json:"pressure_trend"`
	CO2              float32 `json:"CO2"`
	Date_max_temp    int     `json:"date_max_temp"`
	Date_min_temp    int     `json:"date_min_temp"`
	Min_temp         float32 `json:"min_temp"`
	Max_temp         float32 `json:"max_temp"`
}

type User struct {
	Mail           string `json:"mail"`
	Administrative struct {
		Lang           string `json:"lang"`
		Reg_locale     string `json:"reg_locale"`
		Country        string `json:"country"`
		Unit           int    `json:"unit"`
		Windunit       int    `json:"windunit"`
		Pressureunit   int    `json:"pressureunit"`
		Feel_like_algo int    `json:"feel_like_algo"`
	}
}

func (bt *Netatmobeat) GetStationsData() error {
	data := url.Values{}
	//token:=bt.creds.Access_token
	data.Add("access_token", bt.creds.Access_token)
	data.Add("device_id", "70:ee:50:28:90:aa")
	data.Add("get_favorites", "false")
	data.Add("scope", "read_station")

	u, _ := url.ParseRequestURI(netatmoApiUrl)
	u.Path = URI_PATH_STATION
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
	//fmt.Printf(string(body))

	sdata := &StationsData{}
	err = json.Unmarshal([]byte(body), &sdata)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("Response data: %s\n", sdata)

	return nil
}
