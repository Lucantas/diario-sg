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
