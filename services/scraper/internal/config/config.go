package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type Config struct {
	ProjectID          string
	Bucket             string
	TopicFetched       string
	TopicRuns          string
	Source             string
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
		TopicRuns:          os.Getenv("TOPIC_FETCH_COMPLETED"),
		Source:             getenv("SOURCE", domain.SourceDiarioPrefeitura),
		Lookback:           time.Duration(days) * 24 * time.Hour,
		PubSubEmulatorHost: os.Getenv("PUBSUB_EMULATOR_HOST"),
		StorageEmulator:    os.Getenv("STORAGE_EMULATOR_HOST"),
	}
	if !domain.ValidSource(c.Source) {
		return c, fmt.Errorf("SOURCE desconhecida: %q", c.Source)
	}
	c.SourceURL = getenv("SOURCE_URL", defaultURLs[c.Source])
	if err := checkSourceURL(c.Source, c.SourceURL); err != nil {
		return c, err
	}
	var missing []string
	for k, v := range map[string]string{"GCP_PROJECT_ID": c.ProjectID, "GAZETTE_BUCKET": c.Bucket, "TOPIC_GAZETTE_FETCHED": c.TopicFetched, "TOPIC_FETCH_COMPLETED": c.TopicRuns} {
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

var defaultURLs = map[string]string{
	domain.SourceDiarioPrefeitura: "https://do.pmsg.rj.gov.br/",
	domain.SourceDiarioCamara:     "https://www.cmsg.rj.gov.br/diariooficialeletronico/",
}

func checkSourceURL(source, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("SOURCE_URL inválida: %q", raw)
	}
	for other, def := range defaultURLs {
		d, _ := url.Parse(def)
		if other != source && strings.EqualFold(u.Host, d.Host) {
			return fmt.Errorf("SOURCE_URL %q é de %s, mas SOURCE é %s", raw, other, source)
		}
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
