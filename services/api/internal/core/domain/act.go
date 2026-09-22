package domain

import "time"

// ActType classifica um ato publicado.
type ActType string

const (
	ActNomeacao   ActType = "nomeacao"
	ActExoneracao ActType = "exoneracao"
	ActContrato   ActType = "contrato"
	ActAditivo    ActType = "aditivo"
	ActLicitacao  ActType = "licitacao"
	ActDispensa   ActType = "dispensa"
	ActDecreto    ActType = "decreto"
	ActLei        ActType = "lei"
	ActPortaria   ActType = "portaria"
	ActResolucao  ActType = "resolucao"
	ActDespacho   ActType = "despacho"
	ActEdital     ActType = "edital"
	ActAta        ActType = "ata"
	ActOutro      ActType = "outro"
)

var validActTypes = map[ActType]bool{
	ActNomeacao: true, ActExoneracao: true, ActContrato: true, ActAditivo: true, ActLicitacao: true,
	ActDispensa: true, ActDecreto: true, ActLei: true, ActPortaria: true, ActResolucao: true,
	ActDespacho: true, ActEdital: true, ActAta: true, ActOutro: true,
}

func (t ActType) Valid() bool { return validActTypes[t] }

// Act é um ato individual dentro de uma edição (uma portaria, um extrato...).
type Act struct {
	ID        string
	GazetteID string
	Type      ActType
	Title     string
	Body      string
	Position  int
	PageStart int
	PageEnd   int
	Entities  []Entity
}

// ActHit é um ato encontrado numa busca, com contexto da edição.
type ActHit struct {
	Act
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string
	// Snippet contém o trecho relevante; termos encontrados vêm entre ⟦ e ⟧.
	Snippet string
	// CNPJs citados no ato, como aparecem no texto.
	CNPJs []string
}

// ActFilter descreve uma busca por atos.
type ActFilter struct {
	Query  string
	Type   ActType
	From   time.Time
	To     time.Time
	Limit  int
	Offset int
}

// Normalize aplica padrões e valida o filtro.
func (f *ActFilter) Normalize() error {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Type != "" && !f.Type.Valid() {
		return ErrInvalidFilter
	}
	if !f.From.IsZero() && !f.To.IsZero() && f.To.Before(f.From) {
		return ErrInvalidFilter
	}
	return nil
}
