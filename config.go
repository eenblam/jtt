package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const DefaultJailBaseURL = "https://omsweb.public-safety-cloud.com"

type JailConfig struct {
	// How we describe the jail
	PrettyName string
	// URL for the jail. Usually "https://omsweb.public-safety-cloud.com", but not always!
	BaseURL string
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
	for i := range config.Jails {
		jailConfig := &config.Jails[i]
		// Currently, most jails don't have a domain name in the config since I haven't looked them up yet.
		if jailConfig.BaseURL == "" {
			jailConfig.BaseURL = DefaultJailBaseURL
		}
	}
	return nil
}
