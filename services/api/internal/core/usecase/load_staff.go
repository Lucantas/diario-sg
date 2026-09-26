package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadStaff struct {
	src  ports.StaffSource
	repo ports.StaffRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadStaff(src ports.StaffSource, repo ports.StaffRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadStaff {
	return &LoadStaff{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadStaff) Execute(ctx context.Context, from, to int) (domain.FetchRun, error) {
	started := uc.now()
	if from == 0 && to == 0 {
		from, to = started.Year()-1, started.Year()
	}
	if to < from || to > started.Year() {
		return domain.FetchRun{}, fmt.Errorf("%w: anos %d a %d", domain.ErrInvalidInput, from, to)
	}
	from = max(from, domain.FirstStaffYear)
	if from > to {
		return domain.FetchRun{}, nil
	}
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceStaff, StartedAt: started,
		RequestedFrom: time.Date(from, 1, 1, 0, 0, 0, 0, time.UTC), RequestedTo: time.Date(to, 12, 31, 0, 0, 0, 0, time.UTC)}
	err := uc.repo.Ready(ctx)
	if err != nil {
		err = fmt.Errorf("tabela do pessoal: %w", err)
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

func (uc *LoadStaff) loadYear(ctx context.Context, year int, run *domain.FetchRun) error {
	body, err := uc.src.Staff(ctx, year)
	if err != nil {
		return err
	}
	rows, err := domain.ParseStaffJSON(body)
	if err != nil {
		return err
	}
	run.Found += len(rows)
	if len(rows) == 0 {
		return nil
	}
	if err := uc.archive(ctx, year, body, len(rows)); err != nil {
		return err
	}
	if err := uc.repo.ReplaceStaffYear(ctx, year, rows); err != nil {
		return fmt.Errorf("gravar %d: %w", year, err)
	}
	run.Stored += len(rows)
	return nil
}

func (uc *LoadStaff) archive(ctx context.Context, year int, body []byte, rows int) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	name := strconv.Itoa(year)
	if err := uc.raw.Put(ctx, uc.rawPath(name+".json.gz"), "application/gzip", &buf); err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	manifest, err := json.MarshalIndent(archivedFile{SourceSHA256: hex.EncodeToString(sum[:]), Rows: rows}, "", "  ")
	if err != nil {
		return err
	}
	return uc.raw.Put(ctx, uc.rawPath(name+".manifest.json"), "application/json", bytes.NewReader(manifest))
}

func (uc *LoadStaff) rawPath(name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", domain.SourceStaff, uc.now().Format("2006/01/02"), name)
}
