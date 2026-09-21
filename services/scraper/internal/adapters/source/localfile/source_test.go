package localfile

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceServesTheGivenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edicao.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7 conteúdo"), 0o600); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)

	src, err := New(path, day, "1771", "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf")
	if err != nil {
		t.Fatal(err)
	}
	editions, err := src.ListEditions(context.Background(), day.AddDate(0, 0, -3), day)
	if err != nil || len(editions) != 1 {
		t.Fatalf("esperava 1 edição, veio %d (%v)", len(editions), err)
	}
	e := editions[0]
	if e.Number != "1771" || !e.PublishedAt.Equal(day) || e.StoragePath() != "gazettes/2026/09/18/edicao-1771.pdf" {
		t.Errorf("edição inesperada: %+v", e)
	}
	rc, err := src.Download(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if string(body) != "%PDF-1.7 conteúdo" {
		t.Errorf("conteúdo inesperado: %q", body)
	}
}

func TestNewRejectsMissingFileAndZeroDate(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "nao-existe.pdf"), time.Now(), "", "u"); err == nil {
		t.Error("esperava erro para arquivo inexistente")
	}
	path := filepath.Join(t.TempDir(), "x.pdf")
	os.WriteFile(path, []byte("x"), 0o600)
	if _, err := New(path, time.Time{}, "", "u"); err == nil {
		t.Error("esperava erro para data zero")
	}
}
