package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DBUrl            string
	JWTSecret        string
	JFHost           string
	JFApiKey         string
	DisableDashboard bool
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		host := getEnv("POSTGRES_IP", "localhost")
		port := getEnv("POSTGRES_PORT", "5432")
		user := getEnv("POSTGRES_USER", "postgres")
		pass := getEnv("POSTGRES_PASSWORD", "")
		name := getEnv("POSTGRES_DB", "jellystics")
		dbUrl = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
			host, port, user, pass, name)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		Port:             getEnv("PORT", "3000"),
		DBUrl:            dbUrl,
		JWTSecret:        secret,
		JFHost:           os.Getenv("JF_HOST"),
		JFApiKey:         os.Getenv("JF_API_KEY"),
		DisableDashboard: os.Getenv("DISABLE_DASHBOARD") == "true",
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
