package http

import (
	"fmt"
	"net/http"
	"time"
)

const (
	securityContact     = "https://github.com/Lucantas/diario-sg/security/advisories/new"
	securityTxtValidFor = 180 * 24 * time.Hour
)

func (a *API) securityTxt(now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fmt.Fprintf(w, "Contact: %s\nExpires: %s\nCanonical: %s/.well-known/security.txt\nPreferred-Languages: pt, en\n",
			securityContact, now().UTC().Add(securityTxtValidFor).Format(time.RFC3339), a.issuer())
	})
}
