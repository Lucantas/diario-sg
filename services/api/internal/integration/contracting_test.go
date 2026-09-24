//go:build integration

package integration

import "testing"

const contractingGazette = "ATOS DO PREFEITO\nSEMTRAN\n" +
	"EXTRATO DO CONTRATO Nº 2/2024\nModalidade: Dispensa de Licitação nº 7/2024. Contratada: LM CURSOS, CNPJ 18.657.198/0001-46.\n" +
	"Item 1: R$ 100,00. VALOR TOTAL: R$ 57.999,60.\n" +
	"EXTRATO DO CONTRATO Nº 3/2024\nInexigibilidade de licitação, art. 74. Contratada: ALL FOOD, CNPJ 01.742.126/0001-02.\n" +
	"Valor global: R$ 90.000,00.\n" +
	"EXTRATO DO CONTRATO Nº 4/2024\nPregão Eletrônico nº 5/2024. Contratada: INVICTTA, CNPJ 10.746.140/0001-67.\n" +
	"Valor global: R$ 45.000,00."

func TestSearchByModalityAndMainValue(t *testing.T) {
	srv, _ := newServerFor(t, contractingGazette)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	type acts struct {
		Total int `json:"total"`
		Acts  []struct {
			Title     string `json:"titulo"`
			Type      string `json:"tipo"`
			Modality  string `json:"modalidade"`
			MainValue int64  `json:"valor_principal_centavos"`
		} `json:"atos"`
	}

	dispensa, _ := call[acts](t, session, "buscar_atos", map[string]any{"modalidade": "dispensa"})
	inRange, _ := call[acts](t, session, "buscar_atos", map[string]any{"valor_principal_min": 40000, "valor_principal_max": 62000})
	_, bad := call[acts](t, session, "buscar_atos", map[string]any{"modalidade": "carta"})

	if dispensa.Total != 1 || dispensa.Acts[0].Type != "contrato" || dispensa.Acts[0].Modality != "dispensa" || dispensa.Acts[0].MainValue != 5799960 {
		t.Fatalf("o contrato por dispensa tem tipo contrato, modalidade dispensa e o valor total: %+v", dispensa)
	}
	if inRange.Total != 2 {
		t.Fatalf("dois contratos com valor principal entre 40 e 62 mil (o item de R$ 100 não conta): %+v", inRange)
	}
	if !bad.IsError {
		t.Fatal("modalidade desconhecida é erro de entrada")
	}

	var hits struct {
		Items []struct {
			Modality  string `json:"modality"`
			MainValue int64  `json:"main_value_cents"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?modality=inexigibilidade", &hits)
	if len(hits.Items) != 1 || hits.Items[0].Modality != "inexigibilidade" || hits.Items[0].MainValue != 9000000 {
		t.Fatalf("filtro modality na API: %+v", hits.Items)
	}

	groups, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "cnpj", "modalidade": "pregao"})
	if len(groups.Groups) != 1 || groups.Groups[0].Key != "10746140000167" {
		t.Fatalf("agrupar aceita a modalidade: %+v", groups)
	}
}

const quotaGazette = "TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS\n" +
	"Processo n: 1198/2025\nautorizo a publicação da prestação de contas de Cota para o Exercício da Atividade Parlamentar Municipal - " +
	"CEAPM que foi APROVADA, apresentada pelo a JUAN PATRICK\nPINHEIRO DE OLIVEIRA – Vereador JUAN OLIVEIRA, relativo\n" +
	"ao mês de OUTUBRO de 2025, no valor de R$ 10.000,00 (dez\nmil reais)."

func TestReadActShowsPartiesAndParliamentaryQuota(t *testing.T) {
	for _, c := range []struct {
		name, text string
		check      func(t *testing.T, out readWithFacts)
	}{
		{"partes do contrato", contractingGazette, func(t *testing.T, out readWithFacts) {
			if len(out.Parties) != 1 || out.Parties[0].CNPJ != "18657198000146" || out.Parties[0].Name != "LM CURSOS" {
				t.Fatalf("partes: %+v", out.Parties)
			}
		}},
		{"cota parlamentar", quotaGazette, func(t *testing.T, out readWithFacts) {
			if out.Quota == nil || out.Quota.Councillor != "JUAN OLIVEIRA" || out.Quota.Month != "2025-10" || out.Quota.ValueCents != 1000000 {
				t.Fatalf("cota: %+v", out.Quota)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, _ := newServerFor(t, c.text)
			_, key := issueKey(t, srv.URL, "")
			session, err := connect(t, srv.URL, key)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			found, _ := call[struct {
				Acts []struct {
					GazetteID string `json:"edicao_id"`
					Position  int    `json:"posicao"`
				} `json:"atos"`
			}](t, session, "buscar_atos", map[string]any{})
			if len(found.Acts) == 0 {
				t.Fatal("nenhum ato indexado")
			}

			out, _ := call[readWithFacts](t, session, "ler_ato", map[string]any{"edicao_id": found.Acts[0].GazetteID, "posicao": 0})

			c.check(t, out)
		})
	}
}

type readWithFacts struct {
	Parties []struct {
		CNPJ string `json:"cnpj"`
		Name string `json:"nome_provavel"`
	} `json:"partes"`
	Quota *struct {
		Councillor string `json:"vereador"`
		Month      string `json:"mes_referencia"`
		ValueCents int64  `json:"valor_centavos"`
	} `json:"cota_parlamentar"`
}

func TestIndexesAMainValueAboveTheIntegerRange(t *testing.T) {
	srv, _ := newServerFor(t, "EXTRATO DO CONTRATO Nº 9/2026\nPregão Eletrônico nº 1/2026.\nValor global: R$ 5.563.722.283,00.")
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	out, _ := call[struct {
		Acts []struct {
			MainValue int64 `json:"valor_principal_centavos"`
		} `json:"atos"`
	}](t, session, "buscar_atos", map[string]any{})

	if len(out.Acts) != 1 || out.Acts[0].MainValue != 556372228300 {
		t.Fatalf("valor principal acima de 2^31 centavos: %+v", out.Acts)
	}
}
