package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type fakeSnapshot struct {
	tables       map[string]string
	order        []string
	failOn       string
	closed       bool
	headers      map[string][]string
	fingerprints []domain.GazetteFingerprint
	records      map[string][]domain.DumpRecord
	asked        map[string][]string
	wrote        []string
}

func (f *fakeSnapshot) Header(table string) ([]string, error) { return f.headers[table], nil }

func (f *fakeSnapshot) Fingerprints(context.Context) ([]domain.GazetteFingerprint, error) {
	return f.fingerprints, nil
}

func (f *fakeSnapshot) Records(_ context.Context, table string, ids []string) ([]domain.DumpRecord, error) {
	if f.asked == nil {
		f.asked = map[string][]string{}
	}
	f.asked[table] = ids
	return f.records[table], nil
}

func (f *fakeSnapshot) Snapshot(context.Context) (ports.DumpSnapshot, error) { return f, nil }
func (f *fakeSnapshot) Tables() []string                                     { return f.order }
func (f *fakeSnapshot) Close() error {
	f.closed = true
	return nil
}
func (f *fakeSnapshot) WriteTable(_ context.Context, table string, w io.Writer) (int, error) {
	f.wrote = append(f.wrote, table)
	if table == f.failOn {
		return 0, errors.New("consulta falhou")
	}
	content := f.tables[table]
	if _, err := io.WriteString(w, content); err != nil {
		return 0, err
	}
	return strings.Count(content, "\n") - 1, nil
}

type memObjects struct {
	names []string
	data  map[string][]byte
}

func (m *memObjects) Put(_ context.Context, name, _ string, body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if m.data == nil {
		m.data = map[string][]byte{}
	}
	m.names = append(m.names, name)
	m.data[name] = b
	return nil
}

func gunzip(t *testing.T, b []byte) string {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestPublishDumpUploadsTablesThenReadmeThenManifest(t *testing.T) {
	src := &fakeSnapshot{tables: map[string]string{"a": "x,y\n1,2\n", "b": "z\n3\n4\n"}, order: []string{"a", "b"}}
	objects := &memObjects{}
	now := time.Date(2026, 9, 27, 7, 0, 0, 0, time.UTC)

	report, err := NewPublishDump(src, objects, nil).Execute(context.Background(), now)

	if err != nil {
		t.Fatal(err)
	}
	manifest := report.Manifest
	if report.Incremental {
		t.Fatal("sem dump anterior o dump deve ser completo")
	}
	wantOrder := []string{"latest/a.csv.gz", "latest/b.csv.gz", "latest/LEIAME.txt", "latest/manifest.json"}
	if !reflect.DeepEqual(objects.names, wantOrder) {
		t.Fatalf("ordem inesperada: %v", objects.names)
	}
	if got := gunzip(t, objects.data["latest/a.csv.gz"]); got != "x,y\n1,2\n" {
		t.Fatalf("conteúdo de a inesperado: %q", got)
	}
	gz := objects.data["latest/b.csv.gz"]
	sum := sha256.Sum256(gz)
	b := manifest.Files[1]
	if b.Name != "b.csv.gz" || b.Rows != 2 || b.Bytes != int64(len(gz)) || b.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("manifesto de b inesperado: %+v", b)
	}
	var published domain.DumpManifest
	if err := json.Unmarshal(objects.data["latest/manifest.json"], &published); err != nil {
		t.Fatal(err)
	}
	if !published.GeneratedAt.Equal(now) || !reflect.DeepEqual(published.Files, manifest.Files) {
		t.Fatalf("manifesto publicado inesperado: %+v", published)
	}
	if !strings.Contains(string(objects.data["latest/LEIAME.txt"]), "acts.csv.gz") || !src.closed {
		t.Fatal("LEIAME deve descrever os arquivos e o snapshot deve ser fechado")
	}
}

func TestPublishDumpDoesNotPublishManifestWhenATableFails(t *testing.T) {
	src := &fakeSnapshot{tables: map[string]string{"a": "x\n1\n"}, order: []string{"a", "b"}, failOn: "b"}
	objects := &memObjects{}

	_, err := NewPublishDump(src, objects, nil).Execute(context.Background(), time.Now())

	if err == nil || !strings.Contains(err.Error(), "tabela b") {
		t.Fatalf("esperava erro da tabela b, veio %v", err)
	}
	if _, ok := objects.data["latest/manifest.json"]; ok || !src.closed {
		t.Fatalf("manifesto não pode ser publicado com falha: %v", objects.names)
	}
}

type failingObjects struct{}

func (failingObjects) Put(_ context.Context, _, _ string, body io.Reader) error {
	buf := make([]byte, 1)
	_, _ = body.Read(buf)
	return errors.New("upload caiu")
}

func TestPublishDumpReportsUploadFailureWithoutHanging(t *testing.T) {
	big := "x\n" + strings.Repeat("linha longa de texto\n", 100000)
	src := &fakeSnapshot{tables: map[string]string{"a": big}, order: []string{"a"}}

	_, err := NewPublishDump(src, failingObjects{}, nil).Execute(context.Background(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do upload")
	}
}
