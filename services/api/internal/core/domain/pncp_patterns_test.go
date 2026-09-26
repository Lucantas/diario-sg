package domain

import (
	"testing"
	"time"
)

func pncpAt(cnpj, process string, cents int64, signed time.Time) PNCPContract {
	return PNCPContract{ControlNumber: cnpj + process, OrgCNPJ: "28636579000100", Year: signed.Year(), Sequence: 1,
		SupplierCNPJ: cnpj, SupplierName: "EMPRESA " + cnpj, Process: process, ValueCents: cents, SignedAt: &signed}
}

func TestPNCPWithoutExtractSkipsWhatTheDiarioCites(t *testing.T) {
	lastDiario := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	signed := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	contracts := []PNCPContract{
		pncpAt("11111111000111", "100/2026", 50_000_000, signed),
		pncpAt("22222222000122", "200/2026", 50_000_000, signed),
		pncpAt("33333333000133", "300/2026", 50_000_000, signed),
		pncpAt("44444444000144", "400/2026", MinPNCPWithoutExtractCents-1, signed),
		pncpAt("55555555000155", "500/2026", 50_000_000, lastDiario.AddDate(0, 0, -PNCPExtractGraceDays+1)),
		pncpAt("66666666000166", "600/2026", 90_000_000, signed),
	}
	cited := map[string]bool{"11111111000111": true}
	processes := map[string]bool{contracts[1].ProcessKey(): true}

	got := FindPNCPWithoutExtract(contracts, cited, processes, lastDiario, nil)

	if len(got) != 2 || got[0].Contract.SupplierCNPJ != "66666666000166" || got[1].Contract.SupplierCNPJ != "33333333000133" {
		t.Fatalf("contratos sem extrato: %+v", got)
	}
	if got[0].Profile.Name != "EMPRESA 66666666000166" {
		t.Fatalf("sem cadastro, o nome vem do PNCP: %+v", got[0].Profile)
	}
}

func TestPNCPWithoutExtractNeedsADate(t *testing.T) {
	c := pncpAt("33333333000133", "300/2026", 50_000_000, time.Now())
	c.SignedAt = nil

	if got := FindPNCPWithoutExtract([]PNCPContract{c}, nil, nil, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), nil); len(got) != 0 {
		t.Fatalf("sem data de assinatura nem de publicação: %+v", got)
	}
}

func TestPNCPWithoutExtractFindingLinksToThePNCP(t *testing.T) {
	c := pncpAt("33333333000133", "300/2026", 50_000_000, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))

	f := PNCPWithoutExtractFinding(PNCPWithoutExtract{Profile: SupplierProfile{CNPJ: c.SupplierCNPJ}, Contract: c})

	if f.Link == nil || f.Link.URL != c.URL() {
		t.Fatalf("link: %+v", f.Link)
	}
}
