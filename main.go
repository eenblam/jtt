package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
)

var OpenAIAPIKey string

var Config = &AppConfig{}

func init() {
	OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	if OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY must be set")
	}

	configPath := os.Getenv("JTT_CONFIG_DIR")
	if configPath == "" {
		configPath = "config.json"
	} else {
		log.Printf("Using config file from environment (JTT_CONFIG_DIR=\"%s\"", configPath)
	}
	err := LoadConfig(Config, configPath)
	if err != nil {
		panic(err)
	}
}

func main() {
	for _, jailConfig := range Config.Jails {
		// Currently, most jails don't have a domain name in the config since I haven't looked them up yet.
		if jailConfig.DomainName == "" {
			jailConfig.DomainName = DefaultDomainName
		}

		// Right now we do nothing here. Later, the cached data can be used to update a remote database.
		_, err := LoadJailCached(&jailConfig)
		if err != nil {
			panic(err)
		}
	}
}

// LoadJailCached will load the jail data from cache if present, or crawl the jail and save it to the configured
// cache directory if not.
func LoadJailCached(jailConfig *JailConfig) (*Jail, error) {
	var jail *Jail
	filename := JailCachePath(jailConfig.Slug)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrNotExist) { // File doesn't exist; create it
		log.Printf("Cache miss for \"%s\"", filename)
		jail, err := CrawlJail(jailConfig.DomainName, jailConfig.Slug)
		if err != nil {
			return nil, err
		}
		err = SaveJail(jail)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else { // File exists; load from cache
		log.Printf("Loading jail data from \"%s\"", filename)
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &jail)
		if err != nil {
			return nil, err
		}
	}
	return jail, nil
}

// SaveJail saves the jail data to the configured cache directory.
func SaveJail(jail *Jail) error {
	data, err := json.MarshalIndent(jail, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jail data: %w", err)
	}
	filename := JailCachePath(jail.Name)
	log.Printf("Caching jail data as \"%s\"", filename)
	return os.WriteFile(filename, data, 0644)
}

// JailCachePath returns the path to the current jail cache file.
// Caching is currently implemented simply as a JSON file per jail per day.
func JailCachePath(jailName string) string {
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.json", jailName, today)
	return path.Join(Config.Cache, filename)
}
