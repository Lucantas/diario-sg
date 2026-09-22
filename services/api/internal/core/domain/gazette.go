package domain

import "time"

// Gazette é uma edição do Diário Oficial já indexada.
type Gazette struct {
	ID            string
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string
	StoragePath   string
	Checksum      string
	IndexedAt     time.Time
}
