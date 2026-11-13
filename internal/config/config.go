package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

type EnvLoader struct {
	lookup func(key string) (value string, ok bool)
}

var _ Loader = (*EnvLoader)(nil)

func (l *EnvLoader) get(key string) (string, bool) {
	value, ok := l.lookup(key)

	value = strings.TrimSpace(value)

	if !ok || value == "" {
		return "", false
	}

	return value, true
}

func NewEnvLoader(lookup func(key string) (value string, ok bool)) *EnvLoader {
	if lookup == nil {
		lookup = os.LookupEnv
	}

	return &EnvLoader{
		lookup: lookup,
	}
}

func (l *EnvLoader) Load(ctx context.Context) (Config, error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}

	var cfg Config
	var configErrors []error

	httpAddr, ok := l.get("LF_HTTP_ADDR")

	if !ok {
		cfg.HTTPAddr = ":8080"
	} else {
		cfg.HTTPAddr = httpAddr
	}

	_, _, err := net.SplitHostPort(httpAddr)

	if err != nil {
		configErrors = append(configErrors, fmt.Errorf("LF_HTTP_ADDR: config: %w", ErrInvalid))
	}

	adminAddr, ok := l.get("LF_ADMIN_ADDR")

	if !ok {
		cfg.AdminAddr = "127.0.0.1:6060"
	} else {
		cfg.AdminAddr = adminAddr
	}

	_, _, err = net.SplitHostPort(adminAddr)

	if err != nil {
		configErrors = append(configErrors, fmt.Errorf("LF_ADMIN_ADDR: config: %w", ErrInvalid))
	}

	tcpAddr, ok := l.get("LF_TCP_ADDR")

	if !ok {
		cfg.TCPAddr = ":7070"
	} else {
		cfg.TCPAddr = tcpAddr
	}

	_, _, err = net.SplitHostPort(tcpAddr)

	if err != nil {
		configErrors = append(configErrors, fmt.Errorf("LF_TCP_ADDR: config: %w", ErrInvalid))
	}

	pgDSN, ok := l.get("LF_PG_DSN")

	if !ok || len(pgDSN) == 0 {
		configErrors = append(configErrors, fmt.Errorf("LF_PG_DSN: config: %w", ErrMissing))
	} else {
		cfg.PGDSN = pgDSN
	}

	mongoURI, ok := l.get("LF_MONGO_URI")

	if !ok || len(pgDSN) == 0 {
		configErrors = append(configErrors, fmt.Errorf("LF_MONGO_URI: config: %w", ErrMissing))
	} else {
		cfg.MongoURI = mongoURI
	}

	jwtSecret, ok := l.get("LF_JWT_SECRET")

	if !ok {
		configErrors = append(configErrors,
			fmt.Errorf("LF_JWT_SECRET: config: %w", ErrMissing))
	} else if len(jwtSecret) < 32 {
		configErrors = append(configErrors,
			fmt.Errorf("LF_JWT_SECRET: config:  %w", ErrInvalid))
	} else {
		cfg.JWTSecret = jwtSecret
	}

	cacheSize, ok := l.get("LF_CACHE_SIZE")

	if !ok {
		cfg.CacheSize = 100000
	} else {
		n, err := strconv.Atoi(cacheSize)

		if err != nil || n <= 0 {
			configErrors = append(configErrors,
				fmt.Errorf("LF_CACHE_SIZE: config: %w", ErrInvalid))
		} else {
			cfg.CacheSize = n
		}
	}

	logLevel, ok := l.get("LF_LOG_LEVEL")

	if !ok {
		cfg.LogLevel = slog.LevelInfo
	}

	level, err := ParseLevel(logLevel)

	if err == nil {
		cfg.LogLevel = level
	} else {
		configErrors = append(configErrors,
			fmt.Errorf("LF_LOG_LEVEL: config: %w", ErrInvalid))
	}

	if len(configErrors) != 0 {
		return Config{}, errors.Join(configErrors...)
	}

	return cfg, nil

}

func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, ErrInvalid
	}
}

func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_addr", c.HTTPAddr),
		slog.String("admin_addr", c.AdminAddr),
		slog.String("tcp_addr", c.TCPAddr),
		slog.String("pg_dsn", "[redacted]"),
		slog.String("mongo_uri", "[redacted]"),
		slog.String("jwt_secret", "[redacted]"),
		slog.Int("cache_size", c.CacheSize),
		slog.String("log_level", c.LogLevel.String()),
	)
}
