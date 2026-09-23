package events

import "time"

const (
	TypeGazetteFetched = "gazette.fetched.v1"
	TypeGazetteIndexed = "gazette.indexed.v1"
	TypeFetchCompleted = "fetch.completed.v1"

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

type FetchCompleted struct {
	RunID         string    `json:"run_id"`
	Source        string    `json:"source"`
	RequestedFrom string    `json:"requested_from"`
	RequestedTo   string    `json:"requested_to"`
	Found         int       `json:"found"`
	Stored        int       `json:"stored"`
	Skipped       int       `json:"skipped"`
	Failed        int       `json:"failed"`
	Error         string    `json:"error"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}
