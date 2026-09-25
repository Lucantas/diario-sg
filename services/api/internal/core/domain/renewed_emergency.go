package domain

import (
	"sort"
	"time"
)

const (
	minEmergencyRenewalGap = 30 * 24 * time.Hour
	maxEmergencyRenewalGap = 730 * 24 * time.Hour
	cnpjSupplierPrefix     = "cnpj:"
	nameSupplierPrefix     = "nome:"
)

type EmergencyAct struct {
	ActID       string
	CNPJs       []string
	Refs        []string
	Organ       string
	PublishedAt time.Time
	Body        string
}

type EmergencyContract struct {
	First  time.Time
	ActIDs []string
}

type RenewedEmergency struct {
	Supplier      string
	SupplierLabel string
	Organ         string
	Contracts     []EmergencyContract
}

type emergencySupplier struct {
	key, label string
	named      bool
}

type emergencyContract struct {
	EmergencyContract
	supplier emergencySupplier
	organ    string
}

func FindRenewedEmergencies(acts []EmergencyAct) []RenewedEmergency {
	bySupplier := map[[2]string][]emergencyContract{}
	for _, c := range emergencyContracts(acts, supplierAliases(acts)) {
		k := [2]string{c.supplier.key, c.organ}
		bySupplier[k] = append(bySupplier[k], c)
	}
	var out []RenewedEmergency
	for _, contracts := range bySupplier {
		out = append(out, renewalsOf(contracts)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Contracts[0].First.Equal(out[j].Contracts[0].First) {
			return out[i].Contracts[0].First.Before(out[j].Contracts[0].First)
		}
		return out[i].Supplier < out[j].Supplier
	})
	return out
}

func supplierAliases(acts []EmergencyAct) map[string]string {
	aliases := map[string]string{}
	for _, a := range acts {
		for _, p := range PartiesOf(a.Body) {
			if key := SupplierNameKey(p.Name); p.PublicBody == "" && key != "" {
				aliases[key] = p.CNPJ
			}
		}
		if private := privateCNPJs(a.CNPJs); len(private) == 1 {
			if key := SupplierNameKey(SupplierNameOf(a.Body)); key != "" {
				aliases[key] = private[0]
			}
		}
	}
	return aliases
}

func emergencyContracts(acts []EmergencyAct, aliases map[string]string) []emergencyContract {
	sort.Slice(acts, func(i, j int) bool {
		if !acts[i].PublishedAt.Equal(acts[j].PublishedAt) {
			return acts[i].PublishedAt.Before(acts[j].PublishedAt)
		}
		return acts[i].ActID < acts[j].ActID
	})
	groups := newUnionFind(len(acts))
	firstByRef := map[string]int{}
	for i, a := range acts {
		for _, ref := range a.Refs {
			if j, ok := firstByRef[ref]; ok {
				groups.union(i, j)
			} else {
				firstByRef[ref] = i
			}
		}
	}
	var out []emergencyContract
	for _, members := range groups.sets() {
		c := emergencyContract{EmergencyContract: EmergencyContract{First: acts[members[0]].PublishedAt}}
		suppliers := map[string]emergencySupplier{}
		for _, i := range members {
			c.ActIDs = append(c.ActIDs, acts[i].ActID)
			if c.organ == "" {
				c.organ = PrincipalOrgan(acts[i].Organ)
			}
			if s, ok := emergencySupplierOf(acts[i], aliases); ok {
				if prev, seen := suppliers[s.key]; !seen || (!prev.named && s.named) {
					suppliers[s.key] = s
				}
			}
		}
		if len(suppliers) != 1 {
			continue
		}
		for _, s := range suppliers {
			c.supplier = s
		}
		out = append(out, c)
	}
	return out
}

func emergencySupplierOf(a EmergencyAct, aliases map[string]string) (emergencySupplier, bool) {
	private := privateCNPJs(a.CNPJs)
	name := SupplierNameOf(a.Body)
	switch {
	case len(private) == 1:
		if name == "" {
			return emergencySupplier{key: cnpjSupplierPrefix + private[0], label: FormatCNPJ(private[0])}, true
		}
		return emergencySupplier{cnpjSupplierPrefix + private[0], name + " (" + FormatCNPJ(private[0]) + ")", true}, true
	case len(private) > 1:
		return emergencySupplier{}, false
	}
	key := SupplierNameKey(name)
	if key == "" {
		return emergencySupplier{}, false
	}
	if cnpj, ok := aliases[key]; ok {
		return emergencySupplier{cnpjSupplierPrefix + cnpj, name + " (" + FormatCNPJ(cnpj) + ")", true}, true
	}
	return emergencySupplier{nameSupplierPrefix + key, name, true}, true
}

func renewalsOf(contracts []emergencyContract) []RenewedEmergency {
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].First.Before(contracts[j].First) })
	var out []RenewedEmergency
	var run []emergencyContract
	flush := func() {
		if len(run) >= 2 {
			r := RenewedEmergency{Supplier: run[0].supplier.key, SupplierLabel: run[0].supplier.label, Organ: run[0].organ}
			for _, c := range run {
				r.Contracts = append(r.Contracts, c.EmergencyContract)
			}
			out = append(out, r)
		}
	}
	for _, c := range contracts {
		if len(run) == 0 {
			run = append(run, c)
			continue
		}
		gap := c.First.Sub(run[len(run)-1].First)
		switch {
		case gap < minEmergencyRenewalGap:
			continue
		case gap > maxEmergencyRenewalGap:
			flush()
			run = []emergencyContract{c}
		default:
			run = append(run, c)
		}
	}
	flush()
	return out
}

func privateCNPJs(cnpjs []string) []string {
	var out []string
	for _, cnpj := range cnpjs {
		if _, public := PublicBody(cnpj); !public {
			out = append(out, cnpj)
		}
	}
	return out
}
