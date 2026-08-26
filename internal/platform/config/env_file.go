package config

import "github.com/joho/godotenv"

// LoadEnvFile loads dotenv values without replacing variables already present
// in the process environment.
func LoadEnvFile(path string) error {
	return godotenv.Load(path)
}
