package domain

import "testing"

const rree2025 = `{"items":[
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"TotalDespesas","conta":"TOTAL DAS DESPESAS (XII) = (X + XI)","coluna":"DOTAÇÃO INICIAL (d)","valor":2403857521},
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"TotalDespesas","conta":"TOTAL DAS DESPESAS (XII) = (X + XI)","coluna":"DESPESAS EMPENHADAS NO BIMESTRE","valor":250016991.87},
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"TotalDespesas","conta":"TOTAL DAS DESPESAS (XII) = (X + XI)","coluna":"DESPESAS EMPENHADAS ATÉ O BIMESTRE (f)","valor":2587537597.59},
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"TotalDespesas","conta":"TOTAL DAS DESPESAS (XII) = (X + XI)","coluna":"DESPESAS LIQUIDADAS ATÉ O BIMESTRE (h)","valor":2554572607.02},
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"TotalDespesas","conta":"TOTAL DAS DESPESAS (XII) = (X + XI)","coluna":"DESPESAS PAGAS ATÉ O BIMESTRE (j)","valor":2540139355.06},
{"exercicio":2025,"anexo":"RREO-Anexo 01","cod_conta":"DespesasExcetoIntraOrcamentarias","conta":"DESPESAS (EXCETO INTRA-ORÇAMENTÁRIAS) (VIII)","coluna":"DESPESAS PAGAS ATÉ O BIMESTRE (j)","valor":2450128252.42}
]}`

func TestParseRREOTotalsReadsTheYearToDateColumns(t *testing.T) {
	got, ok, err := ParseRREOTotals([]byte(rree2025), 2025, 6)

	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if got != (FiscalTotal{Year: 2025, Period: 6, CommittedCents: 258753759759, LiquidatedCents: 255457260702, PaidCents: 254013935506}) {
		t.Errorf("totais: %+v", got)
	}
}

func TestParseRREOTotalsWithoutItemsIsAMissingPeriod(t *testing.T) {
	_, ok, err := ParseRREOTotals([]byte(`{"items":[],"hasMore":false}`), 2026, 4)

	if err != nil || ok {
		t.Errorf("bimestre sem dado: %v %v", ok, err)
	}
	if _, _, err := ParseRREOTotals([]byte(`{"items":[{"cod_conta":"Outra","coluna":"x","valor":1}]}`), 2026, 3); err == nil {
		t.Error("RREO sem as colunas do total aceito")
	}
}

func TestBuildFiscalControlComparesTCEWithTheRREO(t *testing.T) {
	totals := []FiscalTotal{{Year: 2024, Period: 6, PaidCents: 1000}, {Year: 2025, Period: 6, PaidCents: 1000}, {Year: 2026, Period: 3, PaidCents: 500}}
	tce := []YearPaid{{Year: 2024, PaidCents: 950}, {Year: 2025, PaidCents: 800}, {Year: 2026, PaidCents: 100}}

	got := BuildFiscalControl(totals, tce, map[int]int64{2025: 990})

	if got[0].PaidCoverageBP != 9500 || got[0].LowCoverage() || got[1].PaidCoverageBP != 8000 || !got[1].LowCoverage() {
		t.Errorf("anos fechados: %+v", got[:2])
	}
	if !got[1].PortalLoaded || got[1].PortalPaidCents != 990 || got[0].PortalLoaded {
		t.Errorf("portal: %+v", got[:2])
	}
	if got[2].LowCoverage() {
		t.Error("ano em curso não é comparável ao bimestre publicado")
	}
	if missing := BuildFiscalControl([]FiscalTotal{{Year: 2017, Period: 6, PaidCents: 1000}}, tce, nil)[0]; missing.TCELoaded || missing.LowCoverage() {
		t.Errorf("ano sem carga do TCE marcado como incompleto: %+v", missing)
	}
}
