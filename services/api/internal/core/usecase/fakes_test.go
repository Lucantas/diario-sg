package usecase

import (
	"context"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type memGazettes struct {
	byChecksum map[string]string
	saved      map[string]domain.Gazette
	acts       map[string][]domain.Act
}

func newMemGazettes() *memGazettes {
	return &memGazettes{byChecksum: map[string]string{}, saved: map[string]domain.Gazette{}, acts: map[string][]domain.Act{}}
}
func (m *memGazettes) FindIDByChecksum(_ context.Context, c string) (string, bool, error) {
	id, ok := m.byChecksum[c]
	return id, ok, nil
}
func (m *memGazettes) SaveWithActs(_ context.Context, g *domain.Gazette, acts []domain.Act) error {
	g.ID = "g-" + g.Checksum
	m.byChecksum[g.Checksum] = g.ID
	m.saved[g.ID] = *g
	m.acts[g.ID] = acts
	return nil
}
func (m *memGazettes) FindByID(_ context.Context, id string) (domain.Gazette, error) {
	g, ok := m.saved[id]
	if !ok {
		return g, domain.ErrNotFound
	}
	return g, nil
}
func (m *memGazettes) ListByPeriod(_ context.Context, from, to time.Time) ([]domain.Gazette, error) {
	var out []domain.Gazette
	for _, g := range m.saved {
		if !g.PublishedAt.Before(from) && !g.PublishedAt.After(to) {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PublishedAt.Before(out[j].PublishedAt) })
	return out, nil
}
func (m *memGazettes) ReplaceActs(_ context.Context, id, number string, acts []domain.Act) error {
	g := m.saved[id]
	g.EditionNumber = number
	m.saved[id] = g
	m.acts[id] = acts
	return nil
}

type memStorage struct{}

func (memStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("PDF")), nil
}

type fixedExtractor struct{ text string }

func (f fixedExtractor) Extract(context.Context, io.Reader, string) (string, error) {
	return f.text, nil
}

type lineParser struct{ tag string }

func (p lineParser) Parse(text string) []domain.Act {
	var out []domain.Act
	for i, l := range strings.Split(strings.TrimSpace(text), "\n") {
		out = append(out, domain.Act{Type: domain.ActOutro, Title: p.tag + l, Body: l, Position: i})
	}
	return out
}

type lineParsers struct{}

func (lineParsers) For(source string) ports.ActParser {
	if source == domain.SourceDiarioCamara {
		return lineParser{tag: "câmara: "}
	}
	return lineParser{}
}

type cnpjExtractor struct{}

func (cnpjExtractor) Extract(body string) []domain.Entity {
	if !strings.Contains(body, "CNPJ") {
		return nil
	}
	return []domain.Entity{{Kind: domain.EntityCNPJ, Value: "12.345.678/0001-90", Normalized: "12345678000190"}}
}

func (lineParser) EditionNumber(text string) string {
	if strings.Contains(text, "EDIÇÃO 1771") {
		return "1771"
	}
	return ""
}

type recPublisher struct{ indexed []string }

func (r *recPublisher) GazetteIndexed(_ context.Context, id string, _ int) error {
	r.indexed = append(r.indexed, id)
	return nil
}

type recNotifier struct {
	confirmations int
	matches       int
}

func (r *recNotifier) SendConfirmation(context.Context, domain.Subscription) error {
	r.confirmations++
	return nil
}
func (r *recNotifier) SendMatches(context.Context, domain.Subscription, domain.Gazette, []domain.ActHit) error {
	r.matches++
	return nil
}
