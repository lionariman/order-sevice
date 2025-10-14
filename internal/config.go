package internal

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr         string
	PGURL        string
	Brokers      []string
	Topic        string
	Group        string
	WarmN        int
	CacheEnabled bool
}

func Env() Config {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	get := func(k string) string {
		v := os.Getenv(k)
		if v == "" {
			log.Fatalf("Environment variable %s is not set", k)
		}
		return v
	}
	return Config{
		Addr:         get("HTTP_ADDR"),
		PGURL:        get("PG_URL"),
		Brokers:      strings.Split(get("KAFKA_BROKERS"), ","),
		Topic:        get("KAFKA_TOPIC"),
		Group:        get("KAFKA_GROUP_ID"),
		WarmN:        1000,
		CacheEnabled: loadCache(),
	}
}

func loadCache() bool {
	v := os.Getenv("CACHE_ENABLED")
	on, err := strconv.ParseBool(v) // "1"/"true" -> true
	if err != nil {
		log.Printf("Cache is not enabled (converting error): %v", err)
	}
	if v == "" {
		on = true
	}
	return on
}
