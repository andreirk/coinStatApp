package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all app configuration
type Config struct {
	// Server
	HTTPPort string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// ClickHouse
	ClickHouseDSN string

	// Kafka
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string
	KafkaBatchSize     int
	KafkaBatchTimeout  int // milliseconds

	ClickhouseUsername string
	ClickhousePassword string
	ClickhouseAddr     string
	ClickhouseTimeout  int

	// App settings
	EventBufferSize int
	Debug           bool
}

// LoadConfig loads configuration from environment variables, with optional .env file
func LoadConfig() *Config {
	// Load .env file if it exists
	err := godotenv.Load(filepath.Join("../..", ".env")) // Ignore error if .env doesn't exist
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
		panic(err)
	}

	cfg := &Config{
		// Server
		HTTPPort: getEnv("HTTP_PORT", "8080"),

		// Redis
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// ClickHouse
		ClickHouseDSN:      getEnv("CLICKHOUSE_DSN", "localhost:9000"),
		ClickhouseUsername: getEnv("CLICKHOUSE_USERNAME", ""),
		ClickhousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),
		ClickhouseAddr:     getEnv("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickhouseTimeout:  getEnvAsInt("CLICKHOUSE_TIMEOUT", 10),

		// Kafka
		KafkaBrokers:       getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}, ","),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "swaps"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "coinstat-group"),
		KafkaBatchSize:     getEnvAsInt("KAFKA_BATCH_SIZE", 500),
		KafkaBatchTimeout:  getEnvAsInt("KAFKA_BATCH_TIMEOUT", 3000),

		// App settings
		EventBufferSize: getEnvAsInt("EVENT_BUFFER_SIZE", 10000),
		Debug:           getEnvAsBool("DEBUG", false),
	}

	return cfg
}

// Helper functions for parsing environment variables
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsSlice(key string, defaultVal []string, sep string) []string {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	return strings.Split(valStr, sep)
}
