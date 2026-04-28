package app

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr           string
	TargetKind     string
	TargetName     string
	TargetNS       string
	AllowedOrigins []string
	IdleTimeout    time.Duration
	Version        string
	GitSHA         string
}

func LoadConfig() Config {
	cfg := Config{
		Addr:        getenvDefault("PORT", ":8080"),
		TargetKind:  getenvDefault("TARGET_KIND", "Deployment"),
		TargetName:  os.Getenv("TARGET_NAME"),
		TargetNS:    os.Getenv("TARGET_NAMESPACE"),
		IdleTimeout: 30 * time.Minute,
		Version:     getenvDefault("VERSION", "dev"),
		GitSHA:      getenvDefault("GIT_SHA", "unknown"),
	}

	if v := os.Getenv("IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.IdleTimeout = d
		}
	}

	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		parts := strings.Split(raw, ",")
		cfg.AllowedOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			s := strings.TrimSpace(p)
			if s != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, s)
			}
		}
	}

	return cfg
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
