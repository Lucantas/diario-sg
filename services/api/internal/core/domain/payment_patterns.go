package domain

import (
	"sort"
	"strconv"
	"strings"
)

const (
	MinPaidWithoutPublicationCents = 10_000_000
	MinUnpaidContractCents         = 10_000_000
	MinPaidAboveAnnouncedCents     = 100_000_000
	PaidAboveAnnouncedRatio        = 2
	unpaidGraceYears               = 1
)

type CreditorPaid struct {
	CNPJ      string
	Years     []int
	Units     []string
	PaidCents int64
}

type PaidWithoutPublication struct {
	Profile SupplierProfile
	Paid    CreditorPaid
}

type UnpaidContract struct {
	Profile  SupplierProfile
	Contract SupplierContract
}

type PaidAboveAnnounced struct {
	Profile        SupplierProfile
	Paid           CreditorPaid
	AnnouncedCents int64
	Contracts      []SupplierContract
}

func FindPaidWithoutPublication(paid []CreditorPaid, cited map[string]bool, profiles map[string]SupplierProfile) []PaidWithoutPublication {
	var out []PaidWithoutPublication
	for _, c := range paid {
		if cited[c.CNPJ] || c.PaidCents < MinPaidWithoutPublicationCents || isPublicCreditor(c.CNPJ, profiles) {
			continue
		}
		out = append(out, PaidWithoutPublication{Profile: profileOrCNPJ(profiles, c.CNPJ), Paid: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Paid.PaidCents > out[j].Paid.PaidCents })
	return out
}

func FindUnpaidContracts(contracts []SupplierContract, paid []CreditorPaid, coverage *PaymentCoverage, profiles map[string]SupplierProfile) []UnpaidContract {
	if coverage == nil {
		return nil
	}
	paidYears := map[string]map[int]bool{}
	for _, c := range paid {
		years := map[int]bool{}
		for _, y := range c.Years {
			years[y] = true
		}
		paidYears[c.CNPJ] = years
	}
	var out []UnpaidContract
	for _, c := range contracts {
		year := c.First.Year()
		if c.ContractedCents < MinUnpaidContractCents || year < coverage.FromYear || year+unpaidGraceYears > coverage.ToYear {
			continue
		}
		if _, public := PublicBody(c.CNPJ); public {
			continue
		}
		years := paidYears[c.CNPJ]
		if years[year] || years[year+unpaidGraceYears] {
			continue
		}
		out = append(out, UnpaidContract{Profile: profileOrCNPJ(profiles, c.CNPJ), Contract: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Contract.ContractedCents > out[j].Contract.ContractedCents })
	return out
}

func FindPaidAboveAnnounced(contracts []SupplierContract, amended map[string]int64, paid []CreditorPaid, profiles map[string]SupplierProfile) []PaidAboveAnnounced {
	announced := map[string]int64{}
	byCNPJ := map[string][]SupplierContract{}
	for _, c := range contracts {
		announced[c.CNPJ] += c.ContractedCents + c.RegisteredCents
		byCNPJ[c.CNPJ] = append(byCNPJ[c.CNPJ], c)
	}
	var out []PaidAboveAnnounced
	for _, p := range paid {
		list, ok := byCNPJ[p.CNPJ]
		if !ok {
			continue
		}
		total := announced[p.CNPJ] + amended[p.CNPJ]
		if p.PaidCents-total < MinPaidAboveAnnouncedCents || p.PaidCents < PaidAboveAnnouncedRatio*total || isPublicCreditor(p.CNPJ, profiles) {
			continue
		}
		out = append(out, PaidAboveAnnounced{Profile: profileOrCNPJ(profiles, p.CNPJ), Paid: p, AnnouncedCents: total, Contracts: list})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Paid.PaidCents-out[i].AnnouncedCents > out[j].Paid.PaidCents-out[j].AnnouncedCents
	})
	return out
}

func AmendedBySupplier(acts []PanelAct) map[string]int64 {
	out := map[string]int64{}
	for _, c := range panelContracts(acts) {
		out[c.CNPJ] += c.amendedIn(0)
	}
	return out
}

var publicLawNatures = []string{"orgao publico", "autarquia", "fundo publico", "fundacao publica", "municipio", "estado ou", "uniao",
	"consorcio publico", "servico social autonomo"}

func isPublicCreditor(cnpj string, profiles map[string]SupplierProfile) bool {
	if _, public := PublicBody(cnpj); public {
		return true
	}
	nature := foldAccents(profiles[cnpj].LegalNature)
	for _, prefix := range publicLawNatures {
		if strings.HasPrefix(nature, prefix) {
			return true
		}
	}
	return false
}

func profileOrCNPJ(profiles map[string]SupplierProfile, cnpj string) SupplierProfile {
	if p, ok := profiles[cnpj]; ok {
		return p
	}
	return SupplierProfile{CNPJ: cnpj}
}

func paymentYearsLabel(years []int) string {
	if len(years) == 0 {
		return ""
	}
	sorted := append([]int(nil), years...)
	sort.Ints(sorted)
	first, last := strconv.Itoa(sorted[0]), strconv.Itoa(sorted[len(sorted)-1])
	if first == last {
		return first
	}
	return first + " a " + last
}

func AttributeByName(acts []PanelAct, creditors []CreditorPaid, profiles map[string]SupplierProfile) []PanelAct {
	byName := map[string]string{}
	ambiguous := map[string]bool{}
	for _, c := range creditors {
		key := SupplierNameKey(profiles[c.CNPJ].Name)
		if key == "" {
			continue
		}
		if prev, ok := byName[key]; ok && CNPJBase(prev) != CNPJBase(c.CNPJ) {
			ambiguous[key] = true
		}
		byName[key] = c.CNPJ
	}
	var out []PanelAct
	for _, a := range acts {
		key := SupplierNameKey(SupplierNameOf(a.Head))
		if cnpj, ok := byName[key]; ok && key != "" && !ambiguous[key] {
			a.CNPJ = cnpj
			out = append(out, a)
		}
	}
	return out
}
