package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type recEntityReader struct {
	kind   domain.EntityKind
	key    string
	source string
}

func (r *recEntityReader) ReportByKey(_ context.Context, kind domain.EntityKind, key, source string) (domain.EntityReport, error) {
	r.kind, r.key, r.source = kind, key, source
	return domain.EntityReport{Kind: kind, Key: key}, nil
}

func TestGetEntityNormalizesTheInput(t *testing.T) {
	reader := &recEntityReader{}
	uc := NewGetEntity(reader, &fakeRegistry{}, &fakeRegistry{})

	got, err := uc.Execute(context.Background(), domain.EntityContrato, "001 / 2017", "")

	if err != nil || reader.kind != domain.EntityContrato || reader.key != "1/2017" || got.Key != "1/2017" {
		t.Fatalf("veio %+v %v; leitor recebeu %s %q", got, err, reader.kind, reader.key)
	}
}

func TestGetEntityRejectsInvalidInput(t *testing.T) {
	reader := &recEntityReader{}
	uc := NewGetEntity(reader, &fakeRegistry{}, &fakeRegistry{})

	for _, c := range []struct {
		kind domain.EntityKind
		in   string
		want error
	}{{domain.EntityProcesso, "12", domain.ErrInvalidInput}, {domain.EntityValor, "100", domain.ErrInvalidInput}, {domain.EntityCNPJ, "1", domain.ErrInvalidCNPJ}} {
		if _, err := uc.Execute(context.Background(), c.kind, c.in, ""); !errors.Is(err, c.want) {
			t.Errorf("%s %q: esperava %v, veio %v", c.kind, c.in, c.want, err)
		}
	}
	if reader.key != "" {
		t.Fatal("entrada inválida não deveria chegar ao leitor")
	}
}

func TestGetEntityPassesTheSourceAndRejectsUnknownOnes(t *testing.T) {
	reader := &recEntityReader{}
	uc := NewGetEntity(reader, &fakeRegistry{}, &fakeRegistry{})

	if _, err := uc.Execute(context.Background(), domain.EntityProcesso, "310/2025", domain.SourceDiarioCamara); err != nil || reader.source != domain.SourceDiarioCamara {
		t.Fatalf("fonte não repassada: %q %v", reader.source, err)
	}
	if _, err := uc.Execute(context.Background(), domain.EntityProcesso, "310/2025", "tce"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("fonte desconhecida: %v", err)
	}
}

type fakeEntityReader struct{ report domain.EntityReport }

func (f fakeEntityReader) ReportByKey(context.Context, domain.EntityKind, string, string) (domain.EntityReport, error) {
	return f.report, nil
}

func TestGetEntityFillsPhases(t *testing.T) {
	reader := fakeEntityReader{domain.EntityReport{
		Kind: domain.EntityProcesso,
		Acts: []domain.ActHit{{Act: domain.Act{Type: domain.ActLicitacao, Title: "HOMOLOGAÇÃO"}}},
		TypeTitleCounts: []domain.TypeTitleCount{
			{Type: domain.ActLicitacao, Title: "HOMOLOGAÇÃO", Acts: 2},
			{Type: domain.ActContrato, Title: "EXTRATO DE DISTRATO DE CONTRATO", Acts: 1},
			{Type: domain.ActDespacho, Title: "DESPACHO", Acts: 4},
		},
	}}

	got, err := NewGetEntity(reader, &fakeRegistry{}, &fakeRegistry{}).Execute(context.Background(), domain.EntityProcesso, "2808/2022", "")

	if err != nil {
		t.Fatal(err)
	}
	if got.Acts[0].Phase != domain.PhaseHomologacao {
		t.Fatalf("fase do ato: %s", got.Acts[0].Phase)
	}
	want := map[domain.Phase]int{domain.PhaseHomologacao: 2, domain.PhaseRescisao: 1, domain.PhaseOutro: 4}
	if !reflect.DeepEqual(got.CountByPhase, want) {
		t.Fatalf("contagem por fase: %v", got.CountByPhase)
	}
}

func TestGetEntityAddsTheRegistryOfACNPJ(t *testing.T) {
	month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	reg := &domain.CompanyRegistry{Month: month, Company: domain.RegistryCompany{Name: "F.P. VIEIRA ENGENHARIA LTDA"}}
	registry := &fakeRegistry{byCNPJ: map[string]*domain.CompanyRegistry{"14180324000163": reg}, month: &month}

	got, err := NewGetEntity(&recEntityReader{}, registry, registry).Execute(context.Background(), domain.EntityCNPJ, "14.180.324/0001-63", "")

	if err != nil || got.Registry != reg || got.RegistryMonth == nil || !got.RegistryMonth.Equal(month) {
		t.Fatalf("cadastro: %+v %v", got, err)
	}
}

func TestGetEntityAddsTheSanctionsOfACNPJ(t *testing.T) {
	registry := &fakeRegistry{sanctions: map[string][]domain.Sanction{"14180324000163": {{Register: domain.RegisterCEIS, Code: "1"}}}}

	got, err := NewGetEntity(&recEntityReader{}, registry, registry).Execute(context.Background(), domain.EntityCNPJ, "14.180.324/0001-63", "")

	if err != nil || len(got.Sanctions) != 1 || got.SanctionsListedOn[domain.RegisterCEIS].IsZero() {
		t.Fatalf("sanções: %+v %v", got, err)
	}
}

type companyActs struct{ ports.ActRepository }

func (companyActs) ReportByEntity(context.Context, domain.EntityKind, string) (domain.CompanyReport, error) {
	return domain.CompanyReport{}, nil
}

func TestGetCompanyAddsTheSanctions(t *testing.T) {
	registry := &fakeRegistry{sanctions: map[string][]domain.Sanction{"14180324000163": {{Register: domain.RegisterCNEP, Code: "9"}}}}

	got, err := NewGetCompany(companyActs{}, registry, registry).Execute(context.Background(), "14.180.324/0001-63")

	if err != nil || len(got.Sanctions) != 1 || got.Sanctions[0].Code != "9" {
		t.Fatalf("sanções: %+v %v", got, err)
	}
}

func TestGetEntityDoesNotLookUpTheRegistryOfAProcess(t *testing.T) {
	registry := &fakeRegistry{}

	got, err := NewGetEntity(&recEntityReader{}, registry, registry).Execute(context.Background(), domain.EntityProcesso, "2808/2022", "")

	if err != nil || got.Registry != nil || len(registry.asked) != 0 {
		t.Fatalf("processo não tem cadastro: %+v %v %v", got, err, registry.asked)
	}
}
