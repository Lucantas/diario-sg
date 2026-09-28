package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const SourceBills = "sicam_processos"

var ErrBillNotFound = errors.New("processo não encontrado")

type BillKey struct{ Number, Year int }

func (k BillKey) String() string { return fmt.Sprintf("%d/%d", k.Number, k.Year) }

func (k BillKey) Slug() string { return fmt.Sprintf("%d-%d", k.Number, k.Year) }

var billKeyRe = regexp.MustCompile(`^\s*0*(\d+)\s*[/_-]\s*(\d{4})\s*$`)

func ParseBillKey(s string) (BillKey, error) {
	m := billKeyRe.FindStringSubmatch(s)
	if m == nil {
		return BillKey{}, fmt.Errorf("%w: processo %q (use número/ano, como 5564/2025)", ErrInvalidInput, s)
	}
	number, _ := strconv.Atoi(m[1])
	year, _ := strconv.Atoi(m[2])
	if number == 0 {
		return BillKey{}, fmt.Errorf("%w: processo %q", ErrInvalidInput, s)
	}
	return BillKey{Number: number, Year: year}, nil
}

type Bill struct {
	Key             BillKey
	Kind            string
	DocLabel        string
	DocNumber       int
	DocYear         int
	Summary         string
	Authors         string
	PresentedOn     *time.Time
	Status          string
	CurrentBody     string
	LastMovement    string
	SourceUpdatedAt *time.Time
	LawNumber       int
	LawYear         int
	LawURL          string
	URL             string
	FetchedAt       time.Time
	Events          []BillEvent
	Opinions        []BillOpinion
}

type BillEvent struct {
	Position int
	At       time.Time
	Label    string
	Text     string
	Sector   string
}

type BillOpinion struct {
	Position   int
	Result     string
	On         *time.Time
	Committee  string
	Rapporteur string
}

func (b Bill) LawLabel() string {
	if b.LawNumber == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", b.LawNumber, b.LawYear)
}

var NormativeBillKinds = []string{
	"PROJETO DE LEI",
	"PROJETO DE LEI COMPLEMENTAR",
	"PROJETO DE LEI SUBSTITUTIVO",
	"PROJETO DE RESOLUÇÃO",
	"PROJETO DE EMENDA À LEI ORGÂNICA",
	"MENSAGEM",
	"EMENDA",
}

func IsNormativeKind(kind string) bool {
	k := strings.ToUpper(strings.TrimSpace(kind))
	for _, n := range NormativeBillKinds {
		if k == n {
			return true
		}
	}
	return false
}
