package domain

import "strings"

const alertHitsPerEdition = 20

var actTypeNames = map[ActType]string{
	ActNomeacao: "Nomeação", ActExoneracao: "Exoneração", ActContrato: "Contrato",
	ActAditivo: "Aditivo", ActLicitacao: "Licitação", ActDispensa: "Sem licitação",
	ActDecreto: "Decreto", ActLei: "Lei", ActPortaria: "Portaria", ActResolucao: "Resolução",
	ActDespacho: "Despacho", ActEdital: "Edital", ActAta: "Ata", ActCorrigenda: "Corrigenda",
	ActPrestacaoContas: "Prestação de contas", ActLicencaAmbiental: "Licença ambiental", ActOutro: "Outro",
}

func ActTypeName(t ActType) string { return actTypeNames[t] }

type AlertFilter struct {
	Source string
	Type   ActType
	Organ  string
	Theme  string
}

func (f AlertFilter) IsZero() bool { return f == AlertFilter{} }

func (f AlertFilter) normalized() (AlertFilter, error) {
	af := ActFilter{Source: f.Source, Type: f.Type, Organ: f.Organ, Theme: f.Theme}
	if err := af.Normalize(); err != nil {
		return AlertFilter{}, err
	}
	if af.Source == SourceDiarioCamara && af.Organ != "" {
		return AlertFilter{}, ErrInvalidFilter
	}
	return AlertFilter{Source: af.Source, Type: af.Type, Organ: af.Organ, Theme: af.Theme}, nil
}

func (f AlertFilter) ActFilter(query, gazetteID string) ActFilter {
	return ActFilter{Query: query, Source: f.Source, Type: f.Type, Organ: f.Organ, Theme: f.Theme,
		GazetteID: gazetteID, Limit: alertHitsPerEdition}
}

func (f AlertFilter) Description() string {
	var parts []string
	if f.Type != "" {
		parts = append(parts, strings.ToLower(ActTypeName(f.Type)))
	}
	if f.Organ != "" {
		parts = append(parts, f.Organ)
	}
	if theme, err := ParseTheme(f.Theme); err == nil && !theme.IsZero() {
		parts = append(parts, strings.ToLower(theme.Name))
	}
	if f.Source != "" {
		parts = append(parts, "Diário da "+SourceLabel(f.Source))
	}
	return strings.Join(parts, " · ")
}
