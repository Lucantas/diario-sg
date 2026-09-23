# Reindexação em produção e filtro por órgão — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ter um Cloud Run Job de reindexação executável por workflow manual e expor o órgão de cada ato (sigla, nome por extenso e filtro) na API e no front.

**Architecture:** O catálogo sigla → nome vira `core/domain/organ.go` e substitui a allowlist do parser. `ActFilter` ganha `Organ`, validado contra o catálogo; `Search`/`CountByMonth` filtram por `a.organ`; `GET /v1/organs` junta contagem do banco com o nome do catálogo. A infra ganha `google_cloud_run_v2_job.reindex` com a imagem da API e `.github/workflows/reindex.yml`.

**Tech Stack:** Go 1.23 (`net/http`, `lib/pq`), PostgreSQL 16, React 18 + Vite, Terraform (google ~> 6), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-22-reindex-producao-e-orgao-design.md`

## Global Constraints

- Zero comentários novos no código (AGENTS.md); o porquê vai no commit ou em `docs/`.
- Migrations aplicadas não são editadas; este plano não cria migration.
- Conventional commits em português, sem link de sessão de agente.
- Mensagens de erro e log em português.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck`, `terraform fmt -check -recursive infra` e `terraform validate` em `infra/envs/dev` e `infra/envs/prod`.
- A segmentação e o órgão atribuído pelo parser não mudam (mesmas 125 siglas).

---

### Task 1: Catálogo de órgãos no domínio

**Files:**
- Create: `services/api/internal/core/domain/organ.go`
- Create: `services/api/internal/core/domain/organ_test.go`
- Create: `docs/orgaos.md` (levantamento: sigla, nome, evidência)
- Modify: `services/api/internal/adapters/parser/regex.go` (`sectionOrgan`)
- Delete: `services/api/internal/adapters/parser/organs.go`

**Interfaces:**
- Produces: `domain.Organ{Acronym, Name string}`, `domain.IsKnownOrgan(string) bool`, `domain.OrganName(string) string`, `domain.NormalizeOrgan(string) (string, bool)`, `domain.OrganCount{Organ; Acts int}`.

- [ ] **Step 1: Teste que falha** (`organ_test.go`)

```go
func TestNormalizeOrgan(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"SEMED": {"SEMED", true}, " semed ": {"SEMED", true}, "": {"", true},
		"TOTAL": {"", false}, "SEM ED": {"", false},
	}
	for in, c := range cases {
		got, ok := NormalizeOrgan(in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeOrgan(%q) = %q, %v; esperava %q, %v", in, got, ok, c.want, c.ok)
		}
	}
}

func TestOrganCatalogIsWellFormed(t *testing.T) {
	if len(organNames) != 125 {
		t.Fatalf("esperava as 125 siglas da allowlist, veio %d", len(organNames))
	}
	for acronym, name := range organNames {
		if acronym != strings.ToUpper(acronym) || strings.ContainsAny(acronym, " \t") {
			t.Errorf("sigla fora do formato: %q", acronym)
		}
		if name != strings.TrimSpace(name) {
			t.Errorf("nome de %s com espaço nas pontas: %q", acronym, name)
		}
	}
}

func TestFilterRejectsUnknownOrgan(t *testing.T) {
	f := ActFilter{Organ: "semed"}
	if err := f.Normalize(); err != nil || f.Organ != "SEMED" {
		t.Fatalf("esperava SEMED sem erro, veio %q %v", f.Organ, err)
	}
	f = ActFilter{Organ: "TOTAL"}
	if err := f.Normalize(); !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("esperava ErrInvalidFilter, veio %v", err)
	}
}
```

- [ ] **Step 2:** `cd services/api && go test ./internal/core/domain/` → falha de compilação (`NormalizeOrgan` indefinido).

- [ ] **Step 3: Implementação**

```go
package domain

import "strings"

type Organ struct {
	Acronym string
	Name    string
}

type OrganCount struct {
	Organ
	Acts int
}

func IsKnownOrgan(acronym string) bool {
	_, ok := organNames[acronym]
	return ok
}

func OrganName(acronym string) string { return organNames[acronym] }

func NormalizeOrgan(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return "", true
	}
	if !IsKnownOrgan(s) {
		return "", false
	}
	return s, true
}

var organNames = map[string]string{
	"SEMED": "Secretaria Municipal de Educação",
	// … as 125 siglas de parser/organs.go, nome vazio quando não confirmado,
	// nomes vindos do levantamento registrado em docs/orgaos.md
}
```

O mapa final não leva comentário; as 125 chaves são exatamente as de `parser/organs.go`.
`ActFilter` ganha `Organ string` e, em `Normalize`:

```go
organ, ok := NormalizeOrgan(f.Organ)
if !ok {
	return ErrInvalidFilter
}
f.Organ = organ
```

Parser: `sectionOrgan` troca `knownOrgans[acronym]` por `domain.IsKnownOrgan(acronym)`; apagar `organs.go` (`setOf` só é usado lá).

