package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

//go:embed leiame.txt
var leiame string

const dumpPrefix = "latest/"

type PublishDump struct {
	source  ports.DumpSource
	objects ports.ObjectWriter
}

func NewPublishDump(s ports.DumpSource, o ports.ObjectWriter) *PublishDump {
	return &PublishDump{source: s, objects: o}
}

func (uc *PublishDump) Execute(ctx context.Context, now time.Time) (domain.DumpManifest, error) {
	snap, err := uc.source.Snapshot(ctx)
	if err != nil {
		return domain.DumpManifest{}, err
	}
	defer snap.Close()
	manifest := domain.DumpManifest{GeneratedAt: now}
	for _, table := range snap.Tables() {
		file, err := uc.publishTable(ctx, snap, table)
		if err != nil {
			return domain.DumpManifest{}, fmt.Errorf("tabela %s: %w", table, err)
		}
		manifest.Files = append(manifest.Files, file)
	}
	if err := uc.objects.Put(ctx, dumpPrefix+"LEIAME.txt", "text/plain; charset=utf-8", strings.NewReader(leiame)); err != nil {
		return domain.DumpManifest{}, err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return domain.DumpManifest{}, err
	}
	return manifest, uc.objects.Put(ctx, dumpPrefix+"manifest.json", "application/json", bytes.NewReader(body))
}

type tableResult struct {
	rows int
	err  error
}

type byteCounter struct{ n int64 }

func (c *byteCounter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

func (uc *PublishDump) publishTable(ctx context.Context, snap ports.DumpSnapshot, table string) (domain.DumpFile, error) {
	name := table + ".csv.gz"
	pr, pw := io.Pipe()
	hash := sha256.New()
	counter := &byteCounter{}
	done := make(chan tableResult, 1)
	go func() {
		gz := gzip.NewWriter(io.MultiWriter(pw, hash, counter))
		rows, err := snap.WriteTable(ctx, table, gz)
		if cerr := gz.Close(); err == nil {
			err = cerr
		}
		pw.CloseWithError(err)
		done <- tableResult{rows: rows, err: err}
	}()
	putErr := uc.objects.Put(ctx, dumpPrefix+name, "application/gzip", pr)
	pr.CloseWithError(putErr)
	res := <-done
	if res.err != nil {
		return domain.DumpFile{}, res.err
	}
	if putErr != nil {
		return domain.DumpFile{}, putErr
	}
	return domain.DumpFile{Name: name, Rows: res.rows, Bytes: counter.n, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}
