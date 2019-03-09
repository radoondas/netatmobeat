// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	ClientId        string          `config:"client_id"`
	ClientSecret    string          `config:"client_secret"`
	Username        string          `config:"username"`
	Password        string          `config:"password"`
	WeatherStations WeatherStations `config:"weather_stations"`
	PublicWeather   PublicWeather   `config:"public_weather"`
}

type WeatherStations struct {
	Enabled bool          `config:"enabled"`
	Ids     []string      `config:"ids"`
	Period  time.Duration `config:"period"`
}

type PublicWeather struct {
	Enabled bool          `config:"enabled"`
	Regions []Region      `config:"regions"`
	Period  time.Duration `config:"period"`
}

type Region struct {
	Enabled     bool    `config:"enabled"`
	Name        string  `config:"name"`
	Description string  `config:"description"`
	LatNe       float64 `config:"lat_ne"`
	LonNe       float64 `config:"lon_ne"`
	LatSw       float64 `config:"lat_sw"`
	LonSw       float64 `config:"lon_sw"`
}

var DefaultConfig = Config{
	//Period:   1 * time.Second,
	Username: "",
	Password: "",
}
