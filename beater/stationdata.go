/*
 * API documentation: https://dev.netatmo.com/resources/technical/reference/weatherstation/getstationsdata
 */
package beater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	if err := bt.EnsureValidToken(); err != nil {
		return fmt.Errorf("token validation failed before station data request: %v", err)
	}

	body, err := bt.fetchStationData(stationID)
	if err != nil {
		return err
	}

	sdata := StationsData{}
	if err := json.Unmarshal(body, &sdata); err != nil {
		return fmt.Errorf("failed to parse station data response: %v", err)
	}

	transformedData := bt.TransformStationData(sdata)

	ts := time.Now()
	for _, data := range transformedData {
		event := beat.Event{
			Timestamp: ts,
			Fields: mapstr.M{
				"type":    "netatmobeat",
				"netatmo": data,
			},
		}
		bt.client.Publish(event)
	}

	return nil
}

// fetchStationData performs the HTTP request for station data.
// On auth error (401/403), forces one token refresh and retries once.
// Note: if EnsureValidToken already refreshed, the refreshMu re-check inside
// RefreshAccessToken will skip the redundant refresh (no double-rotation risk).
func (bt *Netatmobeat) fetchStationData(stationID string) ([]byte, error) {
	logger := logp.NewLogger(selector)

	body, statusCode, err := bt.doStationDataRequest(stationID)
	if err != nil {
		return nil, err
	}

	if isAuthError(statusCode) {
		logger.Warn("Station data request got auth error (", statusCode, "), forcing token refresh and retrying.")
		if refreshErr := bt.RefreshAccessToken(); refreshErr != nil {
			return nil, fmt.Errorf("token refresh after auth error failed: %v", refreshErr)
		}
		body, statusCode, err = bt.doStationDataRequest(stationID)
		if err != nil {
			return nil, err
		}
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("station data request failed with HTTP %d", statusCode)
	}

	return body, nil
}

func (bt *Netatmobeat) doStationDataRequest(stationID string) ([]byte, int, error) {
	data := url.Values{}
	data.Add("access_token", bt.getAccessToken())
	data.Add("device_id", stationID)
	data.Add("get_favorites", "false")
	data.Add("scope", "read_station")

	u, _ := url.ParseRequestURI(bt.apiBaseURL)
	u.Path = UriPathStation
	urlStr := u.String()

	encoded := data.Encode()

	r, err := http.NewRequestWithContext(bt.ctx, http.MethodPost, urlStr, strings.NewReader(encoded))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create station data request: %v", err)
	}
	r.Header.Add("Content-Type", cookieContentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(encoded)))

	resp, err := bt.httpClient.Do(r)
	if err != nil {
		return nil, 0, fmt.Errorf("station data request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read station data response: %v", err)
	}

	return body, resp.StatusCode, nil
}

func (bt *Netatmobeat) TransformStationData(data StationsData) []mapstr.M {

	modulesMeasurements := []mapstr.M{}

	for d, device := range data.Body.Devices {
		//logp.NewLogger(selector).Debug("Device data: ", device)

		// Dashboard data
		dd := mapstr.M{
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
		measureMainUnit := mapstr.M{
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

			ddm := mapstr.M{
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
			measureModule := mapstr.M{
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
