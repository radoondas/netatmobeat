// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"fmt"
	"time"
)

type Config struct {
	ClientId     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`
	AccessToken  string `config:"access_token"`
	RefreshToken string `config:"refresh_token"`
	TokenFile    string `config:"token_file"`
	// Deprecated: password grant is no longer supported by Netatmo OAuth.
	Username string `config:"username"`
	// Deprecated: password grant is no longer supported by Netatmo OAuth.
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

// Validate checks the configuration for common errors.
func (r Region) Validate() error {
	if r.LatNe <= r.LatSw {
		return fmt.Errorf("region %q: lat_ne (%f) must be greater than lat_sw (%f)", r.Name, r.LatNe, r.LatSw)
	}
	if r.LonNe <= r.LonSw {
		return fmt.Errorf("region %q: lon_ne (%f) must be greater than lon_sw (%f)", r.Name, r.LonNe, r.LonSw)
	}
	return nil
}

var DefaultConfig = Config{
	TokenFile: "netatmobeat-tokens.json",
}
