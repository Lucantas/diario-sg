package pmsg

import (
	"net/url"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

func editionFor(base string, y int, m time.Month, d int) domain.Edition {
	u, _ := url.Parse(base + "/")
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return domain.Edition{PublishedAt: day, URL: EditionURL(u, day)}
}
