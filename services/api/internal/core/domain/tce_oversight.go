package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	SourceOversight     = "tce_controle"
	RecordStalledWork   = "obra_paralisada"
	tceDateLayout       = "2006-01-02"
	tceDateLength       = 10
	saoGoncaloTCEFolded = "SAO GONCALO"
)

type TCEAccount struct {
	Year        int
	Opinion     string
	Process     string
	Responsible string
}

type TCEPenalty struct {
	Condemnation string
	Process      string
	Year         int
	ValueCents   int64
	Organ        string
	Nature       string
	SessionDate  *time.Time
}

type StalledWork struct {
	Contract       string
	CNPJ           string
	Contractor     string
	Organ          string
	Function       string
	TotalCents     int64
	PaidCents      int64
	StalledAt      *time.Time
	StartedAt      *time.Time
	StalledFor     string
	Reason         string
	ContractStatus string
	Funding        string
}

type TCEOversight struct {
	Accounts  []TCEAccount
	Penalties []TCEPenalty
	Works     []StalledWork
}

func isSaoGoncalo(ente string) bool {
	return foldText(ente) == saoGoncaloTCEFolded
}

func ParseTCEAccounts(body []byte) ([]TCEAccount, error) {
	var rows []struct {
		Municipio, Indicador, Processo, Responsavel string
		Ano                                         int
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("contas de governo do TCE-RJ: %w", err)
	}
	var out []TCEAccount
	for _, r := range rows {
		if isSaoGoncalo(r.Municipio) {
			out = append(out, TCEAccount{Year: r.Ano, Opinion: squeezed(r.Indicador), Process: squeezed(r.Processo), Responsible: squeezed(r.Responsavel)})
		}
	}
	return out, nil
}

func ParseTCEPenalties(body []byte) ([]TCEPenalty, error) {
	var rows []struct {
		Processo, Condenacao, Ente, NomeOrgao, GrupoNatureza, DataSessao string
		AnoCondenacao                                                    int
		ValorPenalidade                                                  float64
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("débitos e multas do TCE-RJ: %w", err)
	}
	var out []TCEPenalty
	for _, r := range rows {
		if !isSaoGoncalo(r.Ente) {
			continue
		}
		session, err := tceDate(r.DataSessao)
		if err != nil {
			return nil, err
		}
		out = append(out, TCEPenalty{Condemnation: squeezed(r.Condenacao), Process: squeezed(r.Processo), Year: r.AnoCondenacao,
			ValueCents: reaisToCents(r.ValorPenalidade), Organ: squeezed(r.NomeOrgao), Nature: squeezed(r.GrupoNatureza), SessionDate: session})
	}
	return out, nil
}

func ParseStalledWorks(body []byte) ([]StalledWork, error) {
	var payload struct {
		Obras []struct {
			Ente, Nome, FuncaoGoverno, NumeroContrato, CNPJContratada, NomeContratada string
			DataParalisacao, DataInicioObra, TempoParalizacao, MotivoParalisacao      string
			StatusContrato, FontePrincipalRecursos                                    string
			ValorTotalContrato, ValorPagoObra                                         float64
		}
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("obras paralisadas do TCE-RJ: %w", err)
	}
	var out []StalledWork
	for _, r := range payload.Obras {
		if !isSaoGoncalo(r.Ente) {
			continue
		}
		stalled, err := tceDate(r.DataParalisacao)
		if err != nil {
			return nil, err
		}
		started, err := tceDate(r.DataInicioObra)
		if err != nil {
			return nil, err
		}
		cnpj, _ := NormalizeCNPJ(r.CNPJContratada)
		out = append(out, StalledWork{Contract: squeezed(r.NumeroContrato), CNPJ: cnpj, Contractor: squeezed(r.NomeContratada),
			Organ: squeezed(r.Nome), Function: squeezed(r.FuncaoGoverno), TotalCents: reaisToCents(r.ValorTotalContrato),
			PaidCents: reaisToCents(r.ValorPagoObra), StalledAt: stalled, StartedAt: started, StalledFor: squeezed(r.TempoParalizacao),
			Reason: squeezed(r.MotivoParalisacao), ContractStatus: squeezed(r.StatusContrato), Funding: squeezed(r.FontePrincipalRecursos)})
	}
	return out, nil
}

func tceDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if len(s) > tceDateLength {
		s = s[:tceDateLength]
	}
	t, err := time.Parse(tceDateLayout, s)
	if err != nil {
		return nil, fmt.Errorf("data %q do TCE-RJ: %w", s, err)
	}
	return &t, nil
}

func reaisToCents(v float64) int64 {
	return int64(math.Round(v * 100))
}

func TCEProcessSearch(process string) string {
	number, _, _ := strings.Cut(process, "/")
	digits, _, _ := strings.Cut(number, "-")
	if len(digits) <= 3 {
		return ""
	}
	return fmt.Sprintf("%q", digits[:len(digits)-3]+"."+digits[len(digits)-3:])
}

type PenaltyProcess struct {
	Process       string
	Search        string
	Organs        []string
	Natures       []string
	Condemnations []TCEPenalty
	TotalCents    int64
	LastSession   *time.Time
}

func GroupPenalties(penalties []TCEPenalty) []PenaltyProcess {
	index := map[string]int{}
	var out []PenaltyProcess
	for _, p := range penalties {
		i, ok := index[p.Process]
		if !ok {
			i = len(out)
			index[p.Process] = i
			out = append(out, PenaltyProcess{Process: p.Process, Search: TCEProcessSearch(p.Process)})
		}
		g := &out[i]
		g.Condemnations = append(g.Condemnations, p)
		g.TotalCents += p.ValueCents
		g.Organs = appendUnique(g.Organs, p.Organ)
		g.Natures = appendUnique(g.Natures, p.Nature)
		if p.SessionDate != nil && (g.LastSession == nil || p.SessionDate.After(*g.LastSession)) {
			g.LastSession = p.SessionDate
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].LastSession, out[j].LastSession
		if a == nil || b == nil {
			return b == nil && a != nil
		}
		return a.After(*b)
	})
	return out
}

func appendUnique(list []string, s string) []string {
	if s == "" || slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}
