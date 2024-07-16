package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

const DefaultJailBaseURL = "https://omsweb.public-safety-cloud.com"

type JailConfig struct {
	// How we describe the jail
	Title string
	// URL for the jail. Usually "https://omsweb.public-safety-cloud.com", but not always!
	BaseURL string
	// Used in API URLs
	Slug string
	// Web page to view roster; just here for logging
	IndexURL string
	// Whether or not we can currently pull data from the jail.
	// Jail might require search, consistently times out or errors, etc
	Usable bool
}

type AppConfig struct {
	Jails []JailConfig
	// Directory to cache jail data
	Cache string
}

// Marshal data from filename into provided config
func (config *AppConfig) LoadConfig(filename string) error {
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

type AppEnv struct {
	OpenAIAPIKey string // "JTT_OPENAI_API_KEY"
	// Directory to cache jail data
	ConfigPath string // "JTT_CONFIG_PATH"
}

// Load sets default values for empty optional environment variables
func (a *AppEnv) Load() {
	a.OpenAIAPIKey = os.Getenv("JTT_OPENAI_API_KEY")

	a.ConfigPath = os.Getenv("JTT_CONFIG_PATH")
	if a.ConfigPath == "" {
		a.ConfigPath = "config.json"
	} else {
		log.Printf("Using config file from environment (JTT_CONFIG_PATH=\"%s\"", a.ConfigPath)
	}
}

// Validate checks that required environment variables are set.
// This is a separate step from Load, since it shouldn't be run on init() during tests.
func (a *AppEnv) ValidateRequired() error {
	if a.OpenAIAPIKey == "" {
		return errors.New("JTT_OPENAI_API_KEY must be set")
	}
	return nil
}
