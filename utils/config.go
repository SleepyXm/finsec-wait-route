package utils

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	DevServer        string
	DevServerBackend string
	ResendAPIKey     string
}

var Cfg Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	Cfg = Config{
		DatabaseURL:      requireEnv("DATABASE"),
		DevServer:        requireEnv("DEV_SERVER"),
		DevServerBackend: requireEnv("DEV_SERVER_BACKEND"),
		ResendAPIKey:     requireEnv("RESEND_API_KEY"),
	}
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Required env var %s is not set", key)
	}
	return val
}

func requireEnvInt(key string) int {
	val := requireEnv(key)
	n, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("Env var %s must be an integer, got: %s", key, val)
	}
	return n
}
