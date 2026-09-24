package domain

import "time"

type EntityKind string

const (
	EntityCNPJ     EntityKind = "cnpj"
	EntityValor    EntityKind = "valor"
	EntityContrato EntityKind = "contrato"
	EntityProcesso EntityKind = "processo"
)

type Entity struct {
	Kind       EntityKind
	Value      string
	Normalized string
}

type CompanyReport struct {
	CNPJ        string
	Acts        []ActHit
	TotalCents  int64
	CountByType map[ActType]int
}

type MonthCount struct {
	Month time.Time
	Count int
}

type EntityReport struct {
	Kind        EntityKind
	Key         string
	Label       string
	Certainty   Certainty
	Sources     int
	TotalActs   int
	CountByType map[ActType]int
	TotalCents  int64
	Acts        []ActHit

	ByProcess          []ProcessSummary
	ProcessCount       int
	ProcessSumCents    int64
	ActsWithoutProcess int

	Organs          []OrganCount
	CountByPhase    map[Phase]int
	Related         []RelatedEntity
	TypeTitleCounts []TypeTitleCount
}

type OrganCount struct {
	Organ string
	Acts  int
}

type RelatedEntity struct {
	Kind  EntityKind
	Key   string
	Label string
	Acts  int
}

type EntityMention struct {
	Kind  EntityKind
	Key   string
	Label string
}

type TypeTitleCount struct {
	Type  ActType
	Title string
	Acts  int
}

type ProcessSummary struct {
	Key           string
	Acts          int
	MaxValueCents int64
	First         time.Time
	Last          time.Time
}
