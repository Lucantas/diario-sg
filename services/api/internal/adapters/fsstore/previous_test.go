package fsstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func writePreviousDump(t *testing.T, dir, name, content, sha string) {
	t.Helper()
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		sum := sha256.Sum256(gz.Bytes())
		sha = hex.EncodeToString(sum[:])
	}
	manifest, err := json.Marshal(domain.DumpManifest{Files: []domain.DumpFile{{Name: name, SHA256: sha}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), gz.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPreviousDumpReadsHeaderAndRows(t *testing.T) {
	dir := t.TempDir()
	writePreviousDump(t, dir, "acts.csv.gz", "id,body\na1,\"linha 1\nlinha 2\"\n", "")

	header, rows, err := NewPreviousDump(dir).Table(context.Background(), "acts")

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(header, []string{"id", "body"}) || !reflect.DeepEqual(rows, [][]string{{"a1", "linha 1\nlinha 2"}}) {
		t.Fatalf("%q %q", header, rows)
	}
}

func TestPreviousDumpWithoutManifestMeansNoPreviousDump(t *testing.T) {
	_, _, err := NewPreviousDump(filepath.Join(t.TempDir(), "nao-existe")).Table(context.Background(), "acts")

	if !errors.Is(err, domain.ErrNoPreviousDump) {
		t.Fatalf("esperava ErrNoPreviousDump, veio %v", err)
	}
}

func TestPreviousDumpWithoutTheTableMeansNoPreviousDump(t *testing.T) {
	dir := t.TempDir()
	writePreviousDump(t, dir, "gazettes.csv.gz", "id\n", "")

	_, _, err := NewPreviousDump(dir).Table(context.Background(), "acts")

	if !errors.Is(err, domain.ErrNoPreviousDump) {
		t.Fatalf("esperava ErrNoPreviousDump, veio %v", err)
	}
}

func TestPreviousDumpRefusesFileThatDoesNotMatchManifest(t *testing.T) {
	dir := t.TempDir()
	writePreviousDump(t, dir, "acts.csv.gz", "id\na1\n", strings.Repeat("0", 64))

	_, _, err := NewPreviousDump(dir).Table(context.Background(), "acts")

	if err == nil || errors.Is(err, domain.ErrNoPreviousDump) || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("esperava erro de sha256, veio %v", err)
	}
}
