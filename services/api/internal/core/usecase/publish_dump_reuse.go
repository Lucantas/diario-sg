package usecase

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	actsTable     = "acts"
	entitiesTable = "act_entities"
)

type mergedTable struct {
	header []string
	rows   [][]string
}

func (m mergedTable) writeCSV(w io.Writer) (int, error) {
	out := csv.NewWriter(w)
	if err := out.Write(m.header); err != nil {
		return 0, err
	}
	for i, r := range m.rows {
		if err := out.Write(r); err != nil {
			return i, err
		}
	}
	out.Flush()
	return len(m.rows), out.Error()
}

type rowsByGazette map[string][][]string

type previousTables struct {
	actsHeader, entitiesHeader []string
	acts, entities             rowsByGazette
}

func (uc *PublishDump) reuse(ctx context.Context, snap ports.DumpSnapshot) (map[string]mergedTable, domain.DumpReport, error) {
	if uc.previous == nil {
		return nil, domain.DumpReport{FallbackCause: "dump anterior não configurado"}, nil
	}
	prev, cause, err := uc.loadPrevious(ctx, snap)
	if err != nil || cause != "" {
		return nil, domain.DumpReport{FallbackCause: cause}, err
	}
	fingerprints, err := snap.Fingerprints(ctx)
	if err != nil {
		return nil, domain.DumpReport{}, fmt.Errorf("impressões digitais: %w", err)
	}
	stale := staleGazettes(fingerprints, prev)
	freshActs, freshEntities, err := fetchStale(ctx, snap, stale)
	if err != nil {
		return nil, domain.DumpReport{}, err
	}
	report := domain.DumpReport{Incremental: true, Gazettes: len(fingerprints), FetchedAgain: len(stale)}
	return map[string]mergedTable{
		actsTable:     merge(prev.actsHeader, fingerprints, stale, prev.acts, freshActs),
		entitiesTable: merge(prev.entitiesHeader, fingerprints, stale, prev.entities, freshEntities),
	}, report, nil
}

func (uc *PublishDump) loadPrevious(ctx context.Context, snap ports.DumpSnapshot) (previousTables, string, error) {
	actsHeader, acts, cause, err := uc.previousTable(ctx, snap, actsTable)
	if err != nil || cause != "" {
		return previousTables{}, cause, err
	}
	entitiesHeader, entities, cause, err := uc.previousTable(ctx, snap, entitiesTable)
	if err != nil || cause != "" {
		return previousTables{}, cause, err
	}
	actID, gazetteID := slices.Index(actsHeader, "id"), slices.Index(actsHeader, "gazette_id")
	entityActID := slices.Index(entitiesHeader, "act_id")
	if actID < 0 || gazetteID < 0 || entityActID < 0 {
		return previousTables{}, "cabeçalho sem id, gazette_id ou act_id", nil
	}
	prev := previousTables{actsHeader: actsHeader, entitiesHeader: entitiesHeader, acts: rowsByGazette{}, entities: rowsByGazette{}}
	gazetteOfAct := make(map[string]string, len(acts))
	for _, r := range acts {
		prev.acts[r[gazetteID]] = append(prev.acts[r[gazetteID]], r)
		gazetteOfAct[r[actID]] = r[gazetteID]
	}
	for _, r := range entities {
		if g, ok := gazetteOfAct[r[entityActID]]; ok {
			prev.entities[g] = append(prev.entities[g], r)
		}
	}
	return prev, "", nil
}

func (uc *PublishDump) previousTable(ctx context.Context, snap ports.DumpSnapshot, table string) ([]string, [][]string, string, error) {
	header, rows, err := uc.previous.Table(ctx, table)
	if errors.Is(err, domain.ErrNoPreviousDump) {
		return nil, nil, "sem dump anterior", nil
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("dump anterior, tabela %s: %w", table, err)
	}
	current, err := snap.Header(table)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cabeçalho de %s: %w", table, err)
	}
	if !slices.Equal(header, current) {
		return nil, nil, "cabeçalho de " + table + " mudou", nil
	}
	return header, rows, "", nil
}

func staleGazettes(fingerprints []domain.GazetteFingerprint, prev previousTables) []string {
	var stale []string
	for _, f := range fingerprints {
		if domain.FingerprintOf(prev.acts[f.GazetteID], prev.entities[f.GazetteID]) != f.Hash {
			stale = append(stale, f.GazetteID)
		}
	}
	return stale
}

func fetchStale(ctx context.Context, snap ports.DumpSnapshot, stale []string) (rowsByGazette, rowsByGazette, error) {
	if len(stale) == 0 {
		return rowsByGazette{}, rowsByGazette{}, nil
	}
	acts, err := fetchByGazette(ctx, snap, actsTable, stale)
	if err != nil {
		return nil, nil, err
	}
	entities, err := fetchByGazette(ctx, snap, entitiesTable, stale)
	return acts, entities, err
}

func fetchByGazette(ctx context.Context, snap ports.DumpSnapshot, table string, ids []string) (rowsByGazette, error) {
	records, err := snap.Records(ctx, table, ids)
	if err != nil {
		return nil, fmt.Errorf("tabela %s, edições alteradas: %w", table, err)
	}
	grouped := rowsByGazette{}
	for _, r := range records {
		grouped[r.GazetteID] = append(grouped[r.GazetteID], r.Fields)
	}
	return grouped, nil
}

func merge(header []string, order []domain.GazetteFingerprint, stale []string, previous, fresh rowsByGazette) mergedTable {
	isStale := make(map[string]bool, len(stale))
	for _, id := range stale {
		isStale[id] = true
	}
	m := mergedTable{header: header}
	for _, f := range order {
		source := previous
		if isStale[f.GazetteID] {
			source = fresh
		}
		m.rows = append(m.rows, source[f.GazetteID]...)
	}
	return m
}
