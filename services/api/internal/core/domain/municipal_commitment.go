package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const SourceMunicipalCommitments = "pmsg_empenhos"

type MunicipalEntity struct {
	ID   int
	Name string
}

type MunicipalCommitment struct {
	EntityID        int
	Entity          string
	Year            int
	CommitmentID    int64
	Number          string
	Date            time.Time
	CNPJ            string
	Name            string
	Object          string
	ProcessKind     string
	Process         string
	Modality        string
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type MunicipalTotal struct {
	Year            int
	EntityID        int
	Entity          string
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type portalEntity struct {
	ID     string `json:"id_entidade"`
	Name   string `json:"ds_entidade"`
	Active string `json:"ativo"`
}

func ParseMunicipalEntities(body []byte) ([]MunicipalEntity, error) {
	var rows []portalEntity
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("entidades do portal: %w", err)
	}
	var out []MunicipalEntity
	for _, r := range rows {
		if r.Active != "1" {
			continue
		}
		id, err := strconv.Atoi(r.ID)
		if err != nil {
			return nil, fmt.Errorf("entidade %q: %w", r.ID, err)
		}
		out = append(out, MunicipalEntity{ID: id, Name: squeezed(r.Name)})
	}
	return out, nil
}

type portalCommitment struct {
	Name           string `json:"nome_razao"`
	Document       string `json:"documento_formatado"`
	Object         string `json:"objeto"`
	ProcessKind    string `json:"ds_tp_processo"`
	ProcessNumber  string `json:"nr_processo"`
	ProcessYear    string `json:"ano_processo"`
	Modality       string `json:"ds_tp_modalidade"`
	ModalityNumber string `json:"nr_modalidade"`
	ModalityYear   string `json:"ano_modalidade"`
	Number         string `json:"nr_empenho"`
	ID             string `json:"id_empenho"`
	Date           string `json:"dt_empenho"`
	Committed      string `json:"vl_empenhado_acu"`
	Liquidated     string `json:"vl_liquidado_acu"`
	Paid           string `json:"vl_pago_acu"`
}

type portalCommitments struct {
	Commitments []portalCommitment `json:"empenhos"`
	Totals      struct {
		Committed  string `json:"total_empenhado_acu"`
		Liquidated string `json:"total_liquidado_acu"`
		Paid       string `json:"total_pago_acu"`
	} `json:"totais"`
}

func ParseMunicipalCommitments(body []byte, year int, entity MunicipalEntity) ([]MunicipalCommitment, MunicipalTotal, error) {
	var page portalCommitments
	total := MunicipalTotal{Year: year, EntityID: entity.ID, Entity: entity.Name}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, total, fmt.Errorf("empenhos de %d da entidade %d: %w", year, entity.ID, err)
	}
	var err error
	if total.CommittedCents, total.LiquidatedCents, total.PaidCents, err = portalValues(page.Totals.Committed, page.Totals.Liquidated, page.Totals.Paid); err != nil {
		return nil, total, fmt.Errorf("totais de %d da entidade %d: %w", year, entity.ID, err)
	}
	var out []MunicipalCommitment
	for _, r := range page.Commitments {
		cnpj, ok := NormalizeCNPJ(r.Document)
		if !ok || !HasValidCheckDigits(cnpj) {
			continue
		}
		c, err := r.toCommitment(year, entity, cnpj)
		if err != nil {
			return nil, total, fmt.Errorf("empenho %q de %d da entidade %d: %w", r.ID, year, entity.ID, err)
		}
		out = append(out, c)
	}
	return out, total, nil
}

func (r portalCommitment) toCommitment(year int, entity MunicipalEntity, cnpj string) (MunicipalCommitment, error) {
	id, err := strconv.ParseInt(r.ID, 10, 64)
	if err != nil {
		return MunicipalCommitment{}, err
	}
	day := dayPtr(&r.Date)
	if day == nil {
		return MunicipalCommitment{}, fmt.Errorf("data %q", r.Date)
	}
	c := MunicipalCommitment{EntityID: entity.ID, Entity: entity.Name, Year: year, CommitmentID: id, Number: squeezed(r.Number), Date: *day, CNPJ: cnpj,
		Name: squeezed(r.Name), Object: squeezed(r.Object), ProcessKind: squeezed(r.ProcessKind), Process: numberYear(r.ProcessNumber, r.ProcessYear),
		Modality: strings.TrimSpace(squeezed(r.Modality) + " " + numberYear(r.ModalityNumber, r.ModalityYear))}
	c.CommittedCents, c.LiquidatedCents, c.PaidCents, err = portalValues(r.Committed, r.Liquidated, r.Paid)
	return c, err
}

func numberYear(number, year string) string {
	number, year = strings.TrimSpace(number), strings.TrimSpace(year)
	if number == "" || number == "-" {
		return ""
	}
	if year == "" || year == "-" {
		return number
	}
	return number + "/" + year
}

func portalValues(committed, liquidated, paid string) (int64, int64, int64, error) {
	var out [3]int64
	for i, s := range []string{committed, liquidated, paid} {
		v, err := portalCents(s)
		if err != nil {
			return 0, 0, 0, err
		}
		out[i] = v
	}
	return out[0], out[1], out[2], nil
}

func portalCents(s string) (int64, error) {
	clean := strings.Join(strings.Fields(strings.ReplaceAll(s, "R$", "")), "")
	if clean == "" {
		return 0, nil
	}
	return federalCents(clean)
}

type MunicipalYear struct {
	Year           int
	Commitments    int
	CommittedCents int64
	PaidCents      int64
}

type MunicipalSupplier struct {
	Years  []MunicipalYear
	Recent []MunicipalCommitment
}

func WithoutRepeatedCommitments(rows []MunicipalCommitment) ([]MunicipalCommitment, int) {
	seen := make(map[int64]bool, len(rows))
	out := make([]MunicipalCommitment, 0, len(rows))
	for _, r := range rows {
		if seen[r.CommitmentID] {
			continue
		}
		seen[r.CommitmentID] = true
		out = append(out, r)
	}
	return out, len(rows) - len(out)
}
