package domain

import "testing"

func TestWithPaidAddsThePaidOfTheYearAndSkipsPublicBodiesAndTheOtherDiario(t *testing.T) {
	p := SupplierPanel{Filter: PanelFilter{Year: 2025}, Rows: []SupplierRow{{CNPJ: "14180324000163"}},
		Years: []PanelTotal{{Key: "2025", ContractedCents: 10}}}
	paid := []PaidTotal{
		{CNPJ: "14180324000163", Year: 2025, PaidCents: 100},
		{CNPJ: "14180324000163", Year: 2024, PaidCents: 50},
		{CNPJ: "14180324000163", Year: 2025, Chamber: true, PaidCents: 7},
		{CNPJ: "28636579000100", Year: 2025, PaidCents: 1000},
	}

	got := WithPaid(p, SourceDiarioPrefeitura, paid)

	if got.Rows[0].PaidCents != 100 || got.PaidCents != 100 {
		t.Errorf("pago no ano: %+v", got)
	}
	if len(got.Years) != 2 || got.Years[0].Key != "2024" || got.Years[0].PaidCents != 50 || got.Years[1].PaidCents != 100 || got.Years[1].ContractedCents != 10 {
		t.Errorf("anos: %+v", got.Years)
	}
	if chamber := WithPaid(p, SourceDiarioCamara, paid); chamber.Rows[0].PaidCents != 7 {
		t.Errorf("Câmara: %+v", chamber.Rows)
	}
}
