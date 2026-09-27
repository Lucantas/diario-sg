package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	PatternPNCPWithoutExtract  PatternID = "pncp_sem_extrato"
	MinPNCPWithoutExtractCents           = 10_000_000
	PNCPExtractGraceDays                 = 30
)

type PNCPWithoutExtract struct {
	Profile  SupplierProfile
	Contract PNCPContract
}

type PNCPCitations struct {
	CNPJs, Processes, Names map[string]bool
}

func PNCPCitedNames(acts []PanelAct) map[string]bool {
	out := map[string]bool{}
	for _, a := range acts {
		if key := SupplierNameKey(SupplierNameOf(a.Head)); key != "" {
			out[key] = true
		}
	}
	return out
}

func FindPNCPWithoutExtract(contracts []PNCPContract, cited PNCPCitations, lastDiario time.Time, profiles map[string]SupplierProfile) []PNCPWithoutExtract {
	deadline := lastDiario.AddDate(0, 0, -PNCPExtractGraceDays)
	var out []PNCPWithoutExtract
	for _, c := range contracts {
		day := c.day()
		if day == nil || day.After(deadline) || c.ValueCents < MinPNCPWithoutExtractCents || cited.includes(c, profiles) {
			continue
		}
		p := profileOrCNPJ(profiles, c.SupplierCNPJ)
		if p.Name == "" {
			p.Name = c.SupplierName
		}
		out = append(out, PNCPWithoutExtract{Profile: p, Contract: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Contract.ValueCents > out[j].Contract.ValueCents })
	return out
}

func (cited PNCPCitations) includes(c PNCPContract, profiles map[string]SupplierProfile) bool {
	if cited.CNPJs[c.SupplierCNPJ] {
		return true
	}
	if key := c.ProcessKey(); key != "" && cited.Processes[key] {
		return true
	}
	for _, name := range []string{c.SupplierName, profiles[c.SupplierCNPJ].Name} {
		if key := SupplierNameKey(name); key != "" && cited.Names[key] {
			return true
		}
	}
	return false
}

func (c PNCPContract) day() *time.Time {
	if c.SignedAt != nil {
		return c.SignedAt
	}
	return c.PublishedAt
}

func pncpPatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternPNCPWithoutExtract: {
			ID:    PatternPNCPWithoutExtract,
			Title: "Contrato no PNCP sem nenhuma citação no Diário",
			Rule: fmt.Sprintf("Contrato de pelo menos %s que a Prefeitura, um fundo, uma fundação ou a Câmara registrou no Portal "+
				"Nacional de Contratações Públicas, assinado pelo menos %d dias antes da última edição do Diário da Prefeitura, sem "+
				"que nenhum ato do Diário da Prefeitura ou da Câmara cite o CNPJ da empresa ou o número do processo, e sem extrato "+
				"com valor e sem CNPJ cujo fornecedor lido do texto tenha o nome da empresa.",
				FormatBRL(MinPNCPWithoutExtractCents), PNCPExtractGraceDays),
			Caveat: "O extrato pode ter sido publicado com o nome da empresa escrito de outro jeito ou com o processo em outro formato: " +
				"procure o nome e o número do contrato na busca. O município registra no PNCP só parte dos contratos, quase todos de " +
				"2024 em diante.",
		},
	}
}

func PNCPWithoutExtractFinding(p PNCPWithoutExtract) Finding {
	c := p.Contract
	signed := "sem data"
	if d := c.day(); d != nil {
		signed = d.Format("02/01/2006")
	}
	detail := fmt.Sprintf("%s %s, %s. Objeto: %s", c.Kind, c.Number, orEmptyLabel(c.UnitName), c.Object)
	if c.Process != "" {
		detail += ". Processo " + c.Process
	}
	return Finding{
		Title:    fmt.Sprintf("%s: contrato de %s assinado em %s", supplierLabel(p.Profile), FormatBRL(c.ValueCents), signed),
		Detail:   detail + ".",
		Entities: pncpMentions(c),
		Link:     &FindingLink{Label: "Ver o contrato no PNCP", URL: c.URL()},
	}
}

func pncpMentions(c PNCPContract) []EntityMention {
	out := cnpjMentions(c.SupplierCNPJ)
	if key := c.ProcessKey(); key != "" {
		out = append(out, EntityMention{Kind: EntityProcesso, Key: key, Label: c.Process})
	}
	return out
}
