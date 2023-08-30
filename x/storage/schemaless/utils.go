package schemaless

import (
	"os"
)

func GetEnvOr(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
