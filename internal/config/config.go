package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	JWTSecret     []byte
	JWTExpiry     time.Duration
	DatabaseURL   string
	RedisURL      string
	ClientURLs    []string
	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPass      string
	SMTPFromName  string
	SMTPFromEmail string
	OTPExpiry     time.Duration
	OTPCooldown   time.Duration
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func parseJWTExpiry(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		if n, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil {
			return time.Duration(n) * 24 * time.Hour
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return 7 * 24 * time.Hour
}


func parseList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" { out = append(out, p) }
	}
	return out
}

func Load() *Config {
	_ = godotenv.Load()
	otpMin, _ := strconv.Atoi(env("OTP_EXPIRES_MIN", "10"))
	coolSec, _ := strconv.Atoi(env("RESEND_OTP_COOLDOWN_SEC", "60"))
	return &Config{
		Port:          env("PORT", "8080"),
		JWTSecret:     []byte(env("JWT_SECRET", "dev-secret-change-me")),
		JWTExpiry:     parseJWTExpiry(env("JWT_EXPIRES_IN", "7d")),
		DatabaseURL:   env("DATABASE_URL", "postgres://bio_user:Imanairankunda123@localhost:5432/bio?sslmode=disable"),
		RedisURL:      env("REDIS_URL", ""),
		ClientURLs:    parseList(env("CLIENT_URL", "http://localhost:3000")),
		SMTPHost:      env("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:      env("SMTP_PORT", "587"),
		SMTPUser:      env("SMTP_USER", ""),
		SMTPPass:      env("SMTP_PASS", ""),
		SMTPFromName:  env("SMTP_FROM_NAME", "bla.link"),
		SMTPFromEmail: env("SMTP_FROM_EMAIL", ""),
		OTPExpiry:     time.Duration(otpMin) * time.Minute,
		OTPCooldown:   time.Duration(coolSec) * time.Second,
	}
}
