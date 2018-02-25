// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period        time.Duration `config:"period"`
	Client_id     string        `config:"client_id"`
	Client_secret string        `config:"client_secret"`
	Username      string        `config:"username"`
	Password      string        `config:"password"`
}

var DefaultConfig = Config{
	Period:   1 * time.Second,
	Username: "",
	Password: "",
}
