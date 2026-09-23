package domain

import "time"

type ActType string

const (
	ActNomeacao        ActType = "nomeacao"
	ActExoneracao      ActType = "exoneracao"
	ActContrato        ActType = "contrato"
	ActAditivo         ActType = "aditivo"
	ActLicitacao       ActType = "licitacao"
	ActDispensa        ActType = "dispensa"
	ActDecreto         ActType = "decreto"
	ActLei             ActType = "lei"
	ActPortaria        ActType = "portaria"
	ActResolucao       ActType = "resolucao"
	ActDespacho        ActType = "despacho"
	ActEdital          ActType = "edital"
	ActAta             ActType = "ata"
	ActCorrigenda      ActType = "corrigenda"
	ActPrestacaoContas ActType = "prestacao_contas"
	ActOutro           ActType = "outro"
)

var validActTypes = map[ActType]bool{
	ActNomeacao: true, ActExoneracao: true, ActContrato: true, ActAditivo: true, ActLicitacao: true,
	ActDispensa: true, ActDecreto: true, ActLei: true, ActPortaria: true, ActResolucao: true,
	ActDespacho: true, ActEdital: true, ActAta: true, ActCorrigenda: true, ActPrestacaoContas: true,
	ActOutro: true,
}

func (t ActType) Valid() bool { return validActTypes[t] }

type Act struct {
	ID        string
	GazetteID string
	Type      ActType
	Title     string
	Body      string
	Position  int
	PageStart int
	PageEnd   int
	Organ     string
	Entities  []Entity
}

type ActHit struct {
	Act
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string

	Snippet string

	CNPJs []string
}

type ActFilter struct {
	Query  string
	Type   ActType
	From   time.Time
	To     time.Time
	Limit  int
	Offset int
}

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
