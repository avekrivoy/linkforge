package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func lookupFromMap(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"LF_PG_DSN":     "postgres://user:password@localhost:5432/db",
		"LF_MONGO_URI":  "mongodb://user:password@localhost:27017",
		"LF_JWT_SECRET": "this-is-a-secret-with-at-least-32-bytes",
	}
}

func requireErrorIs(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("expected error %v, got %v", target, err)
	}
}

func TestLoadDefaults(t *testing.T) {
	loader := NewEnvLoader(lookupFromMap(validEnv()))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}

	if cfg.AdminAddr != "127.0.0.1:6060" {
		t.Errorf("AdminAddr = %q, want %q", cfg.AdminAddr, "127.0.0.1:6060")
	}

	if cfg.TCPAddr != ":7070" {
		t.Errorf("TCPAddr = %q, want %q", cfg.TCPAddr, ":7070")
	}

	if cfg.CacheSize != 100000 {
		t.Errorf("CacheSize = %d, want 100000", cfg.CacheSize)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadOverridesWithSpaces(t *testing.T) {
	env := validEnv()

	env["LF_HTTP_ADDR"] = "  :9090  "
	env["LF_ADMIN_ADDR"] = "  localhost:6061  "
	env["LF_TCP_ADDR"] = "  :7071  "
	env["LF_PG_DSN"] = "  postgres://localhost/test  "
	env["LF_MONGO_URI"] = "  mongodb://localhost/test  "
	env["LF_JWT_SECRET"] = "  this-is-a-secret-with-at-least-32-bytes  "
	env["LF_CACHE_SIZE"] = "  500  "
	env["LF_LOG_LEVEL"] = "  DEBUG  "

	loader := NewEnvLoader(lookupFromMap(env))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}

	if cfg.AdminAddr != "localhost:6061" {
		t.Errorf("AdminAddr = %q, want %q", cfg.AdminAddr, "localhost:6061")
	}

	if cfg.TCPAddr != ":7071" {
		t.Errorf("TCPAddr = %q, want %q", cfg.TCPAddr, ":7071")
	}

	if cfg.PGDSN != "postgres://localhost/test" {
		t.Errorf("PGDSN was not trimmed: %q", cfg.PGDSN)
	}

	if cfg.MongoURI != "mongodb://localhost/test" {
		t.Errorf("MongoURI was not trimmed: %q", cfg.MongoURI)
	}

	if cfg.JWTSecret != "this-is-a-secret-with-at-least-32-bytes" {
		t.Errorf("JWTSecret was not trimmed")
	}

	if cfg.CacheSize != 500 {
		t.Errorf("CacheSize = %d, want 500", cfg.CacheSize)
	}

	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoadValidationRules(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError error
	}{
		{
			name:      "missing PostgreSQL DSN",
			key:       "LF_PG_DSN",
			value:     "",
			wantError: ErrMissing,
		},
		{
			name:      "missing Mongo URI",
			key:       "LF_MONGO_URI",
			value:     "",
			wantError: ErrMissing,
		},
		{
			name:      "missing JWT secret",
			key:       "LF_JWT_SECRET",
			value:     "",
			wantError: ErrMissing,
		},
		{
			name:      "JWT secret too short",
			key:       "LF_JWT_SECRET",
			value:     "short-secret",
			wantError: ErrInvalid,
		},
		{
			name:      "bad HTTP port",
			key:       "LF_HTTP_ADDR",
			value:     "localhost",
			wantError: ErrInvalid,
		},
		{
			name:      "bad admin port",
			key:       "LF_ADMIN_ADDR",
			value:     "localhost:abc",
			wantError: ErrInvalid,
		},
		{
			name:      "bad TCP port",
			key:       "LF_TCP_ADDR",
			value:     ":abc",
			wantError: ErrInvalid,
		},
		{
			name:      "HTTP port out of range",
			key:       "LF_HTTP_ADDR",
			value:     ":65536",
			wantError: ErrInvalid,
		},
		{
			name:      "admin port out of range",
			key:       "LF_ADMIN_ADDR",
			value:     ":70000",
			wantError: ErrInvalid,
		},
		{
			name:      "TCP port out of range",
			key:       "LF_TCP_ADDR",
			value:     ":0",
			wantError: ErrInvalid,
		},
		{
			name:      "bad cache number",
			key:       "LF_CACHE_SIZE",
			value:     "abc",
			wantError: ErrInvalid,
		},
		{
			name:      "zero cache size",
			key:       "LF_CACHE_SIZE",
			value:     "0",
			wantError: ErrInvalid,
		},
		{
			name:      "negative cache size",
			key:       "LF_CACHE_SIZE",
			value:     "-10",
			wantError: ErrInvalid,
		},
		{
			name:      "bad log level",
			key:       "LF_LOG_LEVEL",
			value:     "trace",
			wantError: ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnv()
			env[tt.key] = tt.value

			loader := NewEnvLoader(lookupFromMap(env))

			cfg, err := loader.Load(context.Background())

			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			requireErrorIs(t, err, tt.wantError)

			if cfg != (Config{}) {
				t.Errorf("Load() returned non-zero config on error: %+v", cfg)
			}
		})
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	env := map[string]string{
		"LF_HTTP_ADDR":  "invalid-address",
		"LF_PG_DSN":     "",
		"LF_MONGO_URI":  "",
		"LF_JWT_SECRET": "short",
		"LF_CACHE_SIZE": "zero",
		"LF_LOG_LEVEL":  "trace",
	}

	loader := NewEnvLoader(lookupFromMap(env))

	cfg, err := loader.Load(context.Background())

	if err == nil {
		t.Fatal("expected validation errors, got nil")
	}

	if cfg != (Config{}) {
		t.Errorf("expected zero Config, got %+v", cfg)
	}

	for _, key := range []string{
		"LF_HTTP_ADDR",
		"LF_PG_DSN",
		"LF_MONGO_URI",
		"LF_JWT_SECRET",
		"LF_CACHE_SIZE",
		"LF_LOG_LEVEL",
	} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("combined error does not contain %s: %v", key, err)
		}
	}

	requireErrorIs(t, err, ErrMissing)
	requireErrorIs(t, err, ErrInvalid)
}

