package usecase

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	tablePlans       = "plano_acao_especial"
	tableExecutors   = "executor_especial"
	tableCommitments = "empenho_especial"
	tableDocuments   = "documento_habil_especial"
	tableOrders      = "ordem_pagamento_ordem_bancaria_especial"
	tableWorkPlans   = "plano_trabalho_especial"
	tableReports     = "relatorio_gestao_novo_especial"
	filterPlan       = "id_plano_acao"
)

type LoadSpecialTransfers struct {
	src  ports.SpecialTransferSource
	repo ports.SpecialTransferRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadSpecialTransfers(src ports.SpecialTransferSource, repo ports.SpecialTransferRepository, runs ports.FetchRunRepository,
	raw ports.ObjectWriter, now func() time.Time) *LoadSpecialTransfers {
	return &LoadSpecialTransfers{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadSpecialTransfers) Execute(ctx context.Context) (domain.FetchRun, error) {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceSpecialTransfers, StartedAt: started, RequestedFrom: today, RequestedTo: today}
	rows, err := uc.collect(ctx)
	var transfers []domain.SpecialTransfer
	if err == nil {
		transfers = domain.BuildSpecialTransfers(rows)
		err = uc.repo.ReplaceSpecialTransfers(ctx, transfers)
	}
	if err == nil {
		run.Found, run.Stored = len(transfers), len(transfers)
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

func (uc *LoadSpecialTransfers) collect(ctx context.Context) (domain.SpecialTransferRows, error) {
	var rows domain.SpecialTransferRows
	if err := uc.repo.Ready(ctx); err != nil {
		return rows, fmt.Errorf("tabelas das transferências especiais: %w", err)
	}
	var err error
	if rows.Plans, err = fetchSpecial(ctx, uc, tablePlans, "cnpj_beneficiario_plano_acao", "eq."+domain.MunicipalityCNPJ, domain.ParseSpecialPlans); err != nil {
		return rows, err
	}
	if len(rows.Plans) == 0 {
		return rows, nil
	}
	plans := domain.SpecialFilterIn(domain.PlanIDs(rows.Plans))
	if rows.Executors, err = fetchSpecial(ctx, uc, tableExecutors, filterPlan, plans, domain.ParseSpecialExecutors); err != nil {
		return rows, err
	}
	if rows.WorkPlans, err = fetchSpecial(ctx, uc, tableWorkPlans, filterPlan, plans, domain.ParseSpecialWorkPlans); err != nil {
		return rows, err
	}
	if rows.Reports, err = fetchSpecial(ctx, uc, tableReports, filterPlan, plans, domain.ParseSpecialReports); err != nil {
		return rows, err
	}
	if rows.Commitments, err = fetchSpecial(ctx, uc, tableCommitments, filterPlan, plans, domain.ParseSpecialCommitments); err != nil || len(rows.Commitments) == 0 {
		return rows, err
	}
	commitments := domain.SpecialFilterIn(domain.CommitmentIDs(rows.Commitments))
	if rows.Documents, err = fetchSpecial(ctx, uc, tableDocuments, "id_empenho", commitments, domain.ParseSpecialDocuments); err != nil || len(rows.Documents) == 0 {
		return rows, err
	}
	rows.Orders, err = fetchSpecial(ctx, uc, tableOrders, "id_dh", domain.SpecialFilterIn(domain.DocumentIDs(rows.Documents)), domain.ParseSpecialOrders)
	return rows, err
}

func fetchSpecial[T any](ctx context.Context, uc *LoadSpecialTransfers, table, field, filter string, parse func([]byte) ([]T, error)) ([]T, error) {
	body, err := uc.src.Rows(ctx, table, url.Values{field: {filter}})
	if err != nil {
		return nil, err
	}
	rows, err := parse(body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	path := fmt.Sprintf("raw/%s/%s/%s", domain.SourceSpecialTransfers, uc.now().Format("2006/01/02"), table)
	return rows, archiveJSON(ctx, uc.raw, path, body, len(rows))
}
