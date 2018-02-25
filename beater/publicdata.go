/*
 * API documentation: https://dev.netatmo.com/resources/technical/reference/weatherapi/getpublicdata
 */

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
	"time"

	"github.com/elastic/beats/libbeat/common"
)

const (
	URI_PATH_PUBLIC_DATA = "/api/getpublicdata"
)

type PublicData struct {
	Stations    []PublicStation `json:"body"`
	Status      string          `json:"status"`
	Time_exec   float32         `json:"time_exec"`
	Time_server int             `json:"time_server"`
}

type PublicStation struct {
	Station_id string             `json:"_id"`
	Place      Place              `json:"place"`
	Mark       int                `json:"mark"`
	Measures   map[string]Measure `json:"measures"`
	Modules    []string           `json:"modules"`
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

type Temperature struct {
	Mes_timestamp int
	Value         float32
	Module_id     string
}

type Humidity struct {
	Mes_timestamp int
	Value         float32
	Module_id     string
}

type Rain struct {
	Rain_60min    float32 //"rain_60min": 0
	Rain_24h      float32 //"rain_24h": 1.515
	Rain_live     float32 //"rain_live": 0
	Mes_timestamp int     //"rain_timeutc": 1504796338
	Module_id     string
}

type Wind struct {
	Wind_strength float32 //"wind_strength": 18
	Wind_angle    int     //"wind_angle": 335
	Gust_strength int     //"gust_strength": 33
	Gust_angle    int     //"gust_angle": 341
	Mes_timestamp int     //"wind_timeutc": 1504796344
	Module_id     string
}

type Pressure struct {
	Mes_timestamp int
	Value         float32
	Module_id     string
}

func (bt *Netatmobeat) GetPublicData() error {

	//Slovakia lat_ne=49.659740&lon_ne=22.552247&lat_sw=47.648413&lon_sw=16.835147

	// * access_token yes
	// * lat_ne yes 15
	//latitude of the north east corner of the requested area. -85 <= lat_ne <= 85 and lat_ne>lat_sw
	// * lon_ne yes 20
	//Longitude of the north east corner of the requested area. -180 <= lon_ne <= 180 and lon_ne>lon_sw
	// * lat_sw yes -15
	//latitude of the south west corner of the requested area. -85 <= lat_sw <= 85
	// * lon_sw yes -20
	//Longitude of the south west corner of the requested area. -180 <= lon_sw <= 180
	// * required_data no rain, humidity
	//To filter stations based on relevant measurements you want (e.g. rain will only return stations with rain gauges). Default is no filter. You can find all measurements available on the Thermostat page.
	// * filter no true
	//True to exclude station with abnormal temperature measures. Default is false.

	//PP: ne: 49.110701, 20.394950
	//PP: sw: 49.013293, 20.204749

	data := url.Values{}
	//token:=bt.creds.Access_token
	data.Add("access_token", bt.creds.Access_token)
	//SK
	//data.Add("lat_ne", "49.659740")
	//data.Add("lon_ne", "22.552247")
	//data.Add("lat_sw", "47.648413")
	//data.Add("lon_sw", "16.835147")
	//PP
	data.Add("lat_ne", "49.110701")
	data.Add("lon_ne", "20.394950")
	data.Add("lat_sw", "49.013293")
	data.Add("lon_sw", "20.204749")

	u, _ := url.ParseRequestURI(netatmoApiUrl)
	u.Path = URI_PATH_PUBLIC_DATA
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
	fmt.Printf(string(body))

	sdata := &PublicData{}
	err = json.Unmarshal([]byte(body), &sdata)
	if err != nil {
		fmt.Printf("error: %v", err)
		panic(err)
	}
	//fmt.Printf("Response data: %s\n", sdata)

	bt.Transform(sdata)

	return nil

}

func (bt *Netatmobeat) Transform(data *PublicData) []common.MapStr {

	stations := []common.MapStr{}
	pubdata := common.MapStr {}
	measures := common.MapStr{}

	for index, station := range data.Stations {
		s := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			//"type":       b.Name,
			"type":       "Netatmobeat",
			"station_id": station.Station_id,
			"place":      station.Place, //TODO: need to swap lot,lan
			"mark":       station.Mark,
			"source_type": "publicdata",
		}

		for module_id, ms := range station.Measures {
			fmt.Printf("Measure module: %v\n", module_id)
			if ms.Mes_type != nil {
				for k, v := range ms.Res {
					fmt.Println("k:", k, "v:", v)
					for i, mes := range ms.Mes_type {
						switch t := mes; t {
						case "temperature":
							fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							temp := Temperature{
								Mes_timestamp: dt,
								Value:         v[i],
								Module_id:     module_id,
							}
							measures.Put(t, temp)
							fmt.Println(temp)
						case "humidity":
							fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							hum := Humidity{
								Mes_timestamp: dt,
								Value:         v[i],
								Module_id:     module_id,
							}
							measures.Put(t, hum)
							fmt.Println(hum)
						case "pressure":
							fmt.Printf("type: %v, mes_date: %v, val: %v\n", mes, k, v[i])
							dt, _ := strconv.Atoi(k)
							press := Pressure{
								Mes_timestamp: dt,
								Value:         v[i],
								Module_id:     module_id,
							}
							measures.Put(t, press)
							fmt.Println(press)
						}
					}
				}
			} else {
				fmt.Printf("ms.Mes_type.len (nil): %v\n", len(ms.Mes_type))
				if ms.Wind_timestamp != 0 {
					wind := Wind{
						Module_id:     module_id,
						Wind_angle:    ms.Wind_angle,
						Wind_strength: ms.Wind_strength,
						Gust_strength: ms.Gust_strength,
						Gust_angle:    ms.Gust_angle,
						Mes_timestamp: ms.Wind_timestamp,
					}
					measures.Put("wind", wind)
				} else if ms.Rain_timestamp != 0 {
					rain := Rain{
						Module_id:     module_id,
						Mes_timestamp: ms.Rain_timestamp,
						Rain_24h:      ms.Rain_24h,
						Rain_60min:    ms.Rain_60min,
						Rain_live:     ms.Rain_live,
					}
					measures.Put("rain", rain)
				}
			}
			pubdata.Put("measures", measures)
			s.Put("publicdata", pubdata)

			s.Put("place", station.Place)
		}
		stations = append(stations, s)
		fmt.Printf("Index: %v\n", index)
		fmt.Printf("Station id: %v\n", station.Station_id)
	}

	return nil
}
