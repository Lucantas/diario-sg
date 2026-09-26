package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakePanelSource struct {
	acts     []domain.PanelAct
	source   string
	askedIDs []string
}

func (f *fakePanelSource) PanelActs(_ context.Context, source string) ([]domain.PanelAct, error) {
	f.source = source
	return f.acts, nil
}

func (f *fakePanelSource) HitsByIDs(_ context.Context, ids []string) ([]domain.ActHit, error) {
	f.askedIDs = ids
	out := make([]domain.ActHit, 0, len(ids))
	for _, id := range ids {
		out = append(out, domain.ActHit{Act: domain.Act{ID: id}})
	}
	return out, nil
}

func TestGetSupplierPanelLoadsOnlyTheLargestActOfEachSupplier(t *testing.T) {
	day := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	head := "EXTRATO DO CONTRATO Nº 1/2025 PARTES: MUNICÍPIO DE SÃO GONÇALO"
	src := &fakePanelSource{acts: []domain.PanelAct{
		{ActID: "menor", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 100, Type: domain.ActContrato, Title: head, Head: head, Refs: []string{"processo:1"}},
		{ActID: "maior", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 900, Type: domain.ActContrato, Title: head, Head: head, Refs: []string{"processo:2"}},
	}}

	panel, largest, err := NewGetSupplierPanel(src, &fakeRegistry{}, &fakeRegistry{}).Execute(context.Background(), "", domain.PanelFilter{Year: 2025})

	if err != nil || src.source != domain.SourceDiarioPrefeitura || len(panel.Rows) != 1 || panel.Rows[0].ContractedCents != 1000 {
		t.Fatalf("painel inesperado: %+v %v (fonte %q)", panel, err, src.source)
	}
	if len(src.askedIDs) != 1 || largest["maior"].ID != "maior" {
		t.Fatalf("deveria carregar só o ato maior: %v %+v", src.askedIDs, largest)
	}
}

func TestGetSupplierPanelRejectsUnknownSourceAndYear(t *testing.T) {
	uc := NewGetSupplierPanel(&fakePanelSource{}, &fakeRegistry{}, &fakeRegistry{})
	for _, c := range []struct {
		source string
		f      domain.PanelFilter
	}{{"diario_x", domain.PanelFilter{}}, {"", domain.PanelFilter{Year: 1999}}, {"", domain.PanelFilter{Year: 2101}}} {
		if _, _, err := uc.Execute(context.Background(), c.source, c.f); !errors.Is(err, domain.ErrInvalidFilter) {
			t.Errorf("%+v: esperava filtro inválido, veio %v", c, err)
		}
	}
}

func TestGetSupplierPanelNamesTheSuppliersFromTheRegistry(t *testing.T) {
	day := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	head := "EXTRATO DO CONTRATO Nº 1/2025 PARTES: MUNICÍPIO DE SÃO GONÇALO"
	src := &fakePanelSource{acts: []domain.PanelAct{
		{ActID: "a", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 100, Type: domain.ActContrato, Title: head, Head: head, Refs: []string{"processo:1"}},
		{ActID: "b", CNPJ: "44555666000199", PublishedAt: day, ValueCents: 50, Type: domain.ActContrato, Title: head, Head: head, Refs: []string{"processo:2"}},
	}}
	registry := &fakeRegistry{byCNPJ: map[string]*domain.CompanyRegistry{"11222333000181": {Company: domain.RegistryCompany{Name: "EMPRESA A LTDA"}}}}

	panel, _, err := NewGetSupplierPanel(src, registry, registry).Execute(context.Background(), "", domain.PanelFilter{})

	if err != nil || len(panel.Rows) != 2 || panel.Rows[0].Name != "EMPRESA A LTDA" || panel.Rows[1].Name != "" {
		t.Fatalf("nomes: %+v %v", panel.Rows, err)
	}
}
