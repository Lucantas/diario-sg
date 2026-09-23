package domain

import "time"

type Gazette struct {
	ID            string
	Source        string
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string
	StoragePath   string
	Checksum      string
	IndexedAt     time.Time
}
