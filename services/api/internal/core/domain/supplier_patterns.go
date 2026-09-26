package domain

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	NewCompanyDays           = 180
	CapitalShareBP           = 1000
	MinUndercapitalizedCents = 10_000_000
	basisPointsTotal         = 10000
	minGroupCompanies        = 2
	saoGoncaloOrganFragment  = "SAO GONCALO"
)

type SupplierContract struct {
	CNPJ            string
	First           time.Time
	ContractedCents int64
	RegisteredCents int64
	ActID           string
	Organs          []string
}

type PartnerKey struct {
	Kind     PartnerKind
	Name     string
	Document string
}

type SupplierProfile struct {
	CNPJ         string
	Name         string
	Headquarters bool
	OpenedAt     *time.Time
	CapitalCents int64
	LegalNature  string
	Address      string
	Partners     []PartnerKey
}

type NewCompanyContract struct {
	Profile  SupplierProfile
	Contract SupplierContract
	Days     int
}

type UndercapitalizedContract struct {
	Profile  SupplierProfile
	Contract SupplierContract
}

type SharedSupplierGroup struct {
	Partners  []PartnerKey
	Address   string
	Suppliers []SupplierProfile
	Contracts []SupplierContract
}

type SanctionedContract struct {
	Profile   SupplierProfile
	Contract  SupplierContract
	Sanctions []Sanction
}

func SupplierContracts(acts []PanelAct) []SupplierContract {
	var out []SupplierContract
	for _, c := range panelContracts(acts) {
		if !c.hasContractValue() {
			continue
		}
		out = append(out, SupplierContract{CNPJ: c.CNPJ, First: c.First, ContractedCents: c.ContractedCents,
			RegisteredCents: c.RegisteredCents, ActID: c.LargestActID, Organs: c.Organs})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].First.Equal(out[j].First) {
			return out[i].First.Before(out[j].First)
		}
		return out[i].ActID < out[j].ActID
	})
	return out
}

func FindNewCompanyContracts(contracts []SupplierContract, profiles map[string]SupplierProfile) []NewCompanyContract {
	seen := map[string]bool{}
	var out []NewCompanyContract
	for _, c := range contracts {
		if seen[c.CNPJ] {
			continue
		}
		seen[c.CNPJ] = true
		p, ok := profiles[c.CNPJ]
		if !ok || !p.Headquarters || p.OpenedAt == nil {
			continue
		}
		days := int(c.First.Sub(*p.OpenedAt).Hours() / 24)
		if days >= 0 && days < NewCompanyDays {
			out = append(out, NewCompanyContract{Profile: p, Contract: c, Days: days})
		}
	}
	return out
}

