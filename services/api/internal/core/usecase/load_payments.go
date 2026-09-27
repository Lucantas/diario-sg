package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const firstTCEYear = 2020

type LoadPayments struct {
	src  ports.PaymentSource
	repo ports.PaymentRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadPayments(src ports.PaymentSource, repo ports.PaymentRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadPayments {
	return &LoadPayments{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadPayments) Execute(ctx context.Context, from, to int) (domain.FetchRun, error) {
	started := uc.now()
	if from == 0 && to == 0 {
		from, to = started.Year()-1, started.Year()
	}
	if to < from || to > started.Year() {
		return domain.FetchRun{}, fmt.Errorf("%w: anos %d a %d (até o ano corrente)", domain.ErrInvalidInput, from, to)
	}
	from = max(from, firstTCEYear)
	if from > to {
		return domain.FetchRun{}, nil
	}
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceTCE, StartedAt: started,
		RequestedFrom: time.Date(from, 1, 1, 0, 0, 0, 0, time.UTC), RequestedTo: time.Date(to, 12, 31, 0, 0, 0, 0, time.UTC)}
	err := uc.repo.Ready(ctx)
	if err != nil {
		err = fmt.Errorf("tabela dos pagamentos: %w", err)
	}
	for year := from; year <= to && err == nil; year++ {
		err = uc.loadYear(ctx, year, &run)
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

func (uc *LoadPayments) loadYear(ctx context.Context, year int, run *domain.FetchRun) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	w := csv.NewWriter(gz)
	w.Comma = ';'
	var cols domain.TCEColumns
	var payments []domain.Payment
	sum, err := uc.src.Commitments(ctx, year, func(header, row []string) error {
		if cols == nil {
			parsed, err := domain.NewTCEColumns(header)
			if err != nil {
				return err
			}
			cols = parsed
			if err := w.Write(header); err != nil {
				return err
			}
		}
		if !cols.IsCompany(row) {
			run.Skipped++
			return nil
		}
		run.Found++
		p, err := domain.ParseTCERow(cols, row)
		if err != nil {
			run.Failed++
			return nil
		}
		payments = append(payments, p)
		return w.Write(row)
	})
	if err != nil {
		return err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	name := strconv.Itoa(year)
	if err := uc.raw.Put(ctx, uc.rawPath(name+".csv.gz"), "application/gzip", &buf); err != nil {
		return err
	}
	manifest, err := json.MarshalIndent(archivedFile{SourceSHA256: sum, Rows: len(payments)}, "", "  ")
	if err != nil {
		return err
	}
	if err := uc.raw.Put(ctx, uc.rawPath(name+".manifest.json"), "application/json", bytes.NewReader(manifest)); err != nil {
		return err
	}
	if err := uc.repo.ReplaceYear(ctx, domain.SourceTCE, year, payments); err != nil {
		return fmt.Errorf("gravar %d: %w", year, err)
	}
	run.Stored += len(payments)
	return nil
}

func (uc *LoadPayments) rawPath(name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", domain.SourceTCE, uc.now().Format("2006/01/02"), name)
}
