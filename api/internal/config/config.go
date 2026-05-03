package config

import (
	"errors"
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
	MetricsURL         string `yaml:"metrics_url"          env:"METRICS_URL"          env-default:":9091"`
	NotifierAddr       string `yaml:"notifier_addr"        env:"NOTIFIER_ADDR"        env-default:""`

	// ThrottleLimit is the maximum number of concurrent in-flight HTTP requests.
	ThrottleLimit int `yaml:"throttle_limit" env:"THROTTLE_LIMIT" env-default:"30"`

	// NotificationJobBatch is the number of outbox jobs each worker tick claims at once.
	NotificationJobBatch int `yaml:"notification_job_batch" env:"NOTIFICATION_JOB_BATCH" env-default:"4"`
	// NotificationClaimStaleSeconds is the lease lifetime for a claimed job: a job stuck in
	// 'running' with locked_at older than this is treated as crashed and re-claimable.
	NotificationClaimStaleSeconds int `yaml:"notification_claim_stale_seconds" env:"NOTIFICATION_CLAIM_STALE_SECONDS" env-default:"300"`
}

// LoadConfig reads config from path. If path is empty, defaults to "{env}.env".
// The result is validated before being returned: callers can rely on required
// fields being non-empty and dimensioned reasonably.
func LoadConfig(env, path string) (*Config, error) {
	if path == "" {
		path = fmt.Sprintf("%s.env", env)
	}
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// minSessionSecretLen is the minimum byte length required for the session HMAC key.
// gorilla/securecookie warns about anything shorter; 32 bytes matches the recommended
// HMAC-SHA256 key size.
const minSessionSecretLen = 32

// Validate checks that all required fields are populated and within sane bounds.
// It catches misconfiguration at boot rather than letting the service start and
// fail on the first request.
func (c *Config) Validate() error {
	var errs []error
	if c.DBConnectionString == "" {
		errs = append(errs, fmt.Errorf("DB_CONNECTION_STRING is required"))
	}
	if c.TGBotToken == "" {
		errs = append(errs, fmt.Errorf("TG_BOT_TOKEN is required"))
	}
	if c.SessionSecret == "" {
		errs = append(errs, fmt.Errorf("SESSION_SECRET is required"))
	} else if len(c.SessionSecret) < minSessionSecretLen {
		errs = append(errs, fmt.Errorf("SESSION_SECRET must be at least %d bytes", minSessionSecretLen))
	}
	if c.SessionTTLMinutes <= 0 {
		errs = append(errs, fmt.Errorf("SESSION_TTL_MINUTES must be > 0"))
	}
	if c.NotificationJobBatch <= 0 {
		errs = append(errs, fmt.Errorf("NOTIFICATION_JOB_BATCH must be > 0"))
	}
	if c.NotificationClaimStaleSeconds <= 0 {
		errs = append(errs, fmt.Errorf("NOTIFICATION_CLAIM_STALE_SECONDS must be > 0"))
	}
	return errors.Join(errs...)
}
