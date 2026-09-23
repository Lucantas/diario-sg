package http

import (
	"io"
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func (a *API) gazettePDF(w http.ResponseWriter, r *http.Request) {
	g, err := a.PDF.Gazette(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	etag := `"` + g.Checksum + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body, err := a.PDF.Open(r.Context(), g)
	if err != nil {
		w.Header().Del("ETag")
		w.Header().Del("Cache-Control")
		writeError(w, err, a.Log)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+pdfFilename(g)+`"`)
	if _, err := io.Copy(w, body); err != nil {
		a.Log.Warn("envio do pdf interrompido", "gazette", g.ID, "error", err)
	}
}

func pdfFilename(g domain.Gazette) string {
	number := g.EditionNumber
	if number == "" {
		number = "s-n"
	}
	name := "diario-sg-" + g.PublishedAt.Format(time.DateOnly) + "-" + number
	if g.IsExtra {
		name += "-extra"
	}
	return name + ".pdf"
}
