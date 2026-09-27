package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func archiveJSON(ctx context.Context, raw ports.ObjectWriter, path string, body []byte, rows int) error {
	return archiveRaw(ctx, raw, path, "json", body, rows)
}

func archiveRaw(ctx context.Context, raw ports.ObjectWriter, path, ext string, body []byte, rows int) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := raw.Put(ctx, path+"."+ext+".gz", "application/gzip", &buf); err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	manifest, err := json.MarshalIndent(archivedFile{SourceSHA256: hex.EncodeToString(sum[:]), Rows: rows}, "", "  ")
	if err != nil {
		return err
	}
	return raw.Put(ctx, path+".manifest.json", "application/json", bytes.NewReader(manifest))
}
