package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppBaseURL           string
	DatabaseURL          string
	SESRegion            string
	SESFromEmail         string
	MaxSecretBytes       int
	MaxExpiryMinutes     int
	DefaultExpiryMinutes int
	DefaultOneTime       bool
	AdminStatsToken      string
	LogLevel             slog.Level
	RateLimitCreateHour  int
	RateLimitEmailHour   int
	RecaptchaSiteKey     string
	RecaptchaProjectID   string
	RecaptchaAPIKey      string
	RecaptchaMinScore    float64
	Port                 string
}

func Load() Config {
	return Config{
		AppBaseURL:           trimRightSlash(getenv("APP_BASE_URL", "http://localhost:8080")),
		DatabaseURL:          getenv("DATABASE_URL", "postgres://bquick_secret:bquick_secret@localhost:5432/bquick_secret?sslmode=disable"),
		SESRegion:            getenv("SES_REGION", ""),
		SESFromEmail:         getenv("SES_FROM_EMAIL", ""),
		MaxSecretBytes:       getenvInt("MAX_SECRET_BYTES", 262144),
		MaxExpiryMinutes:     getenvInt("MAX_EXPIRY_MINUTES", 10080),
		DefaultExpiryMinutes: getenvInt("DEFAULT_EXPIRY_MINUTES", 1440),
		DefaultOneTime:       getenvBool("DEFAULT_ONE_TIME", true),
		AdminStatsToken:      getenv("ADMIN_STATS_TOKEN", ""),
		LogLevel:             parseLevel(getenv("LOG_LEVEL", "info")),
		RateLimitCreateHour:  getenvInt("RATE_LIMIT_CREATE_PER_HOUR", 20),
		RateLimitEmailHour:   getenvInt("RATE_LIMIT_EMAIL_PER_HOUR", 20),
		RecaptchaSiteKey:     getenv("RECAPTCHA_SITE_KEY", ""),
		RecaptchaProjectID:   getenv("RECAPTCHA_PROJECT_ID", ""),
		RecaptchaAPIKey:      getenv("RECAPTCHA_API_KEY", ""),
		RecaptchaMinScore:    getenvFloat("RECAPTCHA_MIN_SCORE", 0.5),
		Port:                 getenv("PORT", "8081"),
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getenv(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func getenvFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(getenv(key, ""), 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	value := strings.ToLower(getenv(key, ""))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func trimRightSlash(value string) string {
	return strings.TrimRight(value, "/")
}
