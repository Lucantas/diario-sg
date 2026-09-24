package postgres

import (
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestReportActsLimitIsSmallerForCNPJ(t *testing.T) {
	for kind, want := range map[domain.EntityKind]int{domain.EntityCNPJ: 100, domain.EntityProcesso: 300, domain.EntityContrato: 300} {
		if got := reportActsLimit(kind); got != want {
			t.Errorf("%s: esperava %d, veio %d", kind, want, got)
		}
	}
}
