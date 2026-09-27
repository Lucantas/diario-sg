package domain

import (
	"testing"
	"time"
)

func TestFindingsCarryTheKeysOfWhoTheyAreAbout(t *testing.T) {
	opened := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	profile := SupplierProfile{CNPJ: "11222333000181", Name: "ACME LTDA", OpenedAt: &opened, CapitalCents: 100}
	other := SupplierProfile{CNPJ: "44555666000199", Name: "BETA LTDA"}
	contract := SupplierContract{CNPJ: profile.CNPJ, First: opened, ContractedCents: 1000, ActID: "a1"}
	group := SharedSupplierGroup{Address: "RUA X", Suppliers: []SupplierProfile{profile, other}, Contracts: []SupplierContract{contract}}

	cases := map[string]struct {
		finding Finding
		kind    EntityKind
		keys    []string
	}{
		"fracionamento": {SplitDispensaFinding(SplitDispensa{CNPJ: profile.CNPJ, Contracts: []DispensaContract{
			{Processes: []DispensaProcess{{Key: "06109812025", Label: "06.10981/2025"}}},
			{Processes: []DispensaProcess{{Key: "06109812025", Label: "06.10981/2025"}}}}}), EntityProcesso, []string{"06109812025"}},
		"aditivo":           {ExcessiveAddendumFinding(ExcessiveAddendum{ContractKey: "55/2026"}), EntityContrato, []string{"55/2026"}},
		"emergencial":       {RenewedEmergencyFinding(RenewedEmergency{Supplier: cnpjSupplierPrefix + profile.CNPJ, Contracts: []EmergencyContract{{First: opened}}}), EntityCNPJ, []string{profile.CNPJ}},
		"empresa nova":      {NewCompanyFinding(NewCompanyContract{Profile: profile, Contract: contract}), EntityCNPJ, []string{profile.CNPJ}},
		"capital":           {UndercapitalizedFinding(UndercapitalizedContract{Profile: profile, Contract: contract}), EntityCNPJ, []string{profile.CNPJ}},
		"sócio em comum":    {SharedPartnerFinding(group), EntityCNPJ, []string{profile.CNPJ, other.CNPJ}},
		"endereço":          {SharedAddressFinding(group), EntityCNPJ, []string{profile.CNPJ, other.CNPJ}},
		"sancionado":        {SanctionedFinding(SanctionedContract{Profile: profile, Contract: contract}), EntityCNPJ, []string{profile.CNPJ}},
		"pago sem publicar": {PaidWithoutPublicationFinding(PaidWithoutPublication{Profile: profile}), EntityCNPJ, []string{profile.CNPJ}},
		"sem pagamento":     {UnpaidContractFinding(UnpaidContract{Profile: profile, Contract: contract}), EntityCNPJ, []string{profile.CNPJ}},
		"pago acima":        {PaidAboveAnnouncedFinding(PaidAboveAnnounced{Profile: profile}), EntityCNPJ, []string{profile.CNPJ}},
		"pncp":              {PNCPWithoutExtractFinding(PNCPWithoutExtract{Profile: profile, Contract: PNCPContract{SupplierCNPJ: profile.CNPJ}}), EntityCNPJ, []string{profile.CNPJ}},
	}
	for name, c := range cases {
		for _, key := range c.keys {
			if !c.finding.Mentions(c.kind, key) {
				t.Errorf("%s: sem %s %s em %+v", name, c.kind, key, c.finding.Entities)
			}
		}
	}
	if split := cases["fracionamento"].finding; !split.Mentions(EntityCNPJ, profile.CNPJ) || len(split.Entities) != 2 {
		t.Errorf("fracionamento repete o processo ou perde o CNPJ: %+v", split.Entities)
	}
}

func TestRenewedEmergencyByNameHasNoEntity(t *testing.T) {
	f := RenewedEmergencyFinding(RenewedEmergency{Supplier: nameSupplierPrefix + "ACME", Contracts: []EmergencyContract{{First: time.Now()}}})

	if f.Mentions(EntityCNPJ, "") || len(f.Entities) != 0 {
		t.Errorf("fornecedor só com nome virou entidade: %+v", f.Entities)
	}
}

func TestFindingsSkipInvalidCNPJsAndCarryProcessAndContractKeys(t *testing.T) {
	personal := PNCPWithoutExtractFinding(PNCPWithoutExtract{Contract: PNCPContract{SupplierCNPJ: "", Process: "06.10981/2025-7"}})
	if personal.Mentions(EntityCNPJ, "") || !personal.Mentions(EntityProcesso, "061098120257") || len(personal.Entities) != 1 {
		t.Errorf("PNCP sem CNPJ: %+v", personal.Entities)
	}
	renewed := RenewedEmergencyFinding(RenewedEmergency{Supplier: nameSupplierPrefix + "ACME", Contracts: []EmergencyContract{
		{First: time.Now(), Refs: []string{"processo:1234567", "contrato:55/2026"}},
		{First: time.Now(), Refs: []string{"processo:1234567", "cnpj:11222333000181"}}}})
	if !renewed.Mentions(EntityProcesso, "1234567") || !renewed.Mentions(EntityContrato, "55/2026") || len(renewed.Entities) != 2 {
		t.Errorf("emergencial com processo e contrato: %+v", renewed.Entities)
	}
}