func TestLoadDoesNotExposeSecretsInErrors(t *testing.T) {
	env := validEnv()

	const secret = "very-sensitive-jwt-secret-value"
	env["LF_JWT_SECRET"] = secret
	env["LF_HTTP_ADDR"] = "invalid-address"

	loader := NewEnvLoader(lookupFromMap(env))

	_, err := loader.Load(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if strings.Contains(err.Error(), secret) {
		t.Errorf("error message exposes JWT secret: %v", err)
	}

	if strings.Contains(err.Error(), env["LF_PG_DSN"]) {
		t.Errorf("error message exposes PostgreSQL DSN: %v", err)
	}

	if strings.Contains(err.Error(), env["LF_MONGO_URI"]) {
		t.Errorf("error message exposes Mongo URI: %v", err)
	}
}

func TestLoadCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	loader := NewEnvLoader(lookupFromMap(validEnv()))

	cfg, err := loader.Load(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if cfg != (Config{}) {
		t.Errorf("expected zero Config, got %+v", cfg)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"uppercase", "DEBUG", slog.LevelDebug},
		{"mixed case", "InFo", slog.LevelInfo},
		{"spaces", "  warn  ", slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if err != nil {
				t.Fatalf("ParseLevel(%q) error = %v", tt.input, err)
			}

			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLevelInvalid(t *testing.T) {
	_, err := ParseLevel("trace")

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	requireErrorIs(t, err, ErrInvalid)
}

func TestConfigLogValueRedactsSecrets(t *testing.T) {
	const (
		pgSecret    = "postgres-super-secret"
		mongoSecret = "mongo-super-secret"
		jwtSecret   = "jwt-super-secret"
	)

	cfg := Config{
		HTTPAddr:  ":8080",
		AdminAddr: "localhost:6060",
		TCPAddr:   ":7070",
		PGDSN:     pgSecret,
		MongoURI:  mongoSecret,
		JWTSecret: jwtSecret,
		CacheSize: 100,
		LogLevel:  slog.LevelInfo,
	}

	value := cfg.LogValue()

	if value.Kind() != slog.KindGroup {
		t.Fatalf("LogValue() kind = %v, want Group", value.Kind())
	}

	attrs := value.Group()
	fields := make(map[string]string, len(attrs))

	for _, attr := range attrs {
		fields[attr.Key] = attr.Value.String()
	}

	for _, key := range []string{"pg_dsn", "mongo_uri", "jwt_secret"} {
		if fields[key] != "[redacted]" {
			t.Errorf("%s = %q, want [redacted]", key, fields[key])
		}
	}

	logOutput := fmt.Sprint(attrs)

	for _, secret := range []string{pgSecret, mongoSecret, jwtSecret} {
		if strings.Contains(logOutput, secret) {
			t.Errorf("LogValue() exposes secret %q", secret)
		}
	}
}
