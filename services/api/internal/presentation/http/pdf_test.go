package http

import (
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestPDFFilename(t *testing.T) {
	day := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		g    domain.Gazette
		want string
	}{
		{domain.Gazette{PublishedAt: day, EditionNumber: "1771"}, "diario-sg-2026-09-18-1771.pdf"},
		{domain.Gazette{PublishedAt: day, EditionNumber: "1772", IsExtra: true}, "diario-sg-2026-09-18-1772-extra.pdf"},
		{domain.Gazette{PublishedAt: day}, "diario-sg-2026-09-18-s-n.pdf"},
	}
	for _, c := range cases {
		if got := pdfFilename(c.g); got != c.want {
			t.Errorf("esperava %s, veio %s", c.want, got)
		}
	}
}
