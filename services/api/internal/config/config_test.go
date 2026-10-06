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

func TestLoadReadsTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("TRUSTED_PROXIES", " 172.64.0.0/13, 2606:4700::/32 ,")

	c, err := Load(RoleAPI)

	if err != nil || len(c.TrustedProxies) != 2 || c.TrustedProxies[1].String() != "2606:4700::/32" {
		t.Fatalf("%+v %v", c.TrustedProxies, err)
	}
}

func TestLoadRejectsInvalidTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("TRUSTED_PROXIES", "172.64.0.0/13, 1.2.3.4")

	if _, err := Load(RoleAPI); err == nil {
		t.Fatal("endereço sem máscara deveria ser recusado")
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
