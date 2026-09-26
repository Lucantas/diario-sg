package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	SourceAmendments       = "cgu_emendas"
	SourceTransfers        = "cgu_transferencias"
	RecordAmendmentPayment = "pagamento_emenda"
	SaoGoncaloIBGE         = "3304904"
	saoGoncaloUF           = "RJ"
	federalMonthLayout     = "200601"
	companyFavoredType     = "Pessoa Jurídica"
	noAmendmentInfo        = "Sem informação"
)

type Amendment struct {
	Code            string
	Year            int
	Kind            string
	Author          string
	Number          string
	Function        string
	Subfunction     string
	Program         string
	Action          string
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type AmendmentPayment struct {
	Code        string
	Author      string
	Kind        string
	Month       time.Time
	CNPJ        string
	Name        string
	LegalNature string
	ValueCents  int64
}

type FederalTransfer struct {
	Month      time.Time
	Kind       string
	Organ      string
	Function   string
	Program    string
	Action     string
	Label      string
	CNPJ       string
	Name       string
	ValueCents int64
}

type TransferTotal struct {
	Year       int
	Kind       string
	Function   string
	ValueCents int64
}

type CSVColumns map[string]int

func NewCSVColumns(header []string, required ...string) (CSVColumns, error) {
	cols := CSVColumns{}
	for i, name := range header {
		cols[strings.TrimSpace(strings.TrimPrefix(name, byteOrderMark))] = i
	}
	for _, name := range required {
		if _, ok := cols[name]; !ok {
			return nil, fmt.Errorf("cabeçalho sem a coluna %q", name)
		}
	}
	return cols, nil
}

func (c CSVColumns) Get(row []string, name string) string {
	i, ok := c[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

var (
	AmendmentColumns = []string{"Código da Emenda", "Ano da Emenda", "Tipo de Emenda", "Nome do Autor da Emenda", "Número da emenda",
		"Código Município IBGE", "Nome Função", "Nome Subfunção", "Nome Programa", "Nome Ação", "Valor Empenhado", "Valor Liquidado", "Valor Pago"}
	AmendmentPaymentColumns = []string{"Código da Emenda", "Nome do Autor da Emenda", "Tipo de Emenda", "Ano/Mês", "Código do Favorecido",
		"Favorecido", "Natureza Jurídica", "Tipo Favorecido", "UF Favorecido", "Município Favorecido", "Valor Recebido"}
	TransferColumns = []string{"ANO / MÊS", "TIPO TRANSFERÊNCIA", "UF", "NOME MUNICÍPIO", "NOME ÓRGÃO", "NOME FUNÇÃO", "NOME PROGRAMA",
		"NOME AÇÃO", "LINGUAGEM CIDADÃ", "CÓDIGO FAVORECIDO", "NOME FAVORECIDO", "VALOR TRANSFERIDO"}
)

func IsSaoGoncaloAmendment(c CSVColumns, row []string) bool {
	return c.Get(row, "Código Município IBGE") == SaoGoncaloIBGE
}

func ParseAmendment(c CSVColumns, row []string) (Amendment, error) {
	a := Amendment{Code: orEmptyInfo(c.Get(row, "Código da Emenda")), Kind: squeezed(c.Get(row, "Tipo de Emenda")),
		Author: orEmptyInfo(squeezed(c.Get(row, "Nome do Autor da Emenda"))), Number: orEmptyInfo(c.Get(row, "Número da emenda")),
		Function: squeezed(c.Get(row, "Nome Função")), Subfunction: squeezed(c.Get(row, "Nome Subfunção")),
		Program: squeezed(c.Get(row, "Nome Programa")), Action: squeezed(c.Get(row, "Nome Ação"))}
	if _, err := fmt.Sscan(c.Get(row, "Ano da Emenda"), &a.Year); err != nil {
		return a, fmt.Errorf("ano da emenda %q: %w", c.Get(row, "Ano da Emenda"), err)
	}
	var err error
	if a.CommittedCents, err = federalCents(c.Get(row, "Valor Empenhado")); err != nil {
		return a, err
	}
	if a.LiquidatedCents, err = federalCents(c.Get(row, "Valor Liquidado")); err != nil {
		return a, err
	}
	a.PaidCents, err = federalCents(c.Get(row, "Valor Pago"))
	return a, err
}

func IsSaoGoncaloCompanyPayment(c CSVColumns, row []string) bool {
	return c.Get(row, "UF Favorecido") == saoGoncaloUF && foldText(c.Get(row, "Município Favorecido")) == saoGoncaloTCEFolded &&
		c.Get(row, "Tipo Favorecido") == companyFavoredType
}

func ParseAmendmentPayment(c CSVColumns, row []string) (AmendmentPayment, error) {
	cnpj, ok := NormalizeCNPJ(c.Get(row, "Código do Favorecido"))
	if !ok {
		return AmendmentPayment{}, fmt.Errorf("favorecido sem CNPJ: %q", c.Get(row, "Código do Favorecido"))
	}
	month, err := time.Parse(federalMonthLayout, c.Get(row, "Ano/Mês"))
	if err != nil {
		return AmendmentPayment{}, fmt.Errorf("mês %q: %w", c.Get(row, "Ano/Mês"), err)
	}
	value, err := federalCents(c.Get(row, "Valor Recebido"))
	return AmendmentPayment{Code: orEmptyInfo(c.Get(row, "Código da Emenda")), Author: orEmptyInfo(squeezed(c.Get(row, "Nome do Autor da Emenda"))),
		Kind: squeezed(c.Get(row, "Tipo de Emenda")), Month: month, CNPJ: cnpj, Name: squeezed(c.Get(row, "Favorecido")),
		LegalNature: squeezed(c.Get(row, "Natureza Jurídica")), ValueCents: value}, err
}

func IsSaoGoncaloTransfer(c CSVColumns, row []string) bool {
	return c.Get(row, "UF") == saoGoncaloUF && foldText(c.Get(row, "NOME MUNICÍPIO")) == saoGoncaloTCEFolded
}

func ParseTransfer(c CSVColumns, row []string) (FederalTransfer, bool, error) {
	cnpj, company := NormalizeCNPJ(c.Get(row, "CÓDIGO FAVORECIDO"))
	if !company {
		return FederalTransfer{}, false, nil
	}
	month, err := time.Parse(federalMonthLayout, c.Get(row, "ANO / MÊS"))
	if err != nil {
		return FederalTransfer{}, true, fmt.Errorf("mês %q: %w", c.Get(row, "ANO / MÊS"), err)
	}
	value, err := federalCents(c.Get(row, "VALOR TRANSFERIDO"))
	return FederalTransfer{Month: month, Kind: squeezed(c.Get(row, "TIPO TRANSFERÊNCIA")), Organ: squeezed(c.Get(row, "NOME ÓRGÃO")),
		Function: squeezed(c.Get(row, "NOME FUNÇÃO")), Program: squeezed(c.Get(row, "NOME PROGRAMA")), Action: squeezed(c.Get(row, "NOME AÇÃO")),
		Label: squeezed(c.Get(row, "LINGUAGEM CIDADÃ")), CNPJ: cnpj, Name: squeezed(c.Get(row, "NOME FAVORECIDO")), ValueCents: value}, true, err
}

func federalCents(s string) (int64, error) {
	cents, err := parseRegistryCents(strings.ReplaceAll(s, ".", ""))
	if err != nil {
		return 0, fmt.Errorf("valor %q: %w", s, err)
	}
	return cents, nil
}

func orEmptyInfo(s string) string {
	if s == "" || s == "S/I" {
		return noAmendmentInfo
	}
	return s
}
