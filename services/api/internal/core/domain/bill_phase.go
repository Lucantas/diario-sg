package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BillPhase string

const (
	PhaseLaw             BillPhase = "virou_lei"
	PhaseVetoed          BillPhase = "vetado"
	PhaseRejected        BillPhase = "rejeitado"
	PhaseWithdrawn       BillPhase = "retirado"
	PhaseArchived        BillPhase = "arquivado"
	PhaseSentToExecutive BillPhase = "enviado_ao_executivo"
	PhaseApproved        BillPhase = "aprovado"
	PhaseVoting          BillPhase = "em_votacao"
	PhaseCommittee       BillPhase = "em_comissao"
	PhasePresented       BillPhase = "apresentado"
)

var BillPhasesInOrder = []BillPhase{PhaseLaw, PhaseVetoed, PhaseRejected, PhaseWithdrawn, PhaseArchived,
	PhaseSentToExecutive, PhaseApproved, PhaseVoting, PhaseCommittee, PhasePresented}

var billPhaseRules = []struct {
	phase BillPhase
	re    *regexp.Regexp
}{
	{PhaseLaw, regexp.MustCompile(`^lei n\S*\s*\d`)},
	{PhaseVetoed, regexp.MustCompile(`\bveto\b|\bvetad[oa]`)},
	{PhaseRejected, regexp.MustCompile(`^(rejeitad|reprovad)[oa]\b`)},
	{PhaseWithdrawn, regexp.MustCompile(`retirad[oa] pel[oa] autor|pedido de retirada`)},
	{PhaseArchived, regexp.MustCompile(`arquivad[oa]`)},
	{PhaseSentToExecutive, regexp.MustCompile(`enviado (para|a|ao) (a )?(prefeitura|executivo|poder executivo)|^ao executivo`)},
	{PhaseApproved, regexp.MustCompile(`^aprovad[oa]\b.*votac`)},
	{PhaseVoting, regexp.MustCompile(`para votacao|ordem do dia`)},
	{PhaseCommittee, regexp.MustCompile(`comissao|relatori|parecer`)},
}

func BillPhaseOf(b Bill) BillPhase {
	if b.LawNumber > 0 {
		return PhaseLaw
	}
	texts := make([]string, 0, len(b.Events)+1)
	for _, e := range b.Events {
		texts = append(texts, foldAccents(e.Text))
	}
	for _, rule := range billPhaseRules {
		if rule.phase == PhaseArchived && foldAccents(b.Status) == "arquivado" {
			return PhaseArchived
		}
		for _, t := range texts {
			if rule.re.MatchString(t) {
				return rule.phase
			}
		}
	}
	return PhasePresented
}

func DaysIdle(b Bill, today time.Time) int {
	var last time.Time
	for _, e := range b.Events {
		if e.At.After(last) {
			last = e.At
		}
	}
	if last.IsZero() || today.Before(last) {
		return 0
	}
	return int(today.Sub(last).Hours() / 24)
}

func ParseBillPhase(s string) (BillPhase, error) {
	p := BillPhase(strings.ToLower(strings.TrimSpace(s)))
	if p == "" {
		return "", nil
	}
	for _, known := range BillPhasesInOrder {
		if p == known {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: fase %q", ErrInvalidInput, s)
}

type BillDocRef struct {
	Kind   string
	Number int
	Year   int
}

var billReferenceRe = regexp.MustCompile(`projeto de? lei( complementar)?[^0-9/]{0,6}0*(\d+)\s*/\s*(\d{2,4})\b`)

func BillReference(author string) (BillDocRef, bool) {
	m := billReferenceRe.FindStringSubmatch(foldAccents(author))
	if m == nil {
		return BillDocRef{}, false
	}
	number, _ := strconv.Atoi(m[2])
	ref := BillDocRef{Kind: "PROJETO DE LEI", Number: number, Year: fullYear(m[3])}
	if m[1] != "" {
		ref.Kind = "PROJETO DE LEI COMPLEMENTAR"
	}
	return ref, number > 0
}
