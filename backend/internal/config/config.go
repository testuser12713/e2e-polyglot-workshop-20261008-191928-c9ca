// Package config loads the API configuration from the environment.
//
// Values are read lazily (in Load, never at import time) and validated once, so
// a missing required variable refuses to start with a message that names it
// instead of a bare traceback.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// defaultCORSOrigin is used when CORS_ORIGIN is unset or still carries an
// unresolved ${service:...} placeholder (the web service may land later).
const defaultCORSOrigin = "http://localhost:5173"

// Config is the full runtime configuration of the API.
type Config struct {
	Port             string
	DatabaseURL      string
	ValkeyURL        string
	QueueName        string
	HourlyRateCents  int
	AuthSecret       string
	EmployeeEmail    string
	EmployeePassword string
	CORSOrigin       string
}

// Load reads the configuration from the environment and validates it. It is
// safe to call at runtime; nothing is read at package import time.
func Load() (Config, error) {
	cfg := Config{
		Port:             getenvDefault("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		ValkeyURL:        os.Getenv("VALKEY_URL"),
		QueueName:        getenvDefault("QUEUE_NAME", "workshop-invoices"),
		HourlyRateCents:  8900,
		AuthSecret:       os.Getenv("AUTH_SECRET"),
		EmployeeEmail:    normalizeCredential(os.Getenv("EMPLOYEE_EMAIL")),
		EmployeePassword: normalizeCredential(os.Getenv("EMPLOYEE_PASSWORD")),
		CORSOrigin:       normalizeOrigin(os.Getenv("CORS_ORIGIN")),
	}

	if raw := os.Getenv("HOURLY_RATE_CENTS"); raw != "" {
		cents, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("HOURLY_RATE_CENTS must be an integer number of cents, got %q", raw)
		}
		cfg.HourlyRateCents = cents
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate reports every required variable that is missing, so the process
// refuses to start and says exactly which one to set (see RUN.json).
func (c Config) Validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.ValkeyURL == "" {
		missing = append(missing, "VALKEY_URL")
	}
	if c.AuthSecret == "" {
		missing = append(missing, "AUTH_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s (see RUN.json)", strings.Join(missing, ", "))
	}
	return nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// normalizeCredential turns an injected credential into the value a human
// actually types. An environment value is often delivered with a trailing
// newline or wrapped in one pair of quotes by the surrounding shell or runner;
// bcrypt hashes those extra bytes, so the hash at seed time no longer matches
// the string the login form posts. It trims surrounding whitespace and strips a
// single pair of matching surrounding quotes, and is applied to EMPLOYEE_EMAIL
// and EMPLOYEE_PASSWORD only — never to a URL or key where such bytes matter.
func normalizeCredential(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			v = strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}

// normalizeOrigin turns an unset or unresolved origin into the dev default and
// trims a trailing slash so it matches the browser's Origin header.
func normalizeOrigin(value string) string {
	if value == "" || strings.Contains(value, "${") {
		return defaultCORSOrigin
	}
	return strings.TrimRight(value, "/")
}
