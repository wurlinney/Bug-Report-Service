package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv string

	HTTP struct {
		Addr         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	Log struct {
		Level string
	}

	CORS struct {
		AllowedOrigins []string
	}

	RateLimit struct {
		RPS   float64
		Burst int
	}

	DB struct {
		URL string
	}

	JWT struct {
		Issuer     string
		Secret     string
		AccessTTL  time.Duration
		RefreshTTL time.Duration
	}

	S3 struct {
		Endpoint  string
		Region    string
		Bucket    string
		AccessKey string
		SecretKey string
	}
}

func Load() (Config, error) {
	var c Config

	c.AppEnv = getString("APP_ENV", "local")

	c.HTTP.Addr = getString("HTTP_ADDR", ":8080")
	c.HTTP.ReadTimeout = getDuration("HTTP_READ_TIMEOUT", 10*time.Second)
	c.HTTP.WriteTimeout = getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second)
	c.HTTP.IdleTimeout = getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)

	c.Log.Level = getString("LOG_LEVEL", "info")

	c.CORS.AllowedOrigins = getCSV("CORS_ALLOWED_ORIGINS", "*")

	c.RateLimit.RPS = getFloat("RATE_LIMIT_RPS", 10)
	c.RateLimit.Burst = getInt("RATE_LIMIT_BURST", 20)

	c.DB.URL = getString("DATABASE_URL", "")

	c.JWT.Issuer = getString("JWT_ISSUER", "bug-report-service")
	c.JWT.Secret = getString("JWT_SECRET", "")
	c.JWT.AccessTTL = getDuration("JWT_ACCESS_TTL", 15*time.Minute)
	c.JWT.RefreshTTL = getDuration("JWT_REFRESH_TTL", 30*24*time.Hour)

	c.S3.Endpoint = getString("S3_ENDPOINT", "http://minio:9000")
	c.S3.Region = getString("S3_REGION", "us-east-1")
	c.S3.Bucket = getString("S3_BUCKET", "bug-attachments")
	c.S3.AccessKey = getString("S3_ACCESS_KEY", "")
	c.S3.SecretKey = getString("S3_SECRET_KEY", "")

	if c.HTTP.Addr == "" {
		return Config{}, errors.New("HTTP_ADDR is empty")
	}
	if c.RateLimit.RPS <= 0 || c.RateLimit.Burst <= 0 {
		return Config{}, errors.New("rate limit must be positive")
	}
	if c.JWT.AccessTTL <= 0 || c.JWT.RefreshTTL <= 0 {
		return Config{}, errors.New("jwt ttls must be positive")
	}

	// In non-local environments we require full configuration.
	if strings.ToLower(strings.TrimSpace(c.AppEnv)) != "local" {
		if strings.TrimSpace(c.DB.URL) == "" {
			return Config{}, errors.New("DATABASE_URL is empty")
		}
		if strings.TrimSpace(c.JWT.Secret) == "" {
			return Config{}, errors.New("JWT_SECRET is empty")
		}
		if strings.TrimSpace(c.S3.Bucket) == "" {
			return Config{}, errors.New("S3_BUCKET is empty")
		}
		if strings.TrimSpace(c.S3.AccessKey) == "" || strings.TrimSpace(c.S3.SecretKey) == "" {
			return Config{}, errors.New("S3_ACCESS_KEY or S3_SECRET_KEY is empty")
		}
	}

	return c, nil
}

func getString(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getFloat(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func getDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getCSV(key, def string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		v = def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}
