package domain

import "regexp"

type Phase string

const (
	PhaseLicitacao         Phase = "licitacao"
	PhaseHomologacao       Phase = "homologacao"
	PhaseAtaRegistroPrecos Phase = "ata_registro_precos"
	PhaseDispensa          Phase = "dispensa"
	PhaseContrato          Phase = "contrato"
	PhaseFiscal            Phase = "fiscal"
	PhaseAditivo           Phase = "aditivo"
	PhaseAjusteContas      Phase = "ajuste_contas"
	PhaseRescisao          Phase = "rescisao"
	PhaseOutro             Phase = "outro"
)

var Phases = []Phase{
	PhaseLicitacao, PhaseHomologacao, PhaseAtaRegistroPrecos, PhaseDispensa, PhaseContrato,
	PhaseFiscal, PhaseAditivo, PhaseAjusteContas, PhaseRescisao, PhaseOutro,
}

var phaseRules = []struct {
	phase Phase
	types []ActType
	title *regexp.Regexp
}{
	{PhaseRescisao, nil, regexp.MustCompile(`(?i)rescis|distrato`)},
	{PhaseAditivo, []ActType{ActAditivo}, regexp.MustCompile(`(?i)aditi?vo|apostil`)},
	{PhaseAjusteContas, nil, regexp.MustCompile(`(?i)ajuste\s+de\s+contas|reconhecimento\s+de\s+d[íi]vida`)},
	{PhaseFiscal, nil, regexp.MustCompile(`(?i)\bfisca(?:l|is)\b`)},
	{PhaseHomologacao, nil, regexp.MustCompile(`(?i)homolog|adjudic`)},
	{PhaseAtaRegistroPrecos, nil, regexp.MustCompile(`(?i)registro\s+de\s+pre[çc]o`)},
	{PhaseDispensa, []ActType{ActDispensa}, regexp.MustCompile(`(?i)dispensa|inexigibilidade|ratific`)},
	{PhaseContrato, []ActType{ActContrato}, nil},
	{PhaseLicitacao, []ActType{ActLicitacao, ActEdital, ActAta}, nil},
}

func PhaseOf(t ActType, title string) Phase {
	for _, r := range phaseRules {
		if r.title != nil && r.title.MatchString(title) {
			return r.phase
		}
		for _, rt := range r.types {
			if rt == t {
				return r.phase
			}
		}
	}
	return PhaseOutro
}
