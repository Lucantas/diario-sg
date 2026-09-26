package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type csvArchive struct {
	buf  bytes.Buffer
	gz   *gzip.Writer
	w    *csv.Writer
	rows int
}

func newCSVArchive() *csvArchive {
	a := &csvArchive{}
	a.gz = gzip.NewWriter(&a.buf)
	a.w = csv.NewWriter(a.gz)
	a.w.Comma = ';'
	return a
}

func (a *csvArchive) write(header, row []string) error {
	if a.rows == 0 {
		if err := a.w.Write(header); err != nil {
			return err
		}
	}
	a.rows++
	return a.w.Write(row)
}

func (a *csvArchive) put(ctx context.Context, raw ports.ObjectWriter, prefix, sha string) error {
	a.w.Flush()
	if err := a.w.Error(); err != nil {
		return err
	}
	if err := a.gz.Close(); err != nil {
		return err
	}
	if err := raw.Put(ctx, prefix+".csv.gz", "application/gzip", &a.buf); err != nil {
		return err
	}
	manifest, err := json.MarshalIndent(archivedFile{SourceSHA256: sha, Rows: a.rows}, "", "  ")
	if err != nil {
		return err
	}
	return raw.Put(ctx, prefix+".manifest.json", "application/json", bytes.NewReader(manifest))
}
