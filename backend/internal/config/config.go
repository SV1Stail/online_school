package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddress       = ":8080"
	defaultSessionTTL        = 24 * time.Hour
	defaultRequestTimeout    = 10 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultMaxBodyBytes      = 1 << 20
	defaultRateLimitRPS      = 10
	defaultRateLimitBurst    = 20
	defaultMaxRateClients    = 10_000
	defaultMaxInFlight       = 100
	defaultMaxLoginInFlight  = 4
	defaultLoginAttempts     = 5
	defaultLoginWindow       = 15 * time.Minute
)

type Config struct {
	Environment       string
	HTTPAddress       string
	DatabaseURL       string
	RedisAddress      string
	RedisPassword     string
	RedisDB           int
	LogLevel          string
	LogPretty         bool
	CookieSecure      bool
	SessionTTL        time.Duration
	RequestTimeout    time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxBodyBytes      int64
	RateLimitRPS      float64
	RateLimitBurst    int
	MaxRateClients    int
	MaxInFlight       int
	MaxLoginInFlight  int
	LoginAttempts     int64
	LoginWindow       time.Duration
	TrustedProxyCIDRs []string
	AllowedOrigins    []string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment:       envString("APP_ENV", "development"),
		HTTPAddress:       envString("HTTP_ADDRESS", defaultHTTPAddress),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddress:      envString("REDIS_ADDRESS", "redis:6379"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		LogLevel:          envString("LOG_LEVEL", "info"),
		SessionTTL:        defaultSessionTTL,
		RequestTimeout:    defaultRequestTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxBodyBytes:      defaultMaxBodyBytes,
		RateLimitRPS:      defaultRateLimitRPS,
		RateLimitBurst:    defaultRateLimitBurst,
		MaxRateClients:    defaultMaxRateClients,
		MaxInFlight:       defaultMaxInFlight,
		MaxLoginInFlight:  defaultMaxLoginInFlight,
		LoginAttempts:     defaultLoginAttempts,
		LoginWindow:       defaultLoginWindow,
		TrustedProxyCIDRs: splitCSV(os.Getenv("TRUSTED_PROXY_CIDRS")),
		AllowedOrigins:    splitCSV(os.Getenv("ALLOWED_ORIGINS")),
	}

	var err error
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.RedisDB, err = envInt("REDIS_DB", 0); err != nil {
		return nil, err
	}
	if cfg.LogPretty, err = envBool("LOG_PRETTY", cfg.Environment == "development"); err != nil {
		return nil, err
	}
	if cfg.CookieSecure, err = envBool("COOKIE_SECURE", cfg.Environment != "development"); err != nil {
		return nil, err
	}
	if cfg.SessionTTL, err = envDuration("SESSION_TTL", defaultSessionTTL); err != nil {
		return nil, err
	}
	if cfg.RequestTimeout, err = envDuration("REQUEST_TIMEOUT", defaultRequestTimeout); err != nil {
		return nil, err
	}
	if cfg.ShutdownTimeout, err = envDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout); err != nil {
		return nil, err
	}
	if cfg.MaxBodyBytes, err = envInt64("MAX_BODY_BYTES", defaultMaxBodyBytes); err != nil {
		return nil, err
	}
	if cfg.RateLimitRPS, err = envFloat("RATE_LIMIT_RPS", defaultRateLimitRPS); err != nil {
		return nil, err
	}
	if cfg.RateLimitBurst, err = envInt("RATE_LIMIT_BURST", defaultRateLimitBurst); err != nil {
		return nil, err
	}
	if cfg.MaxRateClients, err = envInt("MAX_RATE_CLIENTS", defaultMaxRateClients); err != nil {
		return nil, err
	}
	if cfg.MaxInFlight, err = envInt("MAX_IN_FLIGHT", defaultMaxInFlight); err != nil {
		return nil, err
	}
	if cfg.MaxLoginInFlight, err = envInt("MAX_LOGIN_IN_FLIGHT", defaultMaxLoginInFlight); err != nil {
		return nil, err
	}
	if cfg.LoginAttempts, err = envInt64("LOGIN_ATTEMPTS", defaultLoginAttempts); err != nil {
		return nil, err
	}
	if cfg.LoginWindow, err = envDuration("LOGIN_WINDOW", defaultLoginWindow); err != nil {
		return nil, err
	}

	if cfg.RateLimitRPS <= 0 || cfg.RateLimitBurst <= 0 || cfg.MaxRateClients <= 0 ||
		cfg.MaxInFlight <= 0 || cfg.MaxLoginInFlight <= 0 {
		return nil, errors.New("request protection limits must be positive")
	}

	return cfg, nil
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}

	return fallback
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func envInt64(name string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func envFloat(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func envBool(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
