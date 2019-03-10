/*
 * API documentation: https://dev.netatmo.com/resources/technical/reference/weatherstation/getstationsdata
 */
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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	UriPathStation = "/api/getstationsdata"
)

type StationsData struct {
	Body struct {
		Devices     []Device `json:"devices"`
		User        User     `json:"user"`
		Status      string   `json:"status"`
		Time_exec   float32  `json:"time_exec"`
		Time_server int64    `json:"time_server"`
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
	Last_message         int64                `json:"last_message"`
	Last_seen            int64                `json:"last_seen"`
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
	Date_max_temp int64   `json:"date_max_temp"`
	Date_min_temp int64   `json:"date_min_temp"`
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
	Time_utc         int64   `json:"time_utc"`
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

func (bt *Netatmobeat) GetStationsData(stationID string) error {
	data := url.Values{}
	//token:=bt.creds.Access_token
	data.Add("access_token", bt.creds.Access_token)
	data.Add("device_id", stationID)
	data.Add("get_favorites", "false")
	data.Add("scope", "read_station")

	u, _ := url.ParseRequestURI(netatmoApiUrl)
	u.Path = UriPathStation
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

	sdata := StationsData{}
	err = json.Unmarshal([]byte(body), &sdata)
	if err != nil {
		panic(err)
	}

	transformedData := bt.TransformStationData(sdata)

	//logp.NewLogger(selector).Debug("Station data: ", transformedData)

	for _, data := range transformedData {
		//logp.NewLogger(selector).Debug("Data: ", data)

		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    "netatmobeat",
				"netatmo": data,
			},
		}
		logp.NewLogger(selector).Debug("Event: ", event)
		bt.client.Publish(event)
		//logp.NewLogger(selector).Info("Event sent")
	}

	return nil
}

func (bt *Netatmobeat) TransformStationData(data StationsData) []common.MapStr {

	modulesMeasurements := []common.MapStr{}

	for d, device := range data.Body.Devices {
		//logp.NewLogger(selector).Debug("Device data: ", device)

		// Dashboard data
		dd := common.MapStr{
			"time_utc":         device.Dashboard_data.Time_utc * 1000,
			"temperature":      device.Dashboard_data.Temperature,
			"co2":              device.Dashboard_data.CO2,
			"humidity":         device.Dashboard_data.Humidity,
			"noise":            device.Dashboard_data.Noise,
			"pressure":         device.Dashboard_data.Pressure,
			"absolutePressure": device.Dashboard_data.AbsolutePressure,
			"min_temp":         device.Dashboard_data.Min_temp,
			"max_temp":         device.Dashboard_data.Max_temp,
			"date_min_temp":    device.Dashboard_data.Date_min_temp * 1000,
			"date_max_temp":    device.Dashboard_data.Date_max_temp * 1000,
			"temp_trend":       device.Dashboard_data.Temp_trend,
			"pressure_trend":   device.Dashboard_data.Pressure_trend,
		}

		// measurement
		measureMainUnit := common.MapStr{
			"station_id":   device.Device_id,
			"place":        device.Place,
			"station_type": device.Type,
			"module_name":  device.Module_name,
			"station_name": device.Station_name,
			"source_type":  "stationdata",
			"stationdata":  dd,
		}
		logp.NewLogger(selector).Debug("Main unit: ", measureMainUnit)

		modulesMeasurements = append(modulesMeasurements, measureMainUnit)

		for _, module := range data.Body.Devices[d].Modules {
			//logp.NewLogger(selector).Debug("Module data: ", module)

			ddm := common.MapStr{
				"time_utc":      module.ModuleDashboard_data.Time_utc * 1000,
				"temperature":   module.ModuleDashboard_data.Temperature,
				"humidity":      module.ModuleDashboard_data.Humidity,
				"min_temp":      module.ModuleDashboard_data.Min_temp,
				"max_temp":      module.ModuleDashboard_data.Max_temp,
				"date_min_temp": module.ModuleDashboard_data.Date_min_temp * 1000,
				"date_max_temp": module.ModuleDashboard_data.Date_max_temp * 1000,
				"temp_trend":    module.ModuleDashboard_data.Temp_trend,
			}

			// measurement
			measureModule := common.MapStr{
				"station_id":      device.Device_id,
				"module_id":       module.Module_id,
				"place":           device.Place,
				"station_type":    module.Type,
				"module_name":     module.Module_name,
				"station_name":    device.Station_name,
				"last_message":    module.Last_message,
				"last_seen":       module.Last_seen,
				"rf_status":       module.Rf_status,
				"battery_vp":      module.Battery_vp,
				"battery_percent": module.Battery_percent,
				"source_type":     "stationdata",
				"stationdata":     ddm,
			}

			modulesMeasurements = append(modulesMeasurements, measureModule)
			logp.NewLogger(selector).Debug("Module unit ", module.Module_name, ": ", measureModule)
		}
	}
	return modulesMeasurements
}
