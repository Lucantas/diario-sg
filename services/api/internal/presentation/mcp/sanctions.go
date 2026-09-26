package mcp

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type sanctionDTO struct {
	Cadastro         string `json:"cadastro"`
	Codigo           string `json:"codigo"`
	CNPJ             string `json:"cnpj"`
	Categoria        string `json:"categoria"`
	Inicio           string `json:"inicio,omitempty"`
	Fim              string `json:"fim,omitempty"`
	Processo         string `json:"processo,omitempty"`
	OrgaoSancionador string `json:"orgao_sancionador"`
	UF               string `json:"uf,omitempty"`
	Esfera           string `json:"esfera"`
	Abrangencia      string `json:"abrangencia"`
	Fundamentacao    string `json:"fundamentacao"`
	MultaCentavos    *int64 `json:"multa_centavos,omitempty"`
	VistaPrimeiraVez string `json:"vista_pela_primeira_vez"`
	VistaUltimaVez   string `json:"vista_pela_ultima_vez"`
	Estado           string `json:"estado"`
}

func sanctionsOf(kind domain.EntityKind, sanctions []domain.Sanction, listedOn map[string]time.Time, today time.Time) []sanctionDTO {
	if kind != domain.EntityCNPJ || len(listedOn) == 0 {
		return nil
	}
	out := make([]sanctionDTO, len(sanctions))
	for i, s := range sanctions {
		out[i] = sanctionDTO{Cadastro: s.Register, Codigo: s.Code, CNPJ: s.CNPJ, Categoria: s.Category,
			Inicio: optionalDate(s.StartsAt), Fim: optionalDate(s.EndsAt), Processo: s.Process, OrgaoSancionador: s.Organ,
			UF: s.OrganUF, Esfera: s.Sphere, Abrangencia: s.Scope, Fundamentacao: s.LegalBasis, MultaCentavos: s.FineCents,
			VistaPrimeiraVez: s.FirstSeen.Format(time.DateOnly), VistaUltimaVez: s.LastSeen.Format(time.DateOnly),
			Estado: string(s.State(listedOn[s.Register], today))}
	}
	return out
}

func sanctionsConsulted(kind domain.EntityKind, listedOn map[string]time.Time) map[string]string {
	if kind != domain.EntityCNPJ || len(listedOn) == 0 {
		return nil
	}
	out := make(map[string]string, len(listedOn))
	for register, day := range listedOn {
		out[register] = day.Format(time.DateOnly)
	}
	return out
}
