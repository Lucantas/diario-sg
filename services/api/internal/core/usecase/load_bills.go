package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	billBatchSize              = 500
	maxConsecutiveBillFailures = 10
)

type LoadBills struct {
	src   ports.BillSource
	repo  ports.BillRepository
	runs  ports.FetchRunRepository
	raw   ports.ObjectWriter
	now   func() time.Time
	batch int
}

func NewLoadBills(src ports.BillSource, repo ports.BillRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadBills {
	return &LoadBills{src: src, repo: repo, runs: runs, raw: raw, now: now, batch: billBatchSize}
}

type rawBillLine struct {
	Processo string `json:"processo"`
	HTML     string `json:"html"`
}

type billBatch struct {
	bills []domain.Bill
	raw   bytes.Buffer
	count int
}

func (uc *LoadBills) Execute(ctx context.Context, full bool, max int) (domain.FetchRun, error) {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceBills, StartedAt: started, RequestedFrom: today, RequestedTo: today}
	err := uc.collect(ctx, &run, full, max)
	run.FinishedAt = uc.now()
	if err != nil {
		run.Error = err.Error()
	}
	if saveErr := uc.runs.Save(ctx, run); saveErr != nil && err == nil {
		err = saveErr
	}
	return run, err
}

func (uc *LoadBills) collect(ctx context.Context, run *domain.FetchRun, full bool, max int) error {
	if err := uc.repo.Ready(ctx); err != nil {
		return fmt.Errorf("tabelas dos processos: %w", err)
	}
	queue, err := uc.queue(ctx, full, max)
	if err != nil {
		return err
	}
	run.Found = len(queue)
	batch := &billBatch{}
	consecutive := 0
	for _, key := range queue {
		page, err := uc.src.ProcessPage(ctx, key)
		if err != nil {
			if ctx.Err() != nil {
				return errors.Join(ctx.Err(), uc.flush(context.WithoutCancel(ctx), run, batch))
			}
			run.Failed++
			consecutive++
			if consecutive >= maxConsecutiveBillFailures {
				return errors.Join(fmt.Errorf("%d páginas seguidas falharam, a última: %w", consecutive, err), uc.flush(ctx, run, batch))
			}
			continue
		}
		consecutive = 0
		uc.add(run, batch, key, page)
		if len(batch.bills) >= uc.batch {
			if err := uc.flush(ctx, run, batch); err != nil {
				return err
			}
		}
	}
	return uc.flush(ctx, run, batch)
}

func (uc *LoadBills) queue(ctx context.Context, full bool, max int) ([]domain.BillKey, error) {
	keys, err := uc.src.ProcessKeys(ctx)
	if err != nil {
		return nil, err
	}
	keys = uniqueKeys(keys)
	var queue []domain.BillKey
	if full {
		queue = keys
	} else {
		known, err := uc.repo.KnownBills(ctx)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			if !known[k] {
				queue = append(queue, k)
			}
		}
		if max <= 0 || len(queue) < max {
			stale, err := uc.repo.StaleOpenBills(ctx, domain.NormativeBillKinds, remaining(max, len(queue)))
			if err != nil {
				return nil, err
			}
			queue = append(queue, stale...)
		}
	}
	if max > 0 && len(queue) > max {
		queue = queue[:max]
	}
	return queue, nil
}

func uniqueKeys(keys []domain.BillKey) []domain.BillKey {
	seen := make(map[domain.BillKey]bool, len(keys))
	out := make([]domain.BillKey, 0, len(keys))
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func remaining(max, used int) int {
	if max <= 0 {
		return 1 << 30
	}
	return max - used
}

func (uc *LoadBills) add(run *domain.FetchRun, batch *billBatch, key domain.BillKey, page []byte) {
	bill, err := domain.ParseBillPage(page, uc.src.BaseURL())
	switch {
	case errors.Is(err, domain.ErrBillNotFound):
		run.Skipped++
		return
	case err != nil || bill.Key != key:
		run.Failed++
		return
	}
	bill.FetchedAt = uc.now()
	batch.bills = append(batch.bills, bill)
	enc := json.NewEncoder(&batch.raw)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(rawBillLine{Processo: key.Slug(), HTML: string(domain.BillPageMain(page))})
}

func (uc *LoadBills) flush(ctx context.Context, run *domain.FetchRun, batch *billBatch) error {
	if len(batch.bills) == 0 {
		return nil
	}
	if err := uc.repo.SaveBills(ctx, batch.bills); err != nil {
		return err
	}
	batch.count++
	path := fmt.Sprintf("raw/%s/%s/%s-%d", domain.SourceBills, run.StartedAt.Format("2006/01/02"), run.ID, batch.count)
	if err := archiveRaw(ctx, uc.raw, path, "jsonl", batch.raw.Bytes(), len(batch.bills)); err != nil {
		return err
	}
	run.Stored += len(batch.bills)
	batch.bills = nil
	batch.raw.Reset()
	return nil
}
