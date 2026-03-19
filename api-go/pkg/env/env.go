package env

import (
	"os"
	"strconv"
	"strings"
)

// GetEnvOrDefault returns the environment variable value or defaultValue if unset or empty.
func GetEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvAsInt parses the environment variable as int. Returns defaultValue if unset or invalid.
func GetEnvAsInt(key string, defaultValue int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue, err
	}
	return i, nil
}

// GetEnvAsBool parses the environment variable as bool. Returns defaultValue if unset or invalid.
func GetEnvAsBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

// ParseCommaSeparated splits s by commas and trims each part.
func ParseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
