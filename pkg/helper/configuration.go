package helper

import (
	"os"

	database "github.com/yishak-cs/Neo4j_DB/internal/database"
)

// LoadConfigFromEnv loads Neo4j configuration from environment variables
func LoadConfigFromEnv() database.Config {
	return database.Config{
		URI:      getEnvOrDefault("NEO4J_URI", ""),
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", ""),
		Database: getEnvOrDefault("NEO4J_DATABASE", "neo4j"),
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
