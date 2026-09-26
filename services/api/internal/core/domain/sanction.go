package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	SourceSanctions = "cgu_sancoes"
	RecordSanction  = "sancao"

	RegisterCEIS  = "CEIS"
	RegisterCNEP  = "CNEP"
	RegisterCEPIM = "CEPIM"

	sanctionDateLayout = "02/01/2006"
	legalEntityKind    = "J"
)

var SanctionRegisters = []string{RegisterCEIS, RegisterCNEP, RegisterCEPIM}

type SanctionState string

const (
	SanctionListed   SanctionState = "no_cadastro"
	SanctionEnded    SanctionState = "prazo_encerrado"
	SanctionDelisted SanctionState = "fora_do_cadastro"
)

type Sanction struct {
	Register    string
	Code        string
	CNPJ        string
	Name        string
	Category    string
	StartsAt    *time.Time
	EndsAt      *time.Time
	PublishedAt *time.Time
	Process     string
	Organ       string
	OrganUF     string
	Sphere      string
	Scope       string
	LegalBasis  string
	FineCents   *int64
	FirstSeen   time.Time
	LastSeen    time.Time
}

type SanctionLoad struct {
	Days      map[string]time.Time
	Sanctions []Sanction
}

func (s Sanction) State(listedOn, today time.Time) SanctionState {
	if s.LastSeen.Before(listedOn) {
		return SanctionDelisted
	}
	if s.EndsAt != nil && s.EndsAt.Before(today) {
		return SanctionEnded
	}
	return SanctionListed
}

func (s Sanction) CoversDay(day time.Time) bool {
	if s.StartsAt != nil && day.Before(*s.StartsAt) {
		return false
	}
	return s.EndsAt == nil || !day.After(*s.EndsAt)
}

type SanctionColumns map[string]int

const (
	colRegister   = "CADASTRO"
	colCode       = "CÓDIGO DA SANÇÃO"
	colKind       = "TIPO DE PESSOA"
	colDocument   = "CPF OU CNPJ DO SANCIONADO"
	colName       = "NOME DO SANCIONADO"
	colRegistry   = "RAZÃO SOCIAL - CADASTRO RECEITA"
	colProcess    = "NÚMERO DO PROCESSO"
	colCategory   = "CATEGORIA DA SANÇÃO"
	colFine       = "VALOR DA MULTA"
	colStarts     = "DATA INÍCIO SANÇÃO"
	colEnds       = "DATA FINAL SANÇÃO"
	colPublished  = "DATA PUBLICAÇÃO"
	colScope      = "ABRAGÊNCIA DA SANÇÃO"
	colOrgan      = "ÓRGÃO SANCIONADOR"
	colOrganUF    = "UF ÓRGÃO SANCIONADOR"
	colSphere     = "ESFERA ÓRGÃO SANCIONADOR"
	colLegalBasis = "FUNDAMENTAÇÃO LEGAL"
)

var requiredSanctionColumns = []string{colRegister, colCode, colKind, colDocument, colName, colRegistry, colProcess, colCategory,
	colStarts, colEnds, colPublished, colScope, colOrgan, colOrganUF, colSphere, colLegalBasis}

func NewSanctionColumns(header []string) (SanctionColumns, error) {
	cols := SanctionColumns{}
	for i, name := range header {
		cols[strings.TrimSpace(name)] = i
	}
	for _, name := range requiredSanctionColumns {
		if _, ok := cols[name]; !ok {
			return nil, fmt.Errorf("cabeçalho sem a coluna %q", name)
		}
	}
	return cols, nil
}

func (c SanctionColumns) get(row []string, name string) string {
	i, ok := c[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func (c SanctionColumns) IsCompany(row []string) bool {
	doc := c.get(row, colDocument)
	return c.get(row, colKind) == legalEntityKind && len(doc) == cnpjDigits
}

func (c SanctionColumns) CNPJ(row []string) string {
	return c.get(row, colDocument)
}

func ParseSanctionRow(c SanctionColumns, row []string) (Sanction, error) {
	if len(row) != len(c) {
		return Sanction{}, fieldCountError("sanção", len(c), len(row))
	}
	if !c.IsCompany(row) {
		return Sanction{}, fmt.Errorf("sanção %s não é de pessoa jurídica", c.get(row, colCode))
	}
	s := Sanction{
		Register: c.get(row, colRegister), Code: c.get(row, colCode), CNPJ: c.get(row, colDocument),
		Name: squeezed(c.get(row, colRegistry)), Category: c.get(row, colCategory),
		StartsAt: parseSanctionDate(c.get(row, colStarts)), EndsAt: parseSanctionDate(c.get(row, colEnds)),
		PublishedAt: parseSanctionDate(c.get(row, colPublished)), Process: c.get(row, colProcess),
		Organ: squeezed(c.get(row, colOrgan)), OrganUF: c.get(row, colOrganUF), Sphere: c.get(row, colSphere),
		Scope: c.get(row, colScope), LegalBasis: squeezed(c.get(row, colLegalBasis)),
	}
	if s.Name == "" {
		s.Name = squeezed(c.get(row, colName))
	}
	if s.Register != RegisterCEIS && s.Register != RegisterCNEP {
		return Sanction{}, fmt.Errorf("sanção %s de cadastro desconhecido %q", s.Code, s.Register)
	}
	if fine := c.get(row, colFine); fine != "" {
		cents, err := parseRegistryCents(fine)
		if err != nil {
			return Sanction{}, fmt.Errorf("multa da sanção %s: %w", s.Code, err)
		}
		if cents > 0 {
			s.FineCents = &cents
		}
	}
	return s, nil
}

type SanctionRows interface {
	IsCompany(row []string) bool
	CNPJ(row []string) string
	Parse(row []string) (Sanction, error)
}

func NewSanctionRows(register string, header []string) (SanctionRows, error) {
	if register == RegisterCEPIM {
		return NewCEPIMColumns(header)
	}
	return NewSanctionColumns(header)
}

func (c SanctionColumns) Parse(row []string) (Sanction, error) {
	return ParseSanctionRow(c, row)
}

func parseSanctionDate(s string) *time.Time {
	t, err := time.Parse(sanctionDateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}
