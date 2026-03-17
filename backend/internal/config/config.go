package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	JWTSecret           string
	TokenTTL            time.Duration
	LogLevel            string
	AllowAllCORS        bool
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	ShutdownGrace       time.Duration
	MongoEnabled        bool
	MongoURI            string
	MongoDatabase       string
	MongoTimeout        time.Duration
	ClawSynapseAPIURL   string
	ClawSynapseNodeID   string
	ClawSynapseTimeout  time.Duration
	ClawSynapsePeerSync time.Duration
}

func Load() Config {
	return Config{
		Port:                getEnv("PORT", "8080"),
		JWTSecret:           getEnv("JWT_SECRET", "trustmesh-dev-secret"),
		TokenTTL:            getEnvDuration("TOKEN_TTL", 24*time.Hour),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		AllowAllCORS:        getEnvBool("ALLOW_ALL_CORS", true),
		ReadTimeout:         getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:        getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		ShutdownGrace:       getEnvDuration("SHUTDOWN_GRACE", 8*time.Second),
		MongoEnabled:        getEnvBool("MONGO_ENABLED", true),
		MongoURI:            getEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase:       getEnv("MONGO_DATABASE", "trustmesh"),
		MongoTimeout:        getEnvDuration("MONGO_TIMEOUT", 5*time.Second),
		ClawSynapseAPIURL:   getEnv("CLAWSYNAPSE_API_URL", "http://127.0.0.1:18080"),
		ClawSynapseNodeID:   getEnv("CLAWSYNAPSE_NODE_ID", ""),
		ClawSynapseTimeout:  getEnvDuration("CLAWSYNAPSE_TIMEOUT", 3*time.Second),
		ClawSynapsePeerSync: getEnvDuration("CLAWSYNAPSE_PEER_SYNC_INTERVAL", 10*time.Second),
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
