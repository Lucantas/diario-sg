package domain

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"time"
)

type DumpManifest struct {
	GeneratedAt time.Time  `json:"generated_at"`
	Files       []DumpFile `json:"files"`
}

type DumpFile struct {
	Name   string `json:"name"`
	Rows   int    `json:"rows"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type DumpReport struct {
	Manifest      DumpManifest
	Incremental   bool
	Gazettes      int
	FetchedAgain  int
	FallbackCause string
}

type DumpRecord struct {
	GazetteID string
	Fields    []string
}

type GazetteFingerprint struct {
	GazetteID string
	Hash      string
}

const (
	fingerprintFieldSep = "\x1f"
	fingerprintRowSep   = "\x1e"
	fingerprintTableSep = "\x1d"
)

func FingerprintOf(acts, entities [][]string) string {
	sum := md5.Sum([]byte(joinRows(acts) + fingerprintTableSep + joinRows(entities)))
	return hex.EncodeToString(sum[:])
}

func joinRows(rows [][]string) string {
	joined := make([]string, len(rows))
	for i, r := range rows {
		joined[i] = strings.Join(r, fingerprintFieldSep)
	}
	return strings.Join(joined, fingerprintRowSep)
}
