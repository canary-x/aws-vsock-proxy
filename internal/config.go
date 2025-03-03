package internal

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type config struct {
	ServerPort   uint32        `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
}

func parseConfigFromEnv() (cfg config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
