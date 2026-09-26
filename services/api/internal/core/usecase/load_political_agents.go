package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const defaultAgentMonths = 3

type LoadPoliticalAgents struct {
	src  ports.PoliticalAgentSource
	repo ports.PoliticalAgentRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadPoliticalAgents(src ports.PoliticalAgentSource, repo ports.PoliticalAgentRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadPoliticalAgents {
	return &LoadPoliticalAgents{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

type agentArchive struct {
	Sources map[string]string `json:"source_sha256"`
	Rows    int               `json:"rows"`
}

func (uc *LoadPoliticalAgents) Execute(ctx context.Context, from, to time.Time) (domain.FetchRun, error) {
	started := uc.now()
	from, to, err := uc.period(from, to, started)
	if err != nil {
		return domain.FetchRun{}, err
	}
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourcePoliticalAgents, StartedAt: started, RequestedFrom: from, RequestedTo: to.AddDate(0, 1, -1)}
	err = uc.load(ctx, from, to, &run)
	run.FinishedAt = uc.now()
	if err != nil {
		run.Error = err.Error()
	}
	if saveErr := uc.runs.Save(ctx, run); saveErr != nil && err == nil {
		err = saveErr
	}
	return run, err
}

func (uc *LoadPoliticalAgents) period(from, to, now time.Time) (time.Time, time.Time, error) {
	current := monthOf(now)
	if to.IsZero() {
		to = current
	}
	if from.IsZero() {
		from = to.AddDate(0, 1-defaultAgentMonths, 0)
	}
	from, to = monthOf(from), monthOf(to)
	if to.Before(from) || to.After(current) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: meses %s a %s", domain.ErrInvalidInput, from.Format("2006-01"), to.Format("2006-01"))
	}
	first, _ := time.Parse("2006-01", domain.FirstPrefeituraPayMonth)
	if from.Before(first) {
		from = first
	}
	return from, to, nil
}

func monthOf(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC) }

func (uc *LoadPoliticalAgents) load(ctx context.Context, from, to time.Time, run *domain.FetchRun) error {
	if err := uc.repo.Ready(ctx); err != nil {
		return fmt.Errorf("tabelas dos agentes políticos: %w", err)
	}
	var errs []error
	for _, fetch := range []func(context.Context, time.Time, time.Time) (bodyLoad, error){uc.prefeitura, uc.camara} {
		body, err := fetch(ctx, from, to)
		if err == nil {
			err = uc.save(ctx, body, run)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", body.load.Body, err))
		}
	}
	return errors.Join(errs...)
}

type bodyLoad struct {
	load       domain.PoliticalAgentLoad
	sources    map[string]string
	councilSrc map[string]string
}

func (uc *LoadPoliticalAgents) prefeitura(ctx context.Context, from, to time.Time) (bodyLoad, error) {
	out := bodyLoad{load: domain.PoliticalAgentLoad{Body: domain.BodyPrefeitura, From: from, To: to}, sources: map[string]string{}}
	for m := from; !m.After(to); m = m.AddDate(0, 1, 0) {
		body, err := uc.src.PrefeituraPay(ctx, m.Year(), int(m.Month()))
		if err != nil {
			return out, err
		}
		rows, err := domain.ParsePrefeituraPay(body, m.Year(), int(m.Month()))
		if err != nil {
			return out, err
		}
		out.sources[m.Format("2006-01")] = sha256Hex(body)
		out.load.Pay = append(out.load.Pay, rows...)
	}
	return out, nil
}

func (uc *LoadPoliticalAgents) camara(ctx context.Context, from, to time.Time) (bodyLoad, error) {
	out := bodyLoad{load: domain.PoliticalAgentLoad{Body: domain.BodyCamara, From: from, To: to}, sources: map[string]string{}, councilSrc: map[string]string{}}
	for year := max(from.Year(), domain.FirstCamaraPayYear); year <= to.Year(); year++ {
		body, err := uc.src.CamaraPay(ctx, year)
		if err != nil {
			return out, err
		}
		rows, err := domain.ParseCamaraPay(body)
		if err != nil {
			return out, err
		}
		out.sources[fmt.Sprint(year)] = sha256Hex(body)
		for _, r := range rows {
			if !r.Month.Before(from) && !r.Month.After(to) {
				out.load.Pay = append(out.load.Pay, r)
			}
		}
	}
	councillors, err := uc.councillors(ctx, from, to, out.councilSrc)
	out.load.Councillors = councillors
	return out, err
}

func (uc *LoadPoliticalAgents) save(ctx context.Context, body bodyLoad, run *domain.FetchRun) error {
	body.load.Pay = domain.MergeAgentPay(body.load.Pay)
	run.Found += len(body.load.Pay)
	if len(body.sources) > 0 {
		if err := uc.put(ctx, body.load.Body, body.load.Pay, agentArchive{Sources: body.sources, Rows: len(body.load.Pay)}); err != nil {
			return err
		}
	}
	if len(body.councilSrc) > 0 {
		if err := uc.put(ctx, "vereadores", body.load.Councillors, agentArchive{Sources: body.councilSrc, Rows: len(body.load.Councillors)}); err != nil {
			return err
		}
	}
	if err := uc.repo.SavePoliticalAgents(ctx, body.load); err != nil {
		return fmt.Errorf("gravar agentes políticos: %w", err)
	}
	run.Stored += len(body.load.Pay)
	return nil
}

func (uc *LoadPoliticalAgents) councillors(ctx context.Context, from, to time.Time, sums map[string]string) ([]domain.Councillor, error) {
	seen := map[int]bool{}
	var out []domain.Councillor
	for year := max(from.Year(), domain.FirstCamaraPayYear); year <= to.Year(); year++ {
		legislature := domain.LegislatureOf(year)
		if seen[legislature] {
			continue
		}
		seen[legislature] = true
		body, err := uc.src.Councillors(ctx, year)
		if err != nil {
			return nil, err
		}
		rows, err := domain.ParseCouncillors(body)
		if err != nil {
			return nil, err
		}
		sums[fmt.Sprint(year)] = sha256Hex(body)
		out = append(out, rows...)
	}
	return out, nil
}

func (uc *LoadPoliticalAgents) put(ctx context.Context, name string, content any, manifest agentArchive) error {
	body, err := json.Marshal(content)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := uc.raw.Put(ctx, uc.rawPath(name+".json.gz"), "application/gzip", &buf); err != nil {
		return err
	}
	m, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return uc.raw.Put(ctx, uc.rawPath(name+".manifest.json"), "application/json", bytes.NewReader(m))
}

func (uc *LoadPoliticalAgents) rawPath(name string) string {
	return fmt.Sprintf("raw/%s/%s/%s", domain.SourcePoliticalAgents, uc.now().Format("2006/01/02"), name)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
