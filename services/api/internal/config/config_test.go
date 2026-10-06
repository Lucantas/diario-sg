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

func TestLoadReadsTrustedProxyHops(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("GAZETTE_BUCKET", "b")
	t.Setenv("TRUSTED_PROXY_HOPS", "2")

	c, err := Load(RoleAPI)

	if err != nil || c.TrustedProxyHops != 2 {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestLoadRejectsInvalidTrustedProxyHops(t *testing.T) {
	for _, v := range []string{"-1", "um"} {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("GAZETTE_BUCKET", "b")
		t.Setenv("TRUSTED_PROXY_HOPS", v)
		if _, err := Load(RoleAPI); err == nil {
			t.Errorf("%q deveria ser recusado", v)
		}
	}
}

func TestLoadAPIWithoutBucket(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("GAZETTE_BUCKET", "")

	if _, err := Load(RoleAPI); err != nil {
		t.Fatalf("a API sobe sem bucket e cai no PDF oficial: %v", err)
	}
}

func TestLoadDumpNeedsBucketOrDir(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("DUMPS_BUCKET", "")
	t.Setenv("DUMP_DIR", "")
	if _, err := Load(RoleDump); err == nil {
		t.Error("sem DUMPS_BUCKET e sem DUMP_DIR deveria falhar")
	}

	t.Setenv("DUMP_DIR", "/tmp/dump")
	c, err := Load(RoleDump)

	if err != nil || c.DumpDir != "/tmp/dump" {
		t.Fatalf("%+v %v", c, err)
	}
}
