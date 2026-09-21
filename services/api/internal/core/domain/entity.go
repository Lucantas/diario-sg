package domain

import "time"

// EntityKind é o tipo de campo extraído do texto de um ato.
type EntityKind string

const (
	EntityCNPJ     EntityKind = "cnpj"
	EntityValor    EntityKind = "valor"
	EntityContrato EntityKind = "contrato"
	EntityProcesso EntityKind = "processo"
)

// Entity é um campo encontrado no corpo de um ato. Value é o texto como
// apareceu; Normalized é a forma canônica para busca e agregação
// (CNPJ só dígitos, valor em centavos, números sem espaços).
type Entity struct {
	Kind       EntityKind
	Value      string
	Normalized string
}

// CompanyReport reúne os atos em que um CNPJ apareceu.
type CompanyReport struct {
	CNPJ        string
	Acts        []ActHit
	TotalCents  int64
	CountByType map[ActType]int
}

// MonthCount é a contagem de atos publicados num mês.
type MonthCount struct {
	Month time.Time
	Count int
}
