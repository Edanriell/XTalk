package utils

import "os"

// TODO
// Code duplication
// Probabbly move to Shared
func GetEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}
