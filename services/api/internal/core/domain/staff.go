package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	SourceStaff    = "tce_pessoal"
	FirstStaffYear = 2024
	staffMonthForm = "2006/01"
)

type StaffRow struct {
	Month             time.Time
	Unit              string
	Situation         string
	Group             string
	Headcount         int
	RemunerationCents int64
}

type tceStaffRecord struct {
	Anomes            string  `json:"Anomes"`
	UnidadeGestora    string  `json:"UnidadeGestora"`
	Quantidade        int     `json:"Quantidade"`
	Remuneracao       float64 `json:"Remuneracao"`
	SituacaoFuncional string  `json:"SituacaoFuncional"`
	Grupo             string  `json:"Grupo"`
}

func ParseStaffJSON(body []byte) ([]StaffRow, error) {
	var payload struct {
		SituacoesFuncionais []tceStaffRecord `json:"SituacoesFuncionais"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("situação funcional do TCE-RJ: %w", err)
	}
	out := make([]StaffRow, 0, len(payload.SituacoesFuncionais))
	for _, r := range payload.SituacoesFuncionais {
		month, err := time.Parse(staffMonthForm, strings.TrimSpace(r.Anomes))
		if err != nil {
			return nil, fmt.Errorf("mês %q do TCE-RJ: %w", r.Anomes, err)
		}
		out = append(out, StaffRow{Month: month, Unit: squeezed(r.UnidadeGestora), Situation: squeezed(r.SituacaoFuncional),
			Group: squeezed(r.Grupo), Headcount: r.Quantidade, RemunerationCents: int64(math.Round(r.Remuneracao * 100))})
	}
	return out, nil
}
