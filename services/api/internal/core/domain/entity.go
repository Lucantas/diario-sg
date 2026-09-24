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
	Certainty   Certainty
	Sources     int
	TotalActs   int
	CountByType map[ActType]int
	TotalCents  int64
	Acts        []ActHit

	ByProcess          []ProcessSummary
	ActsWithoutProcess int
}

type ProcessSummary struct {
	Key           string
	Acts          int
	MaxValueCents int64
	First         time.Time
	Last          time.Time
}

func (r EntityReport) SumOfProcessMaxCents() int64 {
	var sum int64
	for _, p := range r.ByProcess {
		sum += p.MaxValueCents
	}
	return sum
}
