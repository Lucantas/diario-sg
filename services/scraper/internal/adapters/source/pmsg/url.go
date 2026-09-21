package pmsg

import (
	"net/url"
	"time"
)

// EditionURL monta a URL determinística de uma edição no site da prefeitura:
// https://do.pmsg.rj.gov.br/diario/AAAA_MM_DD.pdf. O site não numera as
// edições na URL; o número só aparece dentro do PDF.
func EditionURL(base *url.URL, published time.Time) string {
	rel := &url.URL{Path: "diario/" + published.Format("2006_01_02") + ".pdf"}
	return base.ResolveReference(rel).String()
}
