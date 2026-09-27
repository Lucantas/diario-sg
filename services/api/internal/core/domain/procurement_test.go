package domain

import (
	"testing"
	"time"
)

const muralTenderPage = `<table><tr><th>Nº Edital</th><th>Modalidade</th><th>Abertura</th><th>Objeto</th><th>Situação</th><th></th></tr>
<tr><td><a href="./licitacao.php?licitacao_id=1840"><strong>SRP/PE/21/2026/PMSG</strong></a> <br>
<small title="Processo Administrativo">17378/2025 </small> <br></td>
<td><strong>Pregão eletrônico</strong> <br> <small>MENOR PREÇO UNITARIO POR GRUPO</small></td>
<td>08/10/2026 10:00</td><td>AQUISIÇÃO DE  ALIMENTOS ESPECIAIS</td><td class="status">Em andamento</td>
<td><a href="./licitacao.php?licitacao_id=1840">Detalhes</a></td></tr>
<tr><td><a href="./licitacao.php?licitacao_id=494"><strong>TP 002/2016 PMSG</strong></a><small>13.921/2016</small></td>
<td><strong>Tomada de preço</strong><small>MENOR PREÇO</small></td><td>00/00/0000 00:00</td><td>SERVIÇOS</td><td>Em andamento</td><td></td></tr></table>`

const muralContractPage = `<table><tr><th>Nº Edital</th><th>Modalidade</th><th>Objeto</th><th>Valor</th><th>Fornecedor</th><th>Contrato</th></tr>
<tr><td><a href="./licitacao.php?licitacao_id=1795"><strong>SRP//Exclusivo ME - EPP/PE/PMSG</strong></a><small>02.960/2026 </small></td>
<td><strong>Dispensa</strong> <br> <small>Menor preço por item</small></td><td>Aquisição de água mineral</td><td>R$ 22.050.184,68</td>
<td>CONSTRUTORA METROPOLITANA S/A</td><td><a href="download.php?idf_1=5640" title="Baixar contrato"> Ata de Registro de Preços </a></td></tr>
<tr><td><a href="./licitacao.php?licitacao_id=7"><strong>005/2014</strong></a><small>37.421/2013</small></td>
<td><strong>Pregão eletrônico</strong></td><td>Manutenção</td><td></td><td></td><td> Contrato </td></tr></table>`

func TestProcessKeyDropsDotsAndLeadingZeros(t *testing.T) {
	for in, want := range map[string]string{"02.960/2026": "2960/2026", "35774/2025 ": "35774/2025", "13.921/2016": "13921/2016",
		"071/2026": "71/2026", "30656/25": "30656/2025", "74.00210/2025-0": "210/2025", "/2026": "", "2022": "", "12/2": "",
		"SEI-123/2025": ""} {
		if got := ProcessKey(in); got != want {
			t.Errorf("%q: %q, esperava %q", in, got, want)
		}
	}
}

func TestParseProcurementsReadsTheTenderList(t *testing.T) {
	got, err := ParseProcurements(MuralTenders, []byte(muralTenderPage), "https://mural/")

	if err != nil || len(got) != 2 {
		t.Fatalf("licitações: %+v %v", got, err)
	}
	opens := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	p := got[0]
	if p.ID != 1840 || p.List != MuralTenders || p.Notice != "SRP/PE/21/2026/PMSG" || p.Process != "17378/2025" || p.ProcessKey != "17378/2025" ||
		p.Modality != "Pregão eletrônico" || p.Criterion != "MENOR PREÇO UNITARIO POR GRUPO" || p.OpensAt == nil || !p.OpensAt.Equal(opens) ||
		p.Object != "AQUISIÇÃO DE ALIMENTOS ESPECIAIS" || p.Status != "Em andamento" || p.URL != "https://mural/licitacao.php?licitacao_id=1840" {
		t.Errorf("licitação: %+v", p)
	}
	if got[1].OpensAt != nil || got[1].ProcessKey != "13921/2016" {
		t.Errorf("abertura zerada: %+v", got[1])
	}
}

func TestParseProcurementContractsReadsValueSupplierAndDocument(t *testing.T) {
	got, err := ParseProcurementContracts([]byte(muralContractPage), "https://licitacao.pmsg.rj.gov.br/")

	if err != nil || len(got) != 2 {
		t.Fatalf("contratos: %+v %v", got, err)
	}
	want := ProcurementContract{ProcurementID: 1795, Notice: "SRP//Exclusivo ME - EPP/PE/PMSG", Process: "02.960/2026", ProcessKey: "2960/2026",
		Modality: "Dispensa", Object: "Aquisição de água mineral", ValueCents: 2205018468, Supplier: "CONSTRUTORA METROPOLITANA S/A",
		Instrument: "Ata de Registro de Preços", DocumentURL: "https://licitacao.pmsg.rj.gov.br/download.php?idf_1=5640"}
	if got[0] != want {
		t.Errorf("contrato:\n%+v\n%+v", got[0], want)
	}
	if got[1].ValueCents != 0 || got[1].DocumentURL != "" || got[1].Instrument != "Contrato" {
		t.Errorf("sem valor nem documento: %+v", got[1])
	}
}

func TestParseProcurementsRejectsAChangedPage(t *testing.T) {
	for name, page := range map[string]string{
		"vazia":   `<html><body>manutenção</body></html>`,
		"colunas": `<table><tr><td><a href="./licitacao.php?licitacao_id=1">x</a></td><td>y</td></tr></table>`,
		"link":    `<table><tr><td>x</td><td></td><td></td><td></td><td></td><td></td></tr></table>`,
	} {
		if _, err := ParseProcurements(MuralTenders, []byte(page), ""); err == nil {
			t.Errorf("%s: aceita", name)
		}
	}
}
