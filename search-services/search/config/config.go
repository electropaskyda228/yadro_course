package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel      string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address       string        `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:"localhost:80"`
	WordsAddress  string        `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:81"`
	DBAddress     string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	TtlInit       time.Duration `yaml:"ttl_init" env:"INDEX_TTL" env-default:"20s"`
	BrokerAddress string        `yaml:"broker_address" env:"BROKER_ADDRESS" env-default:"nats://localhost:4222"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
