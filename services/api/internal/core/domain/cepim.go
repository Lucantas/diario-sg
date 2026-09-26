package domain

import (
	"fmt"
	"strings"
)

const (
	cepimCategory = "Impedida de receber transferências da União"
	cepimScope    = "Convênios e transferências voluntárias da União"
	cepimSphere   = "FEDERAL"

	colCEPIMDocument = "CNPJ ENTIDADE"
	colCEPIMName     = "NOME ENTIDADE"
	colCEPIMCovenant = "NÚMERO CONVÊNIO"
	colCEPIMOrgan    = "ÓRGÃO CONCEDENTE"
	colCEPIMReason   = "MOTIVO DO IMPEDIMENTO"
)

var requiredCEPIMColumns = []string{colCEPIMDocument, colCEPIMName, colCEPIMCovenant, colCEPIMOrgan, colCEPIMReason}

type CEPIMColumns struct{ cols SanctionColumns }

func NewCEPIMColumns(header []string) (CEPIMColumns, error) {
	cols := SanctionColumns{}
	for i, name := range header {
		cols[strings.TrimSpace(name)] = i
	}
	for _, name := range requiredCEPIMColumns {
		if _, ok := cols[name]; !ok {
			return CEPIMColumns{}, fmt.Errorf("cabeçalho do CEPIM sem a coluna %q", name)
		}
	}
	return CEPIMColumns{cols: cols}, nil
}

func (c CEPIMColumns) CNPJ(row []string) string {
	return c.cols.get(row, colCEPIMDocument)
}

func (c CEPIMColumns) IsCompany(row []string) bool {
	return len(c.CNPJ(row)) == cnpjDigits
}

func (c CEPIMColumns) Parse(row []string) (Sanction, error) {
	if len(row) != len(c.cols) {
		return Sanction{}, fieldCountError("impedimento do CEPIM", len(c.cols), len(row))
	}
	covenant := c.cols.get(row, colCEPIMCovenant)
	if !c.IsCompany(row) || covenant == "" {
		return Sanction{}, fmt.Errorf("impedimento do CEPIM sem CNPJ ou convênio: %q", row)
	}
	cnpj := c.CNPJ(row)
	return Sanction{
		Register: RegisterCEPIM, Code: cnpj + "/" + covenant, CNPJ: cnpj, Name: squeezed(c.cols.get(row, colCEPIMName)),
		Category: cepimCategory, Process: "Convênio " + covenant, Organ: squeezed(c.cols.get(row, colCEPIMOrgan)),
		Sphere: cepimSphere, Scope: cepimScope, LegalBasis: squeezed(c.cols.get(row, colCEPIMReason)),
	}, nil
}