- [ ] **Step 4:** `go test ./internal/core/domain/ ./internal/adapters/parser/` → PASS (os testes de órgão do parser não mudam).
- [ ] **Step 5:** commit `feat(domain): catálogo de órgãos com nome por extenso`.

### Task 2: Órgão na busca e filtro

**Files:**
- Modify: `services/api/internal/adapters/postgres/acts.go` (`Search`), `entities.go` (`CountByMonth`)
- Modify: `services/api/internal/presentation/http/router.go` (`filterFromQuery`), `dto.go` (`actHitDTO`, `toHitDTO`)
- Create: `services/api/internal/integration/organ_test.go`

**Interfaces:**
- Consumes: `domain.ActFilter.Organ`, `domain.OrganName`.
- Produces: JSON `organ`, `organ_name` em cada item de `/v1/acts`; parâmetro `organ` em `/v1/acts` e `/v1/stats/acts`.

- [ ] **Step 1: Teste de integração que falha** (`organ_test.go`), com texto de edição que tem `SEMAD` e `FMS`:

```go
const organGazette = "ATOS DO PREFEITO\nDECRETO Nº 1/2026\nDispõe sobre o horário.\n" +
	"SEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função.\n" +
	"FMS\nEXTRATO DO CONTRATO Nº 3/2026\nObjeto: medicamentos."
```

Indexa com um `FileStorage` que devolve esse texto; confere:
`/v1/acts?organ=semad` → 1 item com `organ=SEMAD` e `organ_name` igual a `domain.OrganName("SEMAD")`;
`/v1/acts?organ=TOTAL` → 400; `/v1/stats/acts?organ=FMS` → 1 mês com 1 ato;
`/v1/acts` sem filtro → 3 itens, o decreto com `organ` vazio.

- [ ] **Step 2:** `make test-integration` → FAIL (`organ` ignorado).
- [ ] **Step 3: Implementação**
  - `Search`: selecionar `a.organ` e acrescentar `AND ($8 = '' OR a.organ = $8)` (parâmetro `f.Organ`); `Scan` em `&h.Organ`.
  - `CountByMonth`: `AND ($6 = '' OR a.organ = $6)`.
  - `filterFromQuery`: `Organ: q.Get("organ")`.
  - `actHitDTO`: `Organ string json:"organ"`, `OrganName string json:"organ_name"`; `toHitDTO` preenche com `h.Organ` e `domain.OrganName(h.Organ)`.
- [ ] **Step 4:** `make test-integration` → PASS.
- [ ] **Step 5:** commit `feat(api): órgão e filtro por órgão na busca`.

### Task 3: `GET /v1/organs`

**Files:**
- Modify: `services/api/internal/core/ports/ports.go` (`ActRepository.CountByOrgan`)
- Modify: `services/api/internal/core/usecase/queries.go` (`ListOrgans`)
- Create: `services/api/internal/core/usecase/queries_test.go`
- Modify: `services/api/internal/adapters/postgres/entities.go` (`CountByOrgan`)
- Modify: `services/api/internal/presentation/http/router.go`, `dto.go`; `cmd/api/main.go`
- Modify: `services/api/internal/integration/organ_test.go`

**Interfaces:**
- Produces: `ports.ActRepository.CountByOrgan(ctx) (map[string]int, error)`; `usecase.NewListOrgans(ports.ActRepository) *ListOrgans`, `Execute(ctx) ([]domain.OrganCount, error)` ordenado por `Acts` desc e sigla asc; `API.Organs *usecase.ListOrgans`.

- [ ] **Step 1: Teste unitário que falha** (`queries_test.go`, com um fake de `ActRepository` que devolve `{"SEMED": 3, "FMS": 5, "SEMAD": 3}`): espera `FMS(5), SEMAD(3), SEMED(3)` com `Name` do catálogo.
- [ ] **Step 2:** `go test ./internal/core/usecase/` → FAIL.
- [ ] **Step 3: Implementação**

```go
type ListOrgans struct{ acts ports.ActRepository }

func NewListOrgans(a ports.ActRepository) *ListOrgans { return &ListOrgans{acts: a} }

func (uc *ListOrgans) Execute(ctx context.Context) ([]domain.OrganCount, error) {
	counts, err := uc.acts.CountByOrgan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.OrganCount, 0, len(counts))
	for acronym, n := range counts {
		out = append(out, domain.OrganCount{Organ: domain.Organ{Acronym: acronym, Name: domain.OrganName(acronym)}, Acts: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Acts != out[j].Acts {
			return out[i].Acts > out[j].Acts
		}
		return out[i].Acronym < out[j].Acronym
	})
	return out, nil
}
```

Repositório: `SELECT organ, count(*) FROM acts WHERE organ <> '' GROUP BY organ`.
Rota `GET /v1/organs` → `{"items":[{"acronym","name","acts"}]}` com `Cache-Control: public, max-age=3600`.
- [ ] **Step 4:** unitário + integração (`/v1/organs` → `SEMAD 1`, `FMS 1`) → PASS.
- [ ] **Step 5:** commit `feat(api): lista de órgãos com contagem de atos`.

