package entities

import (
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const extrato = `EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO CONTRATO DE LOCAÇÃO 006/2020.
PROCEDIMENTO ADMINISTRATIVO Nº 21.061/2020
PARTES: MUNICÍPIO DE SÃO GONÇALO e EMPRESA EXEMPLO LTDA, CNPJ: 12.345.678/0001-90.
VALOR MENSAL: R$ 7.972,26 (sete mil novecentos e setenta e dois reais e vinte e seis centavos).
VALOR GLOBAL: R$ 95.667,12 (noventa e cinco mil seiscentos e sessenta e sete reais e doze centavos).
Termo Aditivo ao Contrato SEMCOM nº 07/2023. Processo SEI! nº 03.05511/2026-8.
VALOR (R$ 1)
ACRÉSCIMO CANCELAMENTO
Processo: 1613/2026
Processo Administrativo nº 8.189/2025
inscrita sob o CNPJ n° 14.955.752/0001-10, PREGÃO ELETRÔNICO nº 90034/2025`

func TestExtract(t *testing.T) {
	got := New().Extract(extrato)
	want := map[string]string{
		"cnpj:12345678000190":   "12.345.678/0001-90",
		"cnpj:14955752000110":   "14.955.752/0001-10",
		"valor:797226":          "R$ 7.972,26",
		"valor:9566712":         "R$ 95.667,12",
		"contrato:006/2020":     "CONTRATO DE LOCAÇÃO 006/2020",
		"contrato:07/2023":      "Contrato SEMCOM nº 07/2023",
		"processo:210612020":    "PROCEDIMENTO ADMINISTRATIVO Nº 21.061/2020",
		"processo:030551120268": "Processo SEI! nº 03.05511/2026-8",
		"processo:16132026":     "Processo: 1613/2026",
		"processo:81892025":     "Processo Administrativo nº 8.189/2025",
	}
	for _, e := range got {
		key := string(e.Kind) + ":" + e.Normalized
		v, ok := want[key]
		if !ok {
			t.Errorf("entidade inesperada: %+v", e)
			continue
		}
		if e.Value != v {
			t.Errorf("%s: esperava value %q, veio %q", key, v, e.Value)
		}
		delete(want, key)
	}
	for key := range want {
		t.Errorf("entidade não extraída: %s", key)
	}
}

func TestExtractIgnoresTableHeaderAndDuplicates(t *testing.T) {
	got := New().Extract("VALOR (R$ 1) R$ 10,00 R$ 10,00 R$10,00 CNPJ: 12.345.678/0001-901")
	if len(got) != 1 || got[0].Kind != domain.EntityValor || got[0].Normalized != "1000" {
		t.Errorf("esperava só um valor de 1000 centavos, veio %+v", got)
	}
}

func TestExtractContractWithSpacesAndYearFirstAcronym(t *testing.T) {
	got := New().Extract("EXTRATO DE CONTRATO 015/SEMPAD/2026 e CONTRATO Nº 12/2026/SMTC e contrato: R$ 1.000,00")
	var contracts []string
	for _, e := range got {
		if e.Kind == domain.EntityContrato {
			contracts = append(contracts, e.Normalized)
		}
	}
	if len(contracts) != 2 || contracts[0] != "015/SEMPAD/2026" || contracts[1] != "12/2026/SMTC" {
		t.Errorf("contratos inesperados: %v", contracts)
	}
}
