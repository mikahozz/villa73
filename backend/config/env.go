package config

import (
	"fmt"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
func LoadEnv() {
	// Try different possible locations for .env file
	locations := []string{
		".env",       // Current directory
		"../.env",    // Parent directory
		"../../.env", // Two levels up
	}

	var lastErr error
	for _, loc := range locations {
		if err := godotenv.Load(loc); err == nil {
			return
		} else {
			lastErr = err
		}
	}

	panic(fmt.Sprintf("No .env file found in any location. Last error: %v", lastErr))
}
