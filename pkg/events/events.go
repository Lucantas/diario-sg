// Package events define os contratos de mensagens trocadas entre serviços.
// A fonte da verdade são os JSON Schemas em /contracts/events; estes structs
// são a representação em Go. Serviços em outras linguagens usam os schemas.
package events

import "time"

const (
	TypeGazetteFetched = "gazette.fetched.v1"
	TypeGazetteIndexed = "gazette.indexed.v1"

	// AttrType é o atributo da mensagem Pub/Sub com o tipo do evento.
	AttrType = "type"
)

// GazetteFetched é publicado pelo scraper quando uma edição nova do Diário
// Oficial foi baixada e armazenada.
type GazetteFetched struct {
	EditionNumber  string    `json:"edition_number"`
	PublishedAt    time.Time `json:"published_at"`
	SourceURL      string    `json:"source_url"`
	StoragePath    string    `json:"storage_path"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// GazetteIndexed é publicado pelo worker quando os atos de uma edição foram
// extraídos e indexados.
type GazetteIndexed struct {
	GazetteID string    `json:"gazette_id"`
	ActsCount int       `json:"acts_count"`
	IndexedAt time.Time `json:"indexed_at"`
}
