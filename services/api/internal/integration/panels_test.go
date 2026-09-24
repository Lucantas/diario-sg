//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

const (
	semedHomologacao2025 = `ATOS DO PREFEITO
SEMED
EXTRATO DA HOMOLOGAÇÃO DO PREGÃO ELETRÔNICO N°.
90013/2025.
PROCESSO ADMINISTRATIVO Nº. 7717/2025.
Nos termos do relatório final apresentado pelo Pregoeiro, referente
ao Pregão Eletrônico Nº. 90013/2025, cujo objeto é a contratação de
serviços de engenharia de manutenção preventiva e corretiva
predial, com adequações, adaptações e modernizações, incluindo
os projetos e todos os materiais necessários, para atender às
demandas dos imóveis próprios das Unidades Escolares, utilizados
pela Secretaria de Educação do Município de São Gonçalo – RJ.
HOMOLOGO o correspondente procedimento licitatório em favor da
empresa F.P. VIEIRA ENGENHARIA LTDA, CNPJ 14.180.324/0001-63,
com o valor total de R$ 106.292.521,47 (cento e seis milhões e
duzentos e noventa e dois mil e quinhentos e vinte e um reais e
quarenta e sete centavos), para que produza seus efeitos legais e
jurídicos.
EXTRATO DO CONTRATO 010/SEMED/2025 DO PREGÃO
ELETRÔNICO PMSG- 90013/2025.
Processo: 7717/2025.
Partes: MUNICÍPIO DE SÃO GONÇALO e F.P. VIEIRA ENGENHARIA
LTDA, CNPJ 14.180.324/0001-63.
Objeto: Contratação de serviços de engenharia de manutenção
preventiva e corretiva predial, com adequações, adaptações e
modernizações, incluindo os projetos e todos os materiais
necessários, para atender às demandas dos imóveis próprios das
Unidades Escolares, utilizados pela Secretaria de Educação do
Município de São Gonçalo – RJ.
Valor Mensal: R$ 8.857.710,12 (oito milhões e oitocentos e cinquenta
e sete mil e setecentos e dez reais e doze centavos)
Valor Total: R$ 106.292.521,47 (cento e seis milhões e duzentos e
noventa e dois mil e quinhentos e vinte e um reais e quarenta e sete
centavos).
Prazo: 29/05/2025 a 28/05/2026.`
	semmatranAta2026 = `ATOS DO PREFEITO
SEMMATRAN
EXTRATO DA ATA DE REGISTRO DE PREÇOS
O MUNICÍPIO DE SÃO GONÇALO torna público, para o
conhecimento de todos os interessados, o Extrato da Ata de
Registro de Preços Nº 001/SEMMATRAN/2026, referente ao Pregão
Eletrônico N° 90030/2025, Processo Administrativo nº 2.686/2025,
que tem por objeto o Registro de preços para serviços de
engenharia para manutenção e ampliação da sinalização horizontal,
vertical e semafórica no município de São Gonçalo.
TRIGONAL ENGENHARIA LTDA
CNPJ nº 32.040.529/0001-25
TOTAL: R$ 23.949.911,74
BDI 18,99%: R$ 4.548.088,24
VALOR GLOBAL: R$28.497.999,98
São Gonçalo, 11 de março de 2026.`
)

type supplierPanelBody struct {
	Year            *int   `json:"year"`
	Organ           string `json:"organ"`
	Contracts       int    `json:"contracts"`
	Suppliers       int    `json:"suppliers"`
	ContractedCents int64  `json:"contracted_cents"`
	RegisteredCents int64  `json:"registered_cents"`
	Items           []struct {
		CNPJ            string   `json:"cnpj"`
		Contracts       int      `json:"contracts"`
		ContractedCents int64    `json:"contracted_cents"`
		RegisteredCents int64    `json:"registered_cents"`
		Organs          []string `json:"organs"`
		Largest         *struct {
			Title string `json:"title"`
		} `json:"largest"`
	} `json:"items"`
	Years []struct {
		Year      int `json:"year"`
		Contracts int `json:"contracts"`
	} `json:"years"`
	Organs []struct {
		Organ     string `json:"organ"`
		OrganName string `json:"organ_name"`
	} `json:"organs"`
}

func TestSupplierPanelCountsTheContractOnceAndTheAtaApart(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	setOnlyGazetteDate(t, db, "2025-05-30")
	indexAt(t, db, time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC), semmatranAta2026)

	var all supplierPanelBody
	getJSON(t, srv.URL+"/v1/panels/suppliers", &all)

	if all.Year != nil || all.Contracts != 2 || all.Suppliers != 2 || all.ContractedCents != 10629252147 || all.RegisteredCents != 2849799998 {
		t.Fatalf("painel inesperado: %+v", all)
	}
	first := all.Items[0]
	if first.CNPJ != "14180324000163" || first.Contracts != 1 || first.Largest == nil || first.Largest.Title[:20] != "EXTRATO DO CONTRATO " ||
		len(first.Organs) != 1 || first.Organs[0] != "SEMED" {
		t.Fatalf("contrato deveria contar uma vez, pelo extrato: %+v", first)
	}
	if all.Items[1].CNPJ != "32040529000125" || all.Items[1].ContractedCents != 0 || all.Items[1].RegisteredCents != 2849799998 {
		t.Fatalf("ata deveria ficar em registrado: %+v", all.Items[1])
	}
	if len(all.Years) != 2 || all.Years[0].Year != 2025 || all.Years[1].Year != 2026 || len(all.Organs) != 2 || all.Organs[0].OrganName == "" {
		t.Fatalf("totais inesperados: %+v %+v", all.Years, all.Organs)
	}

	var filtered supplierPanelBody
	getJSON(t, srv.URL+"/v1/panels/suppliers?year=2026&organ=semmatran", &filtered)
	if filtered.Year == nil || *filtered.Year != 2026 || filtered.Organ != "SEMMATRAN" || len(filtered.Items) != 1 || filtered.Items[0].CNPJ != "32040529000125" {
		t.Fatalf("filtro inesperado: %+v", filtered)
	}

	for _, q := range []string{"year=abc", "year=0", "year=1999", "source=diario_x"} {
		if r, _ := fetch(t, srv.URL+"/v1/panels/suppliers?"+q); r.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s deveria dar 400, veio %d", q, r.StatusCode)
		}
	}
}
