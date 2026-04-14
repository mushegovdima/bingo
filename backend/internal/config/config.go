package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DBConnectionString string `yaml:"db_connection_string" env:"DB_CONNECTION_STRING"`
	TGBotToken         string `yaml:"tg_bot_token"         env:"TG_BOT_TOKEN"`
	SessionTTLMinutes  int    `yaml:"session_ttl_minutes"  env:"SESSION_TTL_MINUTES" env-default:"43200"` // default: 1 month
	SessionSecret      string `yaml:"session_secret"       env:"SESSION_SECRET"`
	ClientURL          string `yaml:"client_url"           env:"CLIENT_URL"           env-default:"http://localhost:3000"`
	ApiURL             string `yaml:"api_url"              env:"API_URL"              env-default:":8080"`
}

// LoadConfig reads config from path. If path is empty, defaults to "{env}.env".
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
