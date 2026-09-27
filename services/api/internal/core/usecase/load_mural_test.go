package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeMural map[string]string

func (f fakeMural) List(_ context.Context, name string) ([]byte, error) {
	page, ok := f[name]
	if !ok {
		return nil, errors.New("fora do ar")
	}
	return []byte(page), nil
}

func (f fakeMural) BaseURL() string { return "https://mural/" }

type fakeMuralRepo struct {
	procurements []domain.Procurement
	contracts    []domain.ProcurementContract
	saved        bool
}

func (f *fakeMuralRepo) Ready(context.Context) error { return nil }

func (f *fakeMuralRepo) ReplaceMural(_ context.Context, p []domain.Procurement, c []domain.ProcurementContract) error {
	f.procurements, f.contracts, f.saved = p, c, true
	return nil
}

const muralTenderRow = `<table><tr><td><a href="./licitacao.php?licitacao_id=%s"><strong>E</strong></a><small>1/2025</small></td>` +
	`<td><strong>M</strong></td><td></td><td>O</td><td>S</td><td></td></tr></table>`

func muralFixture() fakeMural {
	row := func(id string) string { return fmt.Sprintf(muralTenderRow, id) }
	return fakeMural{
		domain.MuralTenders:       row("1"),
		domain.MuralDirect:        row("2"),
		domain.MuralUnenforceable: row("3"),
		domain.MuralContracts: `<table><tr><td><a href="./licitacao.php?licitacao_id=1"><strong>E</strong></a><small>1/2025</small></td>` +
			`<td><strong>M</strong></td><td>O</td><td>R$ 10,00</td><td>F</td><td><a href="download.php?idf_1=9">Contrato</a></td></tr></table>`,
	}
}

func TestLoadMuralReplacesTheFourLists(t *testing.T) {
	repo, raw := &fakeMuralRepo{}, &memObjects{}

	run, err := NewLoadMural(muralFixture(), repo, &memRuns{}, raw, staffNow).Execute(context.Background())

	if err != nil || run.Stored != 4 || len(repo.procurements) != 3 || repo.procurements[2].List != domain.MuralUnenforceable ||
		len(repo.contracts) != 1 || repo.contracts[0].DocumentURL != "https://mural/download.php?idf_1=9" {
		t.Fatalf("veio %+v %v %+v %+v", run, err, repo.procurements, repo.contracts)
	}
	if _, ok := raw.data["raw/pmsg_mural/2026/09/26/contratos.html.gz"]; !ok {
		t.Errorf("bruto: %v", raw.names)
	}
}

func TestLoadMuralSavesNothingWhenAListFails(t *testing.T) {
	src := muralFixture()
	delete(src, domain.MuralContracts)
	repo, runs := &fakeMuralRepo{}, &memRuns{}

	_, err := NewLoadMural(src, repo, runs, &memObjects{}, staffNow).Execute(context.Background())

	if err == nil || repo.saved || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v", err, runs.runs)
	}
}
