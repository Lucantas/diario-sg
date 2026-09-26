package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	amendmentsFile        = "emendas-parlamentares/UNICO"
	amendmentsCSV         = "EmendasParlamentares.csv"
	amendmentPaymentsCSV  = "EmendasParlamentares_PorFavorecido.csv"
	transfersDataset      = "transferencias"
	defaultTransferMonths = 3
)

type LoadFederal struct {
	src  ports.FederalSource
	repo ports.FederalRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadFederal(src ports.FederalSource, repo ports.FederalRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadFederal {
	return &LoadFederal{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadFederal) Amendments(ctx context.Context) (domain.FetchRun, error) {
	run := uc.newRun(domain.SourceAmendments)
	err := uc.loadAmendments(ctx, &run)
	return run, uc.finish(ctx, &run, err)
}

func (uc *LoadFederal) loadAmendments(ctx context.Context, run *domain.FetchRun) error {
	if err := uc.repo.Ready(ctx); err != nil {
		return fmt.Errorf("tabelas do dinheiro federal: %w", err)
	}
	var amendments []domain.Amendment
	var payments []domain.AmendmentPayment
	archives := map[string]*csvArchive{amendmentsCSV: newCSVArchive(), amendmentPaymentsCSV: newCSVArchive()}
	cols := map[string]domain.CSVColumns{}
	sha, err := uc.src.ZipCSVs(ctx, amendmentsFile, func(name string, header, row []string) error {
		required, ok := map[string][]string{amendmentsCSV: domain.AmendmentColumns, amendmentPaymentsCSV: domain.AmendmentPaymentColumns}[name]
		if !ok {
			return nil
		}
		c, err := columnsFor(cols, name, header, required)
		if err != nil {
			return err
		}
		switch name {
		case amendmentsCSV:
			if !domain.IsSaoGoncaloAmendment(c, row) {
				return nil
			}
			a, err := domain.ParseAmendment(c, row)
			if err != nil {
				run.Failed++
				return nil
			}
			amendments = append(amendments, a)
		case amendmentPaymentsCSV:
			if !domain.IsSaoGoncaloCompanyPayment(c, row) {
				return nil
			}
			p, err := domain.ParseAmendmentPayment(c, row)
			if err != nil {
				run.Failed++
				return nil
			}
			payments = append(payments, p)
		}
		run.Found++
		return archives[name].write(header, row)
	})
	if err != nil {
		return err
	}
	for name, a := range archives {
		if err := a.put(ctx, uc.raw, uc.rawPrefix(domain.SourceAmendments, strings.TrimSuffix(name, ".csv")), sha); err != nil {
			return err
		}
	}
	if err := uc.repo.ReplaceAmendments(ctx, amendments, payments); err != nil {
		return err
	}
	run.Stored = len(amendments) + len(payments)
	return nil
}

func (uc *LoadFederal) Transfers(ctx context.Context, from, to time.Time) (domain.FetchRun, error) {
	run := uc.newRun(domain.SourceTransfers)
	err := uc.loadTransfers(ctx, from, to, &run)
	return run, uc.finish(ctx, &run, err)
}

func (uc *LoadFederal) loadTransfers(ctx context.Context, from, to time.Time, run *domain.FetchRun) error {
	if err := uc.repo.Ready(ctx); err != nil {
		return fmt.Errorf("tabelas do dinheiro federal: %w", err)
	}
	if to.IsZero() {
		latest, err := uc.src.LatestMonth(ctx, transfersDataset)
		if err != nil {
			return err
		}
		to = latest
	}
	if from.IsZero() {
		from = to.AddDate(0, 1-defaultTransferMonths, 0)
	}
	if to.Before(from) {
		return fmt.Errorf("%w: meses %s a %s", domain.ErrInvalidInput, from.Format("01/2006"), to.Format("01/2006"))
	}
	run.RequestedFrom, run.RequestedTo = from, to.AddDate(0, 1, -1)
	for month := from; !month.After(to); month = month.AddDate(0, 1, 0) {
		if err := uc.loadTransferMonth(ctx, month, run); err != nil {
			return err
		}
	}
	return nil
}

func (uc *LoadFederal) loadTransferMonth(ctx context.Context, month time.Time, run *domain.FetchRun) error {
	var transfers []domain.FederalTransfer
	archive := newCSVArchive()
	var cols domain.CSVColumns
	sha, err := uc.src.ZipCSVs(ctx, transfersDataset+"/"+month.Format("200601"), func(_ string, header, row []string) error {
		if cols == nil {
			c, err := domain.NewCSVColumns(header, domain.TransferColumns...)
			if err != nil {
				return err
			}
			cols = c
		}
		if !domain.IsSaoGoncaloTransfer(cols, row) {
			return nil
		}
		run.Found++
		t, company, err := domain.ParseTransfer(cols, row)
		if !company {
			run.Skipped++
			return nil
		}
		if err != nil {
			run.Failed++
			return nil
		}
		transfers = append(transfers, t)
		return archive.write(header, row)
	})
	if err != nil {
		return err
	}
	if err := archive.put(ctx, uc.raw, uc.rawPrefix(domain.SourceTransfers, month.Format("200601")), sha); err != nil {
		return err
	}
	if err := uc.repo.ReplaceTransferMonth(ctx, month, transfers); err != nil {
		return fmt.Errorf("gravar %s: %w", month.Format("01/2006"), err)
	}
	run.Stored += len(transfers)
	return nil
}

func columnsFor(cache map[string]domain.CSVColumns, name string, header, required []string) (domain.CSVColumns, error) {
	if c, ok := cache[name]; ok {
		return c, nil
	}
	c, err := domain.NewCSVColumns(header, required...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	cache[name] = c
	return c, nil
}

func (uc *LoadFederal) newRun(source string) domain.FetchRun {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	return domain.FetchRun{ID: domain.NewRunID(), Source: source, StartedAt: started, RequestedFrom: today, RequestedTo: today}
}

func (uc *LoadFederal) finish(ctx context.Context, run *domain.FetchRun, err error) error {
	run.FinishedAt = uc.now()
	if err != nil {
		run.Error = err.Error()
	}
	if saveErr := uc.runs.Save(ctx, *run); saveErr != nil && err == nil {
		err = saveErr
	}
	return err
}

func (uc *LoadFederal) rawPrefix(source, name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", source, uc.now().Format("2006/01/02"), name)
}
