package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type JailConfig struct {
	// How we describe the jail
	PrettyName string
	// Domain for the jail. Usually "omsweb.public-safety-cloud.com", but not always!
	DomainName string
	// Used in API URLs
	Slug string
}

type AppConfig struct {
	Jails []JailConfig
	// Directory to cache jail data
	Cache string
}

// Marshal data from filename into provided config
func LoadConfig(config *AppConfig, filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	err = json.Unmarshal(file, config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return nil
}
