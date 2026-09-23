package events

import "time"

const (
	TypeGazetteFetched = "gazette.fetched.v1"
	TypeGazetteIndexed = "gazette.indexed.v1"

	AttrType = "type"
)

type GazetteFetched struct {
	EditionNumber  string    `json:"edition_number"`
	PublishedAt    time.Time `json:"published_at"`
	SourceURL      string    `json:"source_url"`
	StoragePath    string    `json:"storage_path"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type GazetteIndexed struct {
	GazetteID string    `json:"gazette_id"`
	ActsCount int       `json:"acts_count"`
	IndexedAt time.Time `json:"indexed_at"`
}