### Task 4: Front — filtro e rótulo de órgão

**Files:**
- Modify: `apps/web/src/api.ts` (`ActHit.organ`, `organ_name`, `Organ`, `listOrgans`, `searchActs(q, type, organ, offset)`)
- Modify: `apps/web/src/App.tsx` (select de órgão)
- Modify: `apps/web/src/components.tsx` (órgão na meta)
- Modify: `apps/web/src/styles.css`

- [ ] **Step 1:** `listOrgans()` → `request<{items: Organ[]}>("/v1/organs")`; `searchActs` aceita `organ` e manda `organ` quando não vazio.
- [ ] **Step 2:** `SearchPage` carrega os órgãos no `useEffect` de montagem; `select` com rótulo visível "Órgão", opção "Todos os órgãos" e `SIGLA · Nome (n atos)`; mudar refaz a busca como `onType`.
- [ ] **Step 3:** `Result` mostra `<span className="organ" title={organ_name}>{organ}{organ_name && ` · ${organ_name}`}</span>` quando `organ` existe.
- [ ] **Step 4:** `npm run typecheck`; conferir na base local com `make run-api` + `npm run dev`.
- [ ] **Step 5:** commit `feat(web): filtro por órgão e órgão no resultado`.

### Task 5: Job de reindexação na nuvem

**Files:**
- Modify: `services/api/Dockerfile` (`for b in api worker migrate reindex`)
- Modify: `infra/stack/main.tf` (`google_cloud_run_v2_job.reindex`), `infra/stack/outputs.tf` (`reindex_job`)
- Modify: `.github/workflows/deploy.yml` (atualiza a imagem do job)
- Create: `.github/workflows/reindex.yml`

- [ ] **Step 1:** job Terraform:

```hcl
resource "google_cloud_run_v2_job" "reindex" {
  name                = "${local.p}-reindex"
  project             = var.project_id
  location            = var.region
  deletion_protection = false

  template {
    task_count = 1
    template {
      service_account = google_service_account.sa["worker"].email
      timeout         = "21600s"
      max_retries     = 0

      containers {
        image   = "us-docker.pkg.dev/cloudrun/container/job:latest"
        command = ["/app/reindex"]
        resources {
          limits = {
            cpu    = "1"
            memory = "1Gi"
          }
        }
        env {
          name  = "GAZETTE_BUCKET"
          value = google_storage_bucket.gazettes.name
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.database_url.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [google_project_service.apis, google_secret_manager_secret_iam_member.database_url]
}
```

- [ ] **Step 2:** `deploy.yml`, depois do scraper: `gcloud run jobs update $PREFIX-reindex --image $REGISTRY/api:$TAG --region $REGION --quiet`.
- [ ] **Step 3:** `reindex.yml`:

```yaml
name: Reindex

on:
  workflow_dispatch:
    inputs:
      environment:
        type: choice
        options: [dev, prod]
        default: dev
      from:
        description: Primeira data (AAAA-MM-DD)
        required: true
      to:
        description: Última data (AAAA-MM-DD)
        required: true

permissions:
  contents: read
  id-token: write

concurrency:
  group: reindex-${{ inputs.environment }}
  cancel-in-progress: false

jobs:
  reindex:
    runs-on: ubuntu-latest
    environment: ${{ inputs.environment }}
    timeout-minutes: 370
    env:
      FROM: ${{ inputs.from }}
      TO: ${{ inputs.to }}
      JOB: diario-${{ inputs.environment }}-reindex
      REGION: ${{ vars.GCP_REGION }}
    steps:
      - name: validar datas
        run: |
          for d in "$FROM" "$TO"; do
            [[ "$d" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { echo "data inválida: $d"; exit 1; }
          done
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ vars.WIF_PROVIDER }}
          service_account: ${{ vars.DEPLOYER_SA }}
      - uses: google-github-actions/setup-gcloud@v2
      - name: executar
        run: gcloud run jobs execute "$JOB" --region "$REGION" --args="-from,$FROM,-to,$TO" --wait
```

- [ ] **Step 4:** `terraform fmt -check -recursive infra`; `terraform -chdir=infra/envs/dev init -backend=false && terraform -chdir=infra/envs/dev validate` (idem prod); `docker build -f services/api/Dockerfile .` e conferir `/app/reindex` na imagem.
- [ ] **Step 5:** commit `feat(infra): job de reindexação e workflow manual`.

### Task 6: Documentação

**Files:**
- Modify: `README.md` (tabela da API: `organ` em `/v1/acts` e `/v1/stats/acts`, `/v1/organs`; como reindexar na nuvem)
- Modify: `docs/roadmap.md` (1c e job de reindexação feitos)
- Modify: `docs/decisoes-de-codigo.md` (catálogo no domínio, nome resolvido na apresentação, reindex manual e sem retry)

- [ ] **Step 1:** atualizar os três arquivos.
- [ ] **Step 2:** commit `docs: órgão na API e reindexação na nuvem`.
