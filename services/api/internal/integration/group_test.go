//go:build integration

package integration

import "testing"

const groupGazette = "SEMED\n" +
	"EXTRATO DO CONTRATO Nº 1/2026\nPartes: MUNICÍPIO DE SÃO GONÇALO, CNPJ 28.636.579/0001-00, e ALL FOOD LTDA, CNPJ 01.742.126/0001-02.\n" +
	"Processo nº 100/2026. Valor global: R$ 50.000,00.\n" +
	"EXTRATO DO TERMO ADITIVO Nº 1 AO CONTRATO Nº 1/2026\nPartes: MUNICÍPIO DE SÃO GONÇALO e ALL FOOD LTDA, CNPJ 01.742.126/0001-02.\n" +
	"Processo nº 100/2026. Acréscimo de R$ 12.500,00.\n" +
	"EXTRATO DO CONTRATO Nº 2/2026\nPartes: MUNICÍPIO DE SÃO GONÇALO, CNPJ 28.636.579/0001-00, e INVICTTA LTDA, CNPJ 10.746.140/0001-67.\n" +
	"Processo nº 200/2026. Valor global: R$ 8.000,00."

type groupsOut struct {
	MatchedActs int `json:"atos_encontrados"`
	Groups      []struct {
		Key           string `json:"chave"`
		Name          string `json:"nome"`
		Acts          int    `json:"atos"`
		MaxValueCents int64  `json:"maior_valor_centavos"`
		Examples      []struct {
			Title string `json:"titulo"`
		} `json:"exemplos"`
	} `json:"grupos"`
}

func TestMCPGroupsActsBySupplierLeavingPublicBodiesOut(t *testing.T) {
	srv, _ := newServerFor(t, "ATOS DO PREFEITO\n"+groupGazette)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	bySupplier, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "cnpj"})
	withPublic, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "cnpj", "incluir_orgaos_publicos": true})
	byProcess, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "processo", "tipo": "contrato"})
	byOrgan, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "orgao"})

	if bySupplier.MatchedActs != 3 || len(bySupplier.Groups) != 2 {
		t.Fatalf("dois fornecedores em três atos: %+v", bySupplier)
	}
	top := bySupplier.Groups[0]
	if top.Key != "01742126000102" || top.Acts != 2 || top.MaxValueCents != 5000000 || len(top.Examples) != 2 {
		t.Errorf("o fornecedor com dois atos vem primeiro, com o maior valor citado: %+v", top)
	}
	municipality := ""
	for _, g := range withPublic.Groups {
		if g.Key == "28636579000100" {
			municipality = g.Name
		}
	}
	if len(withPublic.Groups) != 3 || municipality != "Município de São Gonçalo" {
		t.Errorf("com órgãos públicos, o Município entra e tem nome: %+v", withPublic.Groups)
	}
	if byProcess.MatchedActs != 2 || len(byProcess.Groups) != 2 {
		t.Errorf("os filtros da busca valem no agrupamento: %+v", byProcess)
	}
	if len(byOrgan.Groups) != 1 || byOrgan.Groups[0].Key != "SEMED" || byOrgan.Groups[0].Acts != 3 {
		t.Errorf("agrupamento por órgão: %+v", byOrgan.Groups)
	}
}

func TestMCPGroupRejectsUnknownGrouping(t *testing.T) {
	srv, _ := newServerFor(t, groupGazette)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	_, res := call[groupsOut](t, session, "agrupar", map[string]any{"por": "empresa"})

	if !res.IsError {
		t.Fatal("agrupar por empresa deveria ser erro de entrada")
	}
}
