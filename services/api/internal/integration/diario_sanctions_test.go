//go:build integration

package integration

import (
	"strings"
	"testing"
)

const sanctionGazette = `ATOS DO PREFEITO
SEMED
EXTRATO DE MULTA
O SECRETÁRIO MUNICIPAL DE EDUCAÇÃO, no uso de suas atribuições legais,
decide aplicar a sanção de multa à empresa Home Bread Indústria e
Comércio LTDA., CNPJ 00.768.165/0001-08, pela inexecução parcial do
objeto contratado. Nesse sentido, fica aplicada a multa no valor de
R$ 44.808,00.
EXTRATO DO CONTRATO Nº 12/2025
Partes: Município de São Gonçalo e Home Bread Indústria e Comércio
LTDA., CNPJ 00.768.165/0001-08. Objeto: fornecimento de pães. O contrato
prevê multa de 2% em caso de atraso. Valor: R$ 100.000,00.
`

func TestCompanyShowsPunishmentsPublishedInTheDiario(t *testing.T) {
	srv, _ := newServerFor(t, sanctionGazette)

	var company struct {
		Sanctions []struct {
			Kind string `json:"kind"`
			Act  struct {
				Title string `json:"title"`
			} `json:"act"`
		} `json:"diario_sanctions"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/00768165000108", &company)

	if len(company.Sanctions) != 1 || company.Sanctions[0].Kind != "multa" || !strings.Contains(company.Sanctions[0].Act.Title, "MULTA") {
		t.Fatalf("punições: %+v", company.Sanctions)
	}
}
