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

type Config struct {
	Jails []JailConfig
	// Directory to cache jail data
	Cache string
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	config := &Config{}
	err = json.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return config, nil
}
