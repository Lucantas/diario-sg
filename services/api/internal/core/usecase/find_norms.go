package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const maxNormsFound = 50

type FindNorms struct{ reader ports.NormReader }

func NewFindNorms(reader ports.NormReader) *FindNorms { return &FindNorms{reader: reader} }

type NormQuery struct {
	Kind, Number, Text, Theme string
}

func (uc *FindNorms) Execute(ctx context.Context, q NormQuery) ([]domain.Norm, error) {
	kind, number, text := q.Kind, strings.TrimSpace(q.Number), strings.TrimSpace(q.Text)
	theme, err := domain.ParseTheme(q.Theme)
	if err != nil {
		return nil, err
	}
	var k domain.NormKind
	if strings.TrimSpace(kind) != "" || number != "" {
		var err error
		if k, err = domain.ParseNormKind(kind); err != nil {
			return nil, err
		}
	}
	switch {
	case number != "":
		n, year, suffix, err := domain.ParseNormNumber(number)
		if err != nil {
			return nil, err
		}
		norms, err := uc.reader.NormsByNumber(ctx, k, n, year)
		return withSuffix(norms, suffix), err
	case text != "" || !theme.IsZero():
		return uc.reader.SearchNorms(ctx, k, text, theme, maxNormsFound)
	default:
		return nil, fmt.Errorf("%w: informe tipo e número, ou um texto ou um tema para buscar na ementa e no autor", domain.ErrInvalidInput)
	}
}

func withSuffix(norms []domain.Norm, suffix string) []domain.Norm {
	if suffix == "" {
		return norms
	}
	out := []domain.Norm{}
	for _, n := range norms {
		if n.Suffix == suffix {
			out = append(out, n)
		}
	}
	return out
}
