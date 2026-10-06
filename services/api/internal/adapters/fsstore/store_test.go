package fsstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPutWritesFileUnderDir(t *testing.T) {
	dir := t.TempDir()

	err := New(dir).Put(context.Background(), "latest/manifest.json", "application/json", strings.NewReader(`{"ok":true}`))

	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "latest", "manifest.json"))
	if err != nil || string(b) != `{"ok":true}` {
		t.Fatalf("%q %v", b, err)
	}
}

func TestPutRefusesPathsOutsideDir(t *testing.T) {
	if err := New(t.TempDir()).Put(context.Background(), "../fora.txt", "text/plain", strings.NewReader("x")); err == nil {
		t.Fatal("caminho fora do diretório deveria falhar")
	}
}
