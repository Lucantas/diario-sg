package config

import "testing"

func TestLoadWorkerReadsEventsMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("GCP_PROJECT_ID", "p")
	t.Setenv("GAZETTE_BUCKET", "b")
	t.Setenv("TOPIC_GAZETTE_INDEXED", "gazette-indexed")
	t.Setenv("EVENTS", "push-http")
	t.Setenv("PUSH_BASE_URL", "http://localhost:8081")

	c, err := Load(RoleWorker)

	if err != nil || c.Events != "push-http" || c.PushBaseURL != "http://localhost:8081" {
		t.Fatalf("%+v %v", c, err)
	}
}
