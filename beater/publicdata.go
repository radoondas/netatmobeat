/*
 * API documentation: https://dev.netatmo.com/resources/technical/reference/weatherapi/getpublicdata
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

	"github.com/mitchellh/hashstructure"
	"github.com/radoondas/netatmobeat/config"
)

const (
	UriPathPublicData = "/api/getpublicdata"
)

type PublicData struct {
	Stations   []PublicStation `json:"body"`
	Status     string          `json:"status"`
	TimeExec   float32         `json:"time_exec"`
	TimeServer int             `json:"time_server"`
}

type PublicStation struct {
	StationId string             `json:"_id"`
	Place     Place              `json:"place"`
	Mark      int                `json:"mark"`
	Measures  map[string]Measure `json:"measures"`
	Modules   []string           `json:"modules"`
	//ModuleTypes map[string]string `json:"module_types"` // This data is not required, as they mean nothing valuable. Decision can be changed later.
}

type Measure struct {
	Res            map[string][]float32 `json:"res"`
	Mes_type       []string             `json:"type"`
	Rain_60min     float32              `json:"rain_60min"`    //"rain_60min": 0
	Rain_24h       float32              `json:"rain_24h"`      //"rain_24h": 1.515
	Rain_live      float32              `json:"rain_live"`     //"rain_live": 0
	Rain_timestamp int                  `json:"rain_timeutc"`  //"rain_timeutc": 1504796338
	Wind_strength  float32              `json:"wind_strength"` //"wind_strength": 18
	Wind_angle     int                  `json:"wind_angle"`    //"wind_angle": 335
	Gust_strength  int                  `json:"gust_strength"` //"gust_strength": 33
	Gust_angle     int                  `json:"gust_angle"`    //"gust_angle": 341
	Wind_timestamp int                  `json:"wind_timeutc"`  //"wind_timeutc": 1504796344
}

// * access_token yes
// * lat_ne yes 15
// latitude of the north east corner of the requested area. -85 <= lat_ne <= 85 and lat_ne>lat_sw
// * lon_ne yes 20
// Longitude of the north east corner of the requested area. -180 <= lon_ne <= 180 and lon_ne>lon_sw
// * lat_sw yes -15
// latitude of the south west corner of the requested area. -85 <= lat_sw <= 85
// * lon_sw yes -20
// Longitude of the south west corner of the requested area. -180 <= lon_sw <= 180
// * required_data no rain, humidity
// To filter stations based on relevant measurements you want (e.g. rain will only return stations with rain gauges). Default is no filter. You can find all measurements available on the Thermostat page.
// * filter no true
// True to exclude station with abnormal temperature measures. Default is false.
func (bt *Netatmobeat) GetRegionData(region config.Region) error {
	if err := bt.EnsureValidToken(); err != nil {
		return fmt.Errorf("token validation failed before public data request: %v", err)
	}

	body, err := bt.fetchPublicData(region)
	if err != nil {
		return err
	}

	sdata := PublicData{}
	if err := json.Unmarshal(body, &sdata); err != nil {
		return fmt.Errorf("failed to parse public data response: %v", err)
	}

	transformedData := bt.TransformPublicData(sdata, region.Name, region.Description)

	ts := time.Now()
	for _, data := range transformedData {
		hash, err := hashstructure.Hash(data, nil)
		var myid string
		if err != nil {
			logp.NewLogger(selector).Warn("Hash calculation failed, using timestamp fallback: ", err)
			myid = strconv.FormatInt(ts.UnixNano(), 10)
		} else {
			myid = strconv.FormatUint(hash, 10)
		}

		event := beat.Event{
			Meta: mapstr.M{
				"id": myid,
			},
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

// fetchPublicData performs the HTTP request for public data.
// On auth error (401/403), forces one token refresh and retries once.
// Note: if EnsureValidToken already refreshed, the refreshMu re-check inside
// RefreshAccessToken will skip the redundant refresh (no double-rotation risk).
func (bt *Netatmobeat) fetchPublicData(region config.Region) ([]byte, error) {
	logger := logp.NewLogger(selector)
	logger.Debug("Shape name: ", region.Name, ", Description: ", region.Description)

	body, statusCode, err := bt.doPublicDataRequest(region)
	if err != nil {
		return nil, err
	}

	if isAuthError(statusCode) {
		logger.Warn("Public data request got auth error (", statusCode, "), forcing token refresh and retrying.")
		if refreshErr := bt.RefreshAccessToken(); refreshErr != nil {
			return nil, fmt.Errorf("token refresh after auth error failed: %v", refreshErr)
		}
		body, statusCode, err = bt.doPublicDataRequest(region)
		if err != nil {
			return nil, err
		}
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("public data request failed with HTTP %d", statusCode)
	}

	return body, nil
}

func (bt *Netatmobeat) doPublicDataRequest(region config.Region) ([]byte, int, error) {
	data := url.Values{}
	data.Add("access_token", bt.getAccessToken())
	data.Add("lat_ne", strconv.FormatFloat(region.LatNe, 'f', -1, 32))
	data.Add("lon_ne", strconv.FormatFloat(region.LonNe, 'f', -1, 32))
	data.Add("lat_sw", strconv.FormatFloat(region.LatSw, 'f', -1, 32))
	data.Add("lon_sw", strconv.FormatFloat(region.LonSw, 'f', -1, 32))

	u, _ := url.ParseRequestURI(bt.apiBaseURL)
	u.Path = UriPathPublicData
	urlStr := u.String()

	encoded := data.Encode()

	r, err := http.NewRequestWithContext(bt.ctx, http.MethodPost, urlStr, strings.NewReader(encoded))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create public data request: %v", err)
	}
	r.Header.Add("Content-Type", cookieContentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(encoded)))
	r.Header.Add("Cache-Control", "no-cache, must-revalidate")

	resp, err := bt.httpClient.Do(r)
	if err != nil {
		return nil, 0, fmt.Errorf("public data request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read public data response: %v", err)
	}

	return body, resp.StatusCode, nil
}

func (bt *Netatmobeat) TransformPublicData(data PublicData, regionName string, regionDescription string) []mapstr.M {

	var stations []mapstr.M

	//index instead _
	for _, station := range data.Stations {
		pubdata := mapstr.M{}
		measures := mapstr.M{}

		s := mapstr.M{
			"station_id":       station.StationId,
			"place":            station.Place,
			"mark":             station.Mark,
			"regionName":       regionName,
			"regionDescripion": regionDescription,
			"source_type":      "publicdata",
		}

		for moduleId, ms := range station.Measures {
			//fmt.Printf("Measure module: %v\n", moduleId)
			if ms.Mes_type != nil {
				for k, v := range ms.Res {
					//fmt.Println("k:", k, "v:", v)
					for i, mes := range ms.Mes_type {
						switch t := mes; t {
						case "temperature":
							//fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							temperature := mapstr.M{
								"timestamp": dt * 1000,
								"value":     v[i],
								"moduleId":  moduleId,
							}
							measures.Put(t, temperature)
							//fmt.Println(temp)
						case "humidity":
							//fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							humidity := mapstr.M{
								//"mesTimestamp": time.Unix(int64(dt), 0).String(),
								"timestamp": dt * 1000,
								"value":     v[i],
								"moduleId":  moduleId,
							}
							measures.Put(t, humidity)
							//fmt.Println(hum)
						case "pressure":
							//fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							pressure := mapstr.M{
								//"mesTimestamp": time.Unix(int64(dt), 0).String(),
								"timestamp": dt * 1000,
								"value":     v[i],
								"moduleId":  moduleId,
							}
							measures.Put(t, pressure)
							//fmt.Println(press)
						}
					}
				}
			} else {
				//fmt.Printf("ms.Mes_type.len (nil): %v\n", len(ms.Mes_type))
				if ms.Wind_timestamp != 0 {
					wind := mapstr.M{
						"moduleId":     moduleId,
						"windAngle":    ms.Wind_angle,
						"windStrength": ms.Wind_strength,
						"gustStrength": ms.Gust_strength,
						"gustAngle":    ms.Gust_angle,
						"timestamp":    ms.Wind_timestamp * 1000,
					}
					measures.Put("wind", wind)
				} else if ms.Rain_timestamp != 0 {
					rain := mapstr.M{
						"moduleId":   moduleId,
						"timestamp":  ms.Rain_timestamp * 1000,
						"rain_24h":   ms.Rain_24h,
						"rain_60min": ms.Rain_60min,
						"rain_live":  ms.Rain_live,
					}
					measures.Put("rain", rain)
				}
			}
			pubdata.Put("measures", measures)
			s.Put("publicdata", pubdata)

			s.Put("place", station.Place)
		}
		stations = append(stations, s)

		//logp.NewLogger(selector).Debug("Public data: ", s)
	}

	return stations
}
