package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadSanctions struct {
	src  ports.SanctionSource
	repo ports.SanctionRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadSanctions(src ports.SanctionSource, repo ports.SanctionRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadSanctions {
	return &LoadSanctions{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

type sanctionsLoading struct {
	bases map[string]bool
	load  domain.SanctionLoad
	run   *domain.FetchRun
}

func (uc *LoadSanctions) Execute(ctx context.Context) (domain.FetchRun, error) {
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceSanctions, StartedAt: uc.now()}
	load, err := uc.load(ctx, &run)
	if len(load.Days) > 0 {
		if saveErr := uc.repo.Save(ctx, load); saveErr != nil {
			err = errors.Join(err, saveErr)
		}
	}
	run.FinishedAt = uc.now()
	if err != nil {
		run.Error = err.Error()
	}
	if saveErr := uc.runs.Save(ctx, run); saveErr != nil && err == nil {
		err = saveErr
	}
	return run, err
}

func (uc *LoadSanctions) load(ctx context.Context, run *domain.FetchRun) (domain.SanctionLoad, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return domain.SanctionLoad{}, fmt.Errorf("tabela das sanções: %w", err)
	}
	cited, err := uc.repo.CitedCNPJs(ctx)
	if err != nil {
		return domain.SanctionLoad{}, err
	}
	l := &sanctionsLoading{bases: map[string]bool{}, load: domain.SanctionLoad{Days: map[string]time.Time{}}, run: run}
	for _, cnpj := range cited {
		l.bases[domain.CNPJBase(cnpj)] = true
	}
	var failures []error
	for _, register := range domain.SanctionRegisters {
		stored, found, failed := len(l.load.Sanctions), run.Found, run.Failed
		if err := uc.loadRegister(ctx, register, l); err != nil {
			failures = append(failures, err)
			l.load.Sanctions = l.load.Sanctions[:stored]
			run.Found, run.Failed = found, failed
			delete(l.load.Days, register)
		}
	}
	run.Stored = len(l.load.Sanctions)
	run.Skipped = run.Found - run.Stored - run.Failed
	return l.load, errors.Join(failures...)
}

func (uc *LoadSanctions) loadRegister(ctx context.Context, register string, l *sanctionsLoading) error {
	day, err := uc.src.LatestDay(ctx, register)
	if err != nil {
		return err
	}
	l.load.Days[register] = day
	if l.run.RequestedFrom.IsZero() || day.Before(l.run.RequestedFrom) {
		l.run.RequestedFrom = day
	}
	if day.After(l.run.RequestedTo) {
		l.run.RequestedTo = day
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	w := csv.NewWriter(gz)
	w.Comma = ';'
	var cols domain.SanctionRows
	rows := 0
	sum, err := uc.src.Rows(ctx, register, day, func(header, row []string) error {
		if cols == nil {
			parsed, err := domain.NewSanctionRows(register, header)
			if err != nil {
				return err
			}
			cols = parsed
			if err := w.Write(header); err != nil {
				return err
			}
		}
		if !cols.IsCompany(row) {
			return nil
		}
		l.run.Found++
		if !l.bases[domain.CNPJBase(cols.CNPJ(row))] {
			return nil
		}
		rows++
		l.add(cols, row, day)
		return w.Write(row)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", register, err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := uc.raw.Put(ctx, rawSanctionsPath(day, register+".csv.gz"), "application/gzip", &buf); err != nil {
		return err
	}
	manifest, err := json.MarshalIndent(archivedFile{SourceSHA256: sum, Rows: rows}, "", "  ")
	if err != nil {
		return err
	}
	return uc.raw.Put(ctx, rawSanctionsPath(day, register+".manifest.json"), "application/json", bytes.NewReader(manifest))
}

func (l *sanctionsLoading) add(cols domain.SanctionRows, row []string, day time.Time) {
	s, err := cols.Parse(row)
	if err != nil {
		l.run.Failed++
		return
	}
	s.FirstSeen, s.LastSeen = day, day
	l.load.Sanctions = append(l.load.Sanctions, s)
}

func rawSanctionsPath(day time.Time, name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", domain.SourceSanctions, day.Format("2006/01/02"), name)
}