func FindUndercapitalizedContracts(contracts []SupplierContract, profiles map[string]SupplierProfile) []UndercapitalizedContract {
	largest := map[string]SupplierContract{}
	for _, c := range contracts {
		if c.ContractedCents > largest[c.CNPJ].ContractedCents {
			largest[c.CNPJ] = c
		}
	}
	var out []UndercapitalizedContract
	for cnpj, c := range largest {
		p, ok := profiles[cnpj]
		if !ok || p.CapitalCents <= 0 || c.ContractedCents < MinUndercapitalizedCents || !HasShareCapital(p.LegalNature) {
			continue
		}
		if p.CapitalCents*basisPointsTotal < c.ContractedCents*CapitalShareBP {
			out = append(out, UndercapitalizedContract{Profile: p, Contract: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ri := float64(out[i].Contract.ContractedCents) / float64(out[i].Profile.CapitalCents)
		rj := float64(out[j].Contract.ContractedCents) / float64(out[j].Profile.CapitalCents)
		if ri != rj {
			return ri > rj
		}
		return out[i].Profile.CNPJ < out[j].Profile.CNPJ
	})
	return out
}

var shareCapitalNatures = []string{"sociedade", "empresa individual", "empresario"}

func HasShareCapital(legalNature string) bool {
	folded := foldAccents(legalNature)
	for _, prefix := range shareCapitalNatures {
		if strings.HasPrefix(folded, prefix) {
			return true
		}
	}
	return false
}

func FindSharedPartners(contracts []SupplierContract, profiles map[string]SupplierProfile) []SharedSupplierGroup {
	groups := sharedGroups(contracts, profiles, func(p SupplierProfile) []string {
		keys := make([]string, 0, len(p.Partners))
		for _, k := range p.Partners {
			if k.Name != "" {
				keys = append(keys, partnerGroupKey(k))
			}
		}
		return keys
	}, func(g *SharedSupplierGroup, key string) {
		for _, k := range g.Suppliers[0].Partners {
			if partnerGroupKey(k) == key {
				g.Partners = append(g.Partners, k)
				return
			}
		}
	})
	return mergeSameSuppliers(groups)
}

func mergeSameSuppliers(groups []SharedSupplierGroup) []SharedSupplierGroup {
	index := map[string]int{}
	var out []SharedSupplierGroup
	for _, g := range groups {
		cnpjs := make([]string, len(g.Suppliers))
		for i, s := range g.Suppliers {
			cnpjs[i] = s.CNPJ
		}
		key := strings.Join(cnpjs, ",")
		if i, ok := index[key]; ok {
			out[i].Partners = append(out[i].Partners, g.Partners...)
			continue
		}
		index[key] = len(out)
		out = append(out, g)
	}
	for i := range out {
		sort.Slice(out[i].Partners, func(a, b int) bool { return partnerGroupKey(out[i].Partners[a]) < partnerGroupKey(out[i].Partners[b]) })
	}
	return out
}

func FindSharedAddresses(contracts []SupplierContract, profiles map[string]SupplierProfile) []SharedSupplierGroup {
	return sharedGroups(contracts, profiles, func(p SupplierProfile) []string {
		if key := AddressKey(p.Address); key != "" {
			return []string{key}
		}
		return nil
	}, func(g *SharedSupplierGroup, _ string) { g.Address = g.Suppliers[0].Address })
}

func sharedGroups(contracts []SupplierContract, profiles map[string]SupplierProfile, keysOf func(SupplierProfile) []string, label func(*SharedSupplierGroup, string)) []SharedSupplierGroup {
	byCNPJ := map[string][]SupplierContract{}
	for _, c := range contracts {
		byCNPJ[c.CNPJ] = append(byCNPJ[c.CNPJ], c)
	}
	members := map[string]map[string]string{}
	for cnpj := range byCNPJ {
		p, ok := profiles[cnpj]
		if !ok {
			continue
		}
		for _, key := range keysOf(p) {
			if members[key] == nil {
				members[key] = map[string]string{}
			}
			base := CNPJBase(cnpj)
			if prev, ok := members[key][base]; !ok || cnpj < prev {
				members[key][base] = cnpj
			}
		}
	}
	var out []SharedSupplierGroup
	for key, bases := range members {
		if len(bases) < minGroupCompanies {
			continue
		}
		g := SharedSupplierGroup{}
		for _, cnpj := range bases {
			g.Suppliers = append(g.Suppliers, profiles[cnpj])
			g.Contracts = append(g.Contracts, byCNPJ[cnpj]...)
		}
		sort.Slice(g.Suppliers, func(i, j int) bool { return g.Suppliers[i].CNPJ < g.Suppliers[j].CNPJ })
		sort.Slice(g.Contracts, func(i, j int) bool { return g.Contracts[i].First.Before(g.Contracts[j].First) })
		label(&g, key)
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Suppliers) != len(out[j].Suppliers) {
			return len(out[i].Suppliers) > len(out[j].Suppliers)
		}
		return out[i].Suppliers[0].CNPJ < out[j].Suppliers[0].CNPJ
	})
	return out
}

func partnerGroupKey(k PartnerKey) string {
	return string(rune('0'+k.Kind)) + "|" + foldText(k.Name) + "|" + k.Document
}

func AddressKey(address string) string {
	folded := foldText(address)
	if !strings.ContainsFunc(folded, unicode.IsDigit) {
		return ""
	}
	return folded
}

func foldText(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToUpper(foldAccents(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
			continue
		}
		space = true
	}
	return b.String()
}

func FindSanctionedContracts(contracts []SupplierContract, profiles map[string]SupplierProfile, sanctions []Sanction) []SanctionedContract {
	byBase := map[string][]Sanction{}
	for _, s := range sanctions {
		if s.ReachesSaoGoncalo() {
			byBase[CNPJBase(s.CNPJ)] = append(byBase[CNPJBase(s.CNPJ)], s)
		}
	}
	var out []SanctionedContract
	for _, c := range contracts {
		var active []Sanction
		for _, s := range byBase[CNPJBase(c.CNPJ)] {
			if s.CoversDay(c.First) {
				active = append(active, s)
			}
		}
		if len(active) > 0 {
			p, ok := profiles[c.CNPJ]
			if !ok {
				p = SupplierProfile{CNPJ: c.CNPJ}
			}
			out = append(out, SanctionedContract{Profile: p, Contract: c, Sanctions: active})
		}
	}
	return out
}

func (s Sanction) ReachesSaoGoncalo() bool {
	if s.Register == RegisterCEPIM {
		return false
	}
	if strings.Contains(strings.ToUpper(foldAccents(s.Category)), "INIDONEIDADE") {
		return true
	}
	if strings.HasPrefix(strings.ToUpper(foldAccents(s.Scope)), "TODAS AS ESFERAS") {
		return true
	}
	return strings.Contains(foldText(s.Organ), saoGoncaloOrganFragment)
}

func foldAccents(s string) string {
	return accentFolder.Replace(strings.ToLower(s))
}
