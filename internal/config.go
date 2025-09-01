package internal

import (
	"os"
	"strconv"
	"strings"
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
	get := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return Config{
		Addr:         get("HTTP_ADDR", ":8081"),
		PGURL:        get("PG_URL", "postgres://postgres:postgres@localhost:5432/orders?sslmode=disable"),
		Brokers:      strings.Split(get("KAFKA_BROKERS", "localhost:29092"), ","),
		Topic:        get("KAFKA_TOPIC", "orders"),
		Group:        get("KAFKA_GROUP_ID", "order-svc"),
		WarmN:        1000,
		CacheEnabled: loadCache(),
	}
}

func loadCache() bool {
	v := os.Getenv("CACHE_ENABLED")
	on, _ := strconv.ParseBool(v) // "1"/"true" -> true
	if v == "" {
		on = true
	}
	return on
}
