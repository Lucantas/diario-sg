package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeFederalSource struct {
	files  map[string]map[string]string
	latest time.Time
	asked  []string
}

func (f *fakeFederalSource) LatestMonth(context.Context, string) (time.Time, error) {
	return f.latest, nil
}

func (f *fakeFederalSource) ZipCSVs(_ context.Context, file string, each func(name string, header, row []string) error) (string, error) {
	f.asked = append(f.asked, file)
	csvs, ok := f.files[file]
	if !ok {
		return "", fmt.Errorf("%s indisponível", file)
	}
	for name, text := range csvs {
		r := csv.NewReader(strings.NewReader(text))
		r.Comma = ';'
		rows, err := r.ReadAll()
		if err != nil {
			return "", err
		}
		for _, row := range rows[1:] {
			if err := each(name, rows[0], row); err != nil {
				return "", err
			}
		}
	}
	return "sha-" + file, nil
}

type fakeFederalRepo struct {
	amendments []domain.Amendment
	payments   []domain.AmendmentPayment
	months     map[string][]domain.FederalTransfer
}

func (f *fakeFederalRepo) Ready(context.Context) error { return nil }

func (f *fakeFederalRepo) ReplaceAmendments(_ context.Context, a []domain.Amendment, p []domain.AmendmentPayment) error {
	f.amendments, f.payments = a, p
	return nil
}

func (f *fakeFederalRepo) ReplaceTransferMonth(_ context.Context, month time.Time, t []domain.FederalTransfer) error {
	if f.months == nil {
		f.months = map[string][]domain.FederalTransfer{}
	}
	f.months[month.Format("200601")] = t
	return nil
}

const amendmentsCSVFixture = `Código da Emenda;Ano da Emenda;Tipo de Emenda;Nome do Autor da Emenda;Número da emenda;Código Município IBGE;Nome Função;Nome Subfunção;Nome Programa;Nome Ação;Valor Empenhado;Valor Liquidado;Valor Pago
1;2024;Individual;PARLAMENTAR A;0001;3304904;Saúde;S;P;A;10,00;5,00;5,00
2;2024;Individual;PARLAMENTAR B;0002;3303302;Saúde;S;P;A;10,00;5,00;5,00`

const paymentsCSVFixture = `Código da Emenda;Nome do Autor da Emenda;Tipo de Emenda;Ano/Mês;Código do Favorecido;Favorecido;Natureza Jurídica;Tipo Favorecido;UF Favorecido;Município Favorecido;Valor Recebido
1;PARLAMENTAR A;Individual;202601;11222333000181;EMPRESA;Sociedade;Pessoa Jurídica;RJ;SÃO GONÇALO;100,00
1;PARLAMENTAR A;Individual;202601;***123456**;PESSOA;;Pessoa Fisica;RJ;SÃO GONÇALO;50,00`

func transfersFixture(month string) string {
	return `ANO / MÊS;TIPO TRANSFERÊNCIA;UF;NOME MUNICÍPIO;NOME ÓRGÃO;NOME FUNÇÃO;NOME PROGRAMA;NOME AÇÃO;LINGUAGEM CIDADÃ;CÓDIGO FAVORECIDO;NOME FAVORECIDO;VALOR TRANSFERIDO
` + month + `;Legais;RJ;SAO GONCALO;Saúde;Saúde;P;A;;11222333000181;FUNDO;1000,00
` + month + `;Legais;RJ;NITEROI;Saúde;Saúde;P;A;;11222333000181;FUNDO;1000,00`
}

func TestLoadFederalAmendmentsKeepSaoGoncaloCompanies(t *testing.T) {
	src := &fakeFederalSource{files: map[string]map[string]string{amendmentsFile: {
		amendmentsCSV: amendmentsCSVFixture, amendmentPaymentsCSV: paymentsCSVFixture, "EmendasParlamentares_Convenios.csv": "x\n1"}}}
	repo, raw := &fakeFederalRepo{}, &memObjects{}

	run, err := NewLoadFederal(src, repo, &memRuns{}, raw, staffNow).Amendments(context.Background())

	if err != nil || run.Stored != 2 || len(repo.amendments) != 1 || repo.amendments[0].Author != "PARLAMENTAR A" || len(repo.payments) != 1 ||
		repo.payments[0].ValueCents != 10000 {
		t.Fatalf("veio %+v %v %+v %+v", run, err, repo.amendments, repo.payments)
	}
	archived := string(raw.data["raw/cgu_emendas/2026/09/26/EmendasParlamentares_PorFavorecido.csv.gz"])
	if archived == "" || strings.Contains(archived, "PESSOA") {
		t.Fatalf("bruto: %v", raw.names)
	}
}

func TestLoadFederalTransfersReadTheLastThreeMonths(t *testing.T) {
	src := &fakeFederalSource{latest: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), files: map[string]map[string]string{
		"transferencias/202607": {"t.csv": transfersFixture("202607")},
		"transferencias/202608": {"t.csv": transfersFixture("202608")},
		"transferencias/202609": {"t.csv": transfersFixture("202609")},
	}}
	repo := &fakeFederalRepo{}

	run, err := NewLoadFederal(src, repo, &memRuns{}, &memObjects{}, staffNow).Transfers(context.Background(), time.Time{}, time.Time{})

	if err != nil || run.Stored != 3 || len(repo.months) != 3 || repo.months["202608"][0].ValueCents != 100000 {
		t.Fatalf("veio %+v %v %+v", run, err, repo.months)
	}
}

func TestLoadFederalTransfersStopOnAMissingMonth(t *testing.T) {
	src := &fakeFederalSource{files: map[string]map[string]string{}}
	runs := &memRuns{}
	month := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := NewLoadFederal(src, &fakeFederalRepo{}, runs, &memObjects{}, staffNow).Transfers(context.Background(), month, month)

	if err == nil || len(runs.runs) != 1 || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v", err, runs.runs)
	}
}

func TestLoadFederalAmendmentsRejectTheLoadWhenARowFails(t *testing.T) {
	broken := strings.Replace(paymentsCSVFixture, ";202601;11222333000181;", ";janeiro;11222333000181;", 1)
	src := &fakeFederalSource{files: map[string]map[string]string{amendmentsFile: {amendmentsCSV: amendmentsCSVFixture, amendmentPaymentsCSV: broken}}}
	repo, runs := &fakeFederalRepo{}, &memRuns{}

	_, err := NewLoadFederal(src, repo, runs, &memObjects{}, staffNow).Amendments(context.Background())

	if err == nil || repo.amendments != nil || runs.runs[0].Failed != 1 || !strings.Contains(runs.runs[0].Error, "janeiro") {
		t.Fatalf("veio %v %+v %+v", err, repo.amendments, runs.runs)
	}
}
