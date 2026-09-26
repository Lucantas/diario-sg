package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const registryMonthLayout = "2006-01"

var registryFileKinds = []string{"Empresas", "Estabelecimentos", "Socios"}

type LoadRegistry struct {
	src  ports.RegistrySource
	repo ports.RegistryRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadRegistry(src ports.RegistrySource, repo ports.RegistryRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadRegistry {
	return &LoadRegistry{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

type archivedFile struct {
	SourceSHA256 string `json:"source_sha256"`
	Rows         int    `json:"rows"`
}

type registryLoading struct {
	codes    domain.RegistryCodes
	exact    map[string]bool
	bases    map[string]bool
	load     domain.RegistryLoad
	failed   int
	manifest map[string]archivedFile
}

func (uc *LoadRegistry) Execute(ctx context.Context, month string) (domain.FetchRun, error) {
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceReceita, StartedAt: uc.now()}
	loading, monthStart, err := uc.load(ctx, month, &run)
	if err == nil {
		err = uc.repo.Replace(ctx, monthStart, loading.load)
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

func (uc *LoadRegistry) load(ctx context.Context, month string, run *domain.FetchRun) (*registryLoading, time.Time, error) {
	monthStart, month, err := uc.month(ctx, month)
	if err != nil {
		return nil, monthStart, err
	}
	run.RequestedFrom, run.RequestedTo = monthStart, monthStart
	loading, err := uc.prepare(ctx, month)
	if err != nil {
		return nil, monthStart, err
	}
	run.Found = len(loading.exact)
	files, err := uc.src.Files(ctx, month)
	if err != nil {
		return nil, monthStart, err
	}
	for _, file := range files {
		if kind := registryFileKind(file); kind != "" {
			if err := uc.loadFile(ctx, month, monthStart, file, kind, loading); err != nil {
				return nil, monthStart, err
			}
		}
	}
	if err := uc.putManifest(ctx, monthStart, loading.manifest); err != nil {
		return nil, monthStart, err
	}
	run.Stored, run.Failed = len(loading.load.Establishments), loading.failed
	run.Skipped = run.Found - run.Stored
	return loading, monthStart, nil
}

func (uc *LoadRegistry) month(ctx context.Context, month string) (time.Time, string, error) {
	if month == "" {
		latest, err := uc.src.LatestMonth(ctx)
		if err != nil {
			return time.Time{}, "", err
		}
		month = latest
	}
	start, err := time.Parse(registryMonthLayout, month)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("%w: mês %q (use AAAA-MM)", domain.ErrInvalidInput, month)
	}
	return start, month, nil
}

func (uc *LoadRegistry) prepare(ctx context.Context, month string) (*registryLoading, error) {
	cited, err := uc.repo.CitedCNPJs(ctx)
	if err != nil {
		return nil, err
	}
	codes, err := uc.src.Codes(ctx, month)
	if err != nil {
		return nil, err
	}
	l := &registryLoading{codes: codes, exact: map[string]bool{}, bases: map[string]bool{}, manifest: map[string]archivedFile{}}
	for _, cnpj := range cited {
		l.exact[cnpj], l.bases[domain.CNPJBase(cnpj)] = true, true
	}
	return l, nil
}

func registryFileKind(file string) string {
	for _, kind := range registryFileKinds {
		if strings.HasPrefix(file, kind) {
			return kind
		}
	}
	return ""
}

func (uc *LoadRegistry) loadFile(ctx context.Context, month string, monthStart time.Time, file, kind string, l *registryLoading) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	w := csv.NewWriter(gz)
	w.Comma = ';'
	rows := 0
	sum, err := uc.src.Rows(ctx, month, file, func(f []string) error {
		if !l.wanted(kind, f) {
			return nil
		}
		rows++
		l.add(kind, f)
		return w.Write(f)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	l.manifest[file] = archivedFile{SourceSHA256: sum, Rows: rows}
	return uc.raw.Put(ctx, rawRegistryPath(monthStart, strings.TrimSuffix(file, ".zip")+".csv.gz"), "application/gzip", &buf)
}

func (l *registryLoading) wanted(kind string, f []string) bool {
	if len(f) == 0 || !l.bases[f[0]] {
		return false
	}
	if kind == "Estabelecimentos" {
		return len(f) > 2 && l.exact[f[0]+f[1]+f[2]]
	}
	return true
}

func (l *registryLoading) add(kind string, f []string) {
	var err error
	switch kind {
	case "Empresas":
		var c domain.RegistryCompany
		if c, err = domain.ParseCompanyRow(f, l.codes); err == nil {
			l.load.Companies = append(l.load.Companies, c)
		}
	case "Estabelecimentos":
		var e domain.RegistryEstablishment
		if e, err = domain.ParseEstablishmentRow(f, l.codes); err == nil {
			l.load.Establishments = append(l.load.Establishments, e)
		}
	case "Socios":
		var p domain.RegistryPartner
		if p, err = domain.ParsePartnerRow(f, l.codes); err == nil {
			l.load.Partners = append(l.load.Partners, p)
		}
	}
	if err != nil {
		l.failed++
	}
}

func (uc *LoadRegistry) putManifest(ctx context.Context, monthStart time.Time, manifest map[string]archivedFile) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return uc.raw.Put(ctx, rawRegistryPath(monthStart, "manifest.json"), "application/json", bytes.NewReader(body))
}

func rawRegistryPath(monthStart time.Time, name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", domain.SourceReceita, monthStart.Format("2006/01/02"), name)
}
