package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type Role string

const (
	RoleAPI     Role = "api"
	RoleWorker  Role = "worker"
	RoleMigrate Role = "migrate"
	RoleReindex Role = "reindex"
	RoleDump    Role = "dump"
	RoleReports Role = "reports"
	RoleKeys    Role = "keys"
)

type Config struct {
	Port               string
	DatabaseURL        string
	ProjectID          string
	Bucket             string
	DumpsBucket        string
	TopicIndexed       string
	PubSubEmulatorHost string
	StorageEmulator    string
	Notifier           string
	ResendAPIKey       string
	EmailFrom          string
	PublicWebURL       string
}

func Load(role Role) (Config, error) {
	c := Config{
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		ProjectID:          os.Getenv("GCP_PROJECT_ID"),
		Bucket:             os.Getenv("GAZETTE_BUCKET"),
		DumpsBucket:        os.Getenv("DUMPS_BUCKET"),
		TopicIndexed:       os.Getenv("TOPIC_GAZETTE_INDEXED"),
		PubSubEmulatorHost: os.Getenv("PUBSUB_EMULATOR_HOST"),
		StorageEmulator:    os.Getenv("STORAGE_EMULATOR_HOST"),
		Notifier:           getenv("NOTIFIER", "log"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		EmailFrom:          os.Getenv("EMAIL_FROM"),
		PublicWebURL:       getenv("PUBLIC_WEB_URL", "http://localhost:5173"),
	}

	required := map[string]string{"DATABASE_URL": c.DatabaseURL}
	if role == RoleWorker {
		required["GCP_PROJECT_ID"] = c.ProjectID
		required["GAZETTE_BUCKET"] = c.Bucket
		required["TOPIC_GAZETTE_INDEXED"] = c.TopicIndexed
	}
	if role == RoleReindex || role == RoleAPI {
		required["GAZETTE_BUCKET"] = c.Bucket
	}
	if role == RoleDump {
		required["DUMPS_BUCKET"] = c.DumpsBucket
	}
	if (role == RoleAPI || role == RoleWorker) && c.Notifier == "resend" {
		required["RESEND_API_KEY"] = c.ResendAPIKey
		required["EMAIL_FROM"] = c.EmailFrom
	}
	var missing []string
	for k, v := range required {
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
