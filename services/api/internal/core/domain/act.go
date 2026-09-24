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

	Modality       Modality
	MainValueCents int64
}

type ActHit struct {
	Act
	Source        string
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string
	Checksum      string
	TitleOnly     bool
	Signatures    int

	Snippet string

	CNPJs       []string
	ValuesCents []int64
}

const (
	ExportLimit = 10000
	FeedLimit   = 50
)

type ActFilter struct {
	Query    string
	Source   string
	Type     ActType
	Organ    string
	From     time.Time
	To       time.Time
	MinCents int64
	MaxCents int64
	Modality Modality

	MainMinCents int64
	MainMaxCents int64
	Recent       bool
	Limit        int
	Offset       int
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
	if f.Source != "" && !ValidSource(f.Source) {
		return ErrInvalidFilter
	}
	organ, ok := NormalizeOrgan(f.Organ)
	if !ok {
		return ErrInvalidFilter
	}
	f.Organ = organ
	if f.MinCents < 0 || f.MaxCents < 0 || (f.MaxCents > 0 && f.MinCents > f.MaxCents) {
		return ErrInvalidFilter
	}
	if f.MainMinCents < 0 || f.MainMaxCents < 0 || (f.MainMaxCents > 0 && f.MainMinCents > f.MainMaxCents) {
		return ErrInvalidFilter
	}
	if f.Modality != "" && !f.Modality.Valid() {
		return ErrInvalidFilter
	}
	f.Query = TranslateOperators(f.Query)
	if !f.From.IsZero() && !f.To.IsZero() && f.To.Before(f.From) {
		return ErrInvalidFilter
	}
	return nil
}
