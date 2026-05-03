package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DBConnectionString      string `env:"DB_CONNECTION_STRING"`
	TGBotToken              string `env:"TG_BOT_TOKEN"`
	GRPCPort                string `env:"GRPC_PORT"                env-default:":50051"`
	WorkerIntervalSec       int    `env:"WORKER_INTERVAL_SEC"      env-default:"30"`
	WorkerBatchSize         int    `env:"WORKER_BATCH_SIZE"           env-default:"50"`
	WorkerReservationTTLSec int    `env:"WORKER_RESERVATION_TTL_SEC"  env-default:"30"`
}

func LoadConfig(env, path string) (*Config, error) {
	if path == "" {
		path = fmt.Sprintf("%s.env", env)
	}
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
