package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var exportHeader = []string{"data", "edicao", "extra", "tipo", "orgao", "orgao_nome", "titulo", "pagina_inicio", "pagina_fim",
	"valores_reais", "cnpjs", "pdf_original", "pdf_arquivado", "sha256_pdf", "avisos", "texto", "fonte"}

const utf8BOM = "\xef\xbb\xbf"

func (a *API) exportActs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		writeError(w, domain.ErrInvalidFilter, a.Log)
		return
	}
	f, ok := a.filterFromQuery(w, r)
	if !ok {
		return
	}
	out := &exportWriter{format: format, w: w, base: a.PublicWebURL}
	err := a.Export.Execute(r.Context(), f, out.write)
	if err == nil {
		err = out.close()
	}
	if err != nil && !out.started {
		writeError(w, err, a.Log)
		return
	}
	if err != nil {
		a.Log.Warn("exportação interrompida", "error", err)
	}
}

type exportWriter struct {
	format  string
	w       http.ResponseWriter
	base    string
	started bool
	rows    int
	csv     *csv.Writer
}

func (e *exportWriter) start(total int) error {
	e.started = true
	h := e.w.Header()
	h.Set("Content-Disposition", `attachment; filename="diario-sg-busca-`+time.Now().Format(time.DateOnly)+"."+e.format+`"`)
	h.Set("X-Total-Count", strconv.Itoa(total))
	h.Set("X-Export-Truncated", strconv.FormatBool(total > domain.ExportLimit))
	if e.format == "json" {
		h.Set("Content-Type", "application/json; charset=utf-8")
		_, err := fmt.Fprintf(e.w, `{"total":%d,"truncated":%t,"items":[`, total, total > domain.ExportLimit)
		return err
	}
	h.Set("Content-Type", "text/csv; charset=utf-8")
	if _, err := io.WriteString(e.w, utf8BOM); err != nil {
		return err
	}
	e.csv = csv.NewWriter(e.w)
	e.csv.Comma = ';'
	e.csv.UseCRLF = true
	return e.csv.Write(exportHeader)
}

func (e *exportWriter) write(h domain.ActHit, total int) error {
	if !e.started {
		if err := e.start(total); err != nil {
			return err
		}
	}
	e.rows++
	if e.format == "csv" {
		return e.csv.Write(csvRecord(e.base, h))
	}
	if e.rows > 1 {
		if _, err := io.WriteString(e.w, ","); err != nil {
			return err
		}
	}
	return json.NewEncoder(e.w).Encode(exportItemDTO{actHitDTO: toHitDTO(h), Body: h.Body, ArchivedPDFURL: archivedURL(e.base, h)})
}

func (e *exportWriter) close() error {
	if !e.started {
		if err := e.start(0); err != nil {
			return err
		}
	}
	if e.format == "json" {
		_, err := io.WriteString(e.w, "]}\n")
		return err
	}
	e.csv.Flush()
	return e.csv.Error()
}

type exportItemDTO struct {
	actHitDTO
	Snippet        string `json:"snippet,omitempty"`
	Body           string `json:"body"`
	ArchivedPDFURL string `json:"archived_pdf_url"`
}

func csvRecord(base string, h domain.ActHit) []string {
	values := make([]string, 0, len(h.ValuesCents))
	for _, c := range h.ValuesCents {
		values = append(values, reaisBR(c))
	}
	extra := "não"
	if h.IsExtra {
		extra = "sim"
	}
	return []string{
		h.PublishedAt.Format(time.DateOnly), h.EditionNumber, extra, string(h.Type), h.Organ, csvCell(domain.OrganName(h.Organ)),
		csvCell(h.Title), pageText(h.PageStart), pageText(h.PageEnd), strings.Join(values, " | "), strings.Join(h.CNPJs, " | "),
		h.SourceURL + pageSuffix(h), archivedURL(base, h), h.Checksum,
		strings.Join(domain.ActWarnings(h.WarningFacts()), " | "), csvCell(h.Body),
		domain.SourceOrDefault(h.Source),
	}
}

func csvCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func reaisBR(cents int64) string {
	return fmt.Sprintf("%d,%02d", cents/100, cents%100)
}

func pageText(p int) string {
	if p <= 0 {
		return ""
	}
	return strconv.Itoa(p)
}

func pageSuffix(h domain.ActHit) string {
	if h.PageStart <= 0 {
		return ""
	}
	return "#page=" + strconv.Itoa(h.PageStart)
}

func archivedURL(base string, h domain.ActHit) string {
	return strings.TrimRight(base, "/") + "/api/v1/gazettes/" + h.GazetteID + "/pdf" + pageSuffix(h)
}
