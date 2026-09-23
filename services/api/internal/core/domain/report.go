package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type ReportKind string

const (
	ReportWrongText  ReportKind = "texto_errado"
	ReportWrongType  ReportKind = "tipo_errado"
	ReportWrongOrgan ReportKind = "orgao_errado"
	ReportWrongPage  ReportKind = "pagina_errada"
	ReportOther      ReportKind = "outro"
)

var validReportKinds = map[ReportKind]bool{
	ReportWrongText: true, ReportWrongType: true, ReportWrongOrgan: true, ReportWrongPage: true, ReportOther: true,
}

type ReportStatus string

const (
	ReportOpen      ReportStatus = "aberto"
	ReportResolved  ReportStatus = "resolvido"
	ReportDiscarded ReportStatus = "descartado"
)

func (s ReportStatus) ClosesReport() bool { return s == ReportResolved || s == ReportDiscarded }

func (s ReportStatus) Valid() bool { return s == ReportOpen || s.ClosesReport() }

const (
	maxReportTitleRunes   = 500
	maxReportMessageRunes = 2000
)

var ErrInvalidReport = errors.New("reporte inválido: confira o tipo do problema e o tamanho da descrição")

type ErrorReport struct {
	ID        string
	GazetteID string
	Position  int
	ActTitle  string
	Kind      ReportKind
	Message   string
	Status    ReportStatus
	CreatedAt time.Time
	ClosedAt  time.Time

	EditionNumber string
	PublishedAt   time.Time
	PageStart     int
}

func NewErrorReport(gazetteID string, position int, title string, kind ReportKind, message string) (ErrorReport, error) {
	title, message = strings.TrimSpace(title), strings.TrimSpace(message)
	switch {
	case gazetteID == "", position < 0, title == "", utf8.RuneCountInString(title) > maxReportTitleRunes:
		return ErrorReport{}, ErrInvalidReport
	case !validReportKinds[kind], kind == ReportOther && message == "":
		return ErrorReport{}, ErrInvalidReport
	case utf8.RuneCountInString(message) > maxReportMessageRunes:
		return ErrorReport{}, ErrInvalidReport
	}
	return ErrorReport{GazetteID: gazetteID, Position: position, ActTitle: title, Kind: kind, Message: message, Status: ReportOpen}, nil
}
