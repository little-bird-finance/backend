package config

import (
	"github.com/caarlos0/env/v6"
)

type config struct {
	Port        int    `env:"PORT" envDefault:"3000"`
	DatabaseUrl string `env:"DATABASE_URL"`
}

var Config config

func InitConfig() error {
	return env.Parse(&Config)
}
