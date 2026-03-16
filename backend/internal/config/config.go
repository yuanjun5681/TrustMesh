package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	JWTSecret              string
	TokenTTL               time.Duration
	LogLevel               string
	AllowAllCORS           bool
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	ShutdownGrace          time.Duration
	HeartbeatTTL           time.Duration
	HeartbeatSweepInterval time.Duration
	MongoEnabled           bool
	MongoURI               string
	MongoDatabase          string
	MongoTimeout           time.Duration
	NATSEnabled            bool
	NATSURL                string
	NATSClient             string
	NATSTimeout            time.Duration
}

func Load() Config {
	return Config{
		Port:                   getEnv("PORT", "8080"),
		JWTSecret:              getEnv("JWT_SECRET", "trustmesh-dev-secret"),
		TokenTTL:               getEnvDuration("TOKEN_TTL", 24*time.Hour),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		AllowAllCORS:           getEnvBool("ALLOW_ALL_CORS", true),
		ReadTimeout:            getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:           getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		ShutdownGrace:          getEnvDuration("SHUTDOWN_GRACE", 8*time.Second),
		HeartbeatTTL:           getEnvDuration("HEARTBEAT_TTL", 30*time.Second),
		HeartbeatSweepInterval: getEnvDuration("HEARTBEAT_SWEEP_INTERVAL", 5*time.Second),
		MongoEnabled:           getEnvBool("MONGO_ENABLED", true),
		MongoURI:               getEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase:          getEnv("MONGO_DATABASE", "trustmesh"),
		MongoTimeout:           getEnvDuration("MONGO_TIMEOUT", 5*time.Second),
		NATSEnabled:            getEnvBool("NATS_ENABLED", true),
		NATSURL:                getEnv("NATS_URL", "nats://127.0.0.1:4222"),
		NATSClient:             getEnv("NATS_CLIENT_NAME", "trustmesh-backend"),
		NATSTimeout:            getEnvDuration("NATS_TIMEOUT", 3*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}
