package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ProjectID          string
	Bucket             string
	TopicFetched       string
	SourceURL          string
	Lookback           time.Duration
	PubSubEmulatorHost string
	StorageEmulator    string
}

func Load() (Config, error) {
	days, err := strconv.Atoi(getenv("LOOKBACK_DAYS", "3"))
	if err != nil || days < 1 {
		days = 3
	}
	c := Config{
		ProjectID:          os.Getenv("GCP_PROJECT_ID"),
		Bucket:             os.Getenv("GAZETTE_BUCKET"),
		TopicFetched:       os.Getenv("TOPIC_GAZETTE_FETCHED"),
		SourceURL:          getenv("SOURCE_URL", "https://do.pmsg.rj.gov.br/"),
		Lookback:           time.Duration(days) * 24 * time.Hour,
		PubSubEmulatorHost: os.Getenv("PUBSUB_EMULATOR_HOST"),
		StorageEmulator:    os.Getenv("STORAGE_EMULATOR_HOST"),
	}
	var missing []string
	for k, v := range map[string]string{"GCP_PROJECT_ID": c.ProjectID, "GAZETTE_BUCKET": c.Bucket, "TOPIC_GAZETTE_FETCHED": c.TopicFetched} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return c, fmt.Errorf("variáveis obrigatórias ausentes: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
