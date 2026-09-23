package pmsg

import (
	"net/url"
	"time"
)

func EditionURL(base *url.URL, published time.Time) string {
	rel := &url.URL{Path: "diario/" + published.Format("2006_01_02") + ".pdf"}
	return base.ResolveReference(rel).String()
}
