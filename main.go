package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var OPENAI_API_KEY string

func init() {
	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		log.Fatal("OPENAI_API_KEY must be set")
	}
}

func main() {
	config, err := LoadConfig("config.json")
	if err != nil {
		panic(err)
	}

	for _, jail := range config.Jails {
		err := UpdateJail(jail.Slug)
		if err != nil {
			panic(err)
		}
	}
}

func UpdateJail(jailName string) error {
	var jail *Jail
	filename := JailFileName(jailName)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrNotExist) { // File doesn't exist; create it
		log.Printf("Cache miss for \"%s\"", filename)
		jail, err := CrawlJail(jailName)
		if err != nil {
			return err
		}
		err = SaveJail(jail)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else { // File exists; load from cache
		log.Printf("Loading jail data from \"%s\"", filename)
		data, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &jail)
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveJail(jail *Jail) error {
	data, err := json.MarshalIndent(jail, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jail data: %w", err)
	}
	filename := JailFileName(jail.Name)
	log.Printf("Caching jail data as \"%s\"", filename)
	return os.WriteFile(filename, data, 0644)
}

func JailFileName(jailName string) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("cache/%s-%s.json", jailName, today)
}
