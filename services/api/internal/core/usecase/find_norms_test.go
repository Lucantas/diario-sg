package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeNormReader struct {
	byNumber []domain.Norm
	searched string
	kind     domain.NormKind
}

func (f *fakeNormReader) NormsByNumber(_ context.Context, kind domain.NormKind, number, year int) ([]domain.Norm, error) {
	f.kind = kind
	return f.byNumber, nil
}

func (f *fakeNormReader) SearchNorms(_ context.Context, kind domain.NormKind, text string, _ int) ([]domain.Norm, error) {
	f.kind, f.searched = kind, text
	return []domain.Norm{}, nil
}

func TestFindNormsByNumberFiltersTheSuffix(t *testing.T) {
	r := &fakeNormReader{byNumber: []domain.Norm{{Number: 57, Year: 1955, Suffix: "A"}, {Number: 57, Year: 1955, Suffix: "B"}}}

	got, err := NewFindNorms(r).Execute(context.Background(), "lei", "057/1955 b", "")

	if err != nil || len(got) != 1 || got[0].Suffix != "B" || r.kind != domain.NormLaw {
		t.Errorf("veio %+v %v", got, err)
	}
}

func TestFindNormsSearchesTextWithOptionalKind(t *testing.T) {
	r := &fakeNormReader{}

	if _, err := NewFindNorms(r).Execute(context.Background(), "", "", " subsídio "); err != nil || r.searched != "subsídio" || r.kind != "" {
		t.Errorf("busca: %v %q %q", err, r.searched, r.kind)
	}
	for _, in := range [][3]string{{"", "", ""}, {"", "1/2020", ""}, {"portaria", "", "x"}, {"lei", "abc", ""}} {
		if _, err := NewFindNorms(r).Execute(context.Background(), in[0], in[1], in[2]); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%v aceito: %v", in, err)
		}
	}
}
