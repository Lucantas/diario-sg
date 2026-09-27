package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	SourcePNCP       = "pncp_contratos"
	RecordPNCP       = "contrato_pncp"
	FirstPNCPYear    = 2021
	pncpContractPage = "https://pncp.gov.br/app/contratos/%s/%d/%d"
)

type PNCPContract struct {
	ControlNumber string
	OrgCNPJ       string
	UnitName      string
	Year          int
	Sequence      int
	Kind          string
	Process       string
	Number        string
	SupplierCNPJ  string
	SupplierName  string
	Object        string
	ValueCents    int64
	SignedAt      *time.Time
	PublishedAt   *time.Time
	StartsAt      *time.Time
	EndsAt        *time.Time
}

func (c PNCPContract) URL() string {
	return fmt.Sprintf(pncpContractPage, c.OrgCNPJ, c.Year, c.Sequence)
}

func (c PNCPContract) ProcessKey() string {
	key, err := ParseEntityInput(EntityProcesso, c.Process)
	if err != nil {
		return ""
	}
	return key
}

func MunicipalOrgCNPJs() []string {
	var others []string
	for _, cnpj := range PublicBodyCNPJs() {
		if cnpj != MunicipalityCNPJ && HasValidCheckDigits(cnpj) {
			others = append(others, cnpj)
		}
	}
	sort.Strings(others)
	return append([]string{MunicipalityCNPJ}, others...)
}
