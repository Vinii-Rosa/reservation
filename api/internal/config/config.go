package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string
	DatabaseURL string
	JWTSecret string
	AppPublicURL string
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPassword string
	SMTPFrom string
	CronCleanupReservations string
	WaitlistCallTimeoutMinutes int
}

func Load() Config {
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	waitlistTimeout, _ := strconv.Atoi(getEnv("WAITLIST_CALL_TIMEOUT_MINUTES", "10"))

	return Config{
		Port: getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://root:root@localhost:5432/reservation?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-me"),
		AppPublicURL: getEnv("APP_PUBLIC_URL", "http://localhost:8080"),
		SMTPHost: getEnv("SMTP_HOST", "localhost"),
		SMTPPort: smtpPort,
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@reservation.local"),
		CronCleanupReservations: getEnv("CRON_CLEANUP_RESERVATIONS", "0 * * * *"),
		WaitlistCallTimeoutMinutes: waitlistTimeout,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
