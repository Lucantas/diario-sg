package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type recEntityReader struct {
	kind domain.EntityKind
	key  string
}

func (r *recEntityReader) ReportByKey(_ context.Context, kind domain.EntityKind, key string) (domain.EntityReport, error) {
	r.kind, r.key = kind, key
	return domain.EntityReport{Kind: kind, Key: key}, nil
}

func TestGetEntityNormalizesTheInput(t *testing.T) {
	reader := &recEntityReader{}
	uc := NewGetEntity(reader)

	got, err := uc.Execute(context.Background(), domain.EntityContrato, "001 / 2017")

	if err != nil || reader.kind != domain.EntityContrato || reader.key != "1/2017" || got.Key != "1/2017" {
		t.Fatalf("veio %+v %v; leitor recebeu %s %q", got, err, reader.kind, reader.key)
	}
}

func TestGetEntityRejectsInvalidInput(t *testing.T) {
	reader := &recEntityReader{}
	uc := NewGetEntity(reader)

	for _, c := range []struct {
		kind domain.EntityKind
		in   string
		want error
	}{{domain.EntityProcesso, "12", domain.ErrInvalidInput}, {domain.EntityValor, "100", domain.ErrInvalidInput}, {domain.EntityCNPJ, "1", domain.ErrInvalidCNPJ}} {
		if _, err := uc.Execute(context.Background(), c.kind, c.in); !errors.Is(err, c.want) {
			t.Errorf("%s %q: esperava %v, veio %v", c.kind, c.in, c.want, err)
		}
	}
	if reader.key != "" {
		t.Fatal("entrada inválida não deveria chegar ao leitor")
	}
}
