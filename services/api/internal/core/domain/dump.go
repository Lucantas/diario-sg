package domain

import "time"

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
