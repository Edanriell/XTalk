package config

import "os"

// GetEnv returns the environment variable value or a default.
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
