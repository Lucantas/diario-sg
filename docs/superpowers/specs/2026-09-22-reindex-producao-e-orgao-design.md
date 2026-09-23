# Reindexação em produção e filtro por órgão (Entrega 1c)

Data: 2026-09-22. Parte da Entrega 1 de `docs/roadmap.md`. Junta duas peças
pequenas que dependem uma da outra: o filtro por órgão só fica certo em
produção depois que a base de lá for reindexada com o parser atual, e a
reindexação hoje só roda na base local.

## Objetivo

- Produção (e dev) passa a ter um **Cloud Run Job de reindexação**, com a
  mesma imagem da API, e um workflow manual do GitHub Actions que o executa
  para um período.
- Cada ato passa a expor o **órgão** (sigla e nome por extenso, quando
  conhecido) na busca, e a busca ganha **filtro por órgão**.
- `GET /v1/organs` lista os órgãos com a contagem de atos, para o front
  montar o filtro.

## Fora do escopo

- Página, hash do PDF e cópia arquivada (1b).
- Separar a sigla seguida do nome por extenso no parser (limitação de
  `docs/fase-1-relatorio.md` §2.3).
- Rodar a reindexação na nuvem neste PR: não existe ambiente publicado (os
  workflows Infra e Deploy falham na autenticação porque o Environment do
  GitHub não tem as Variables). O PR entrega a infra validada com
  `terraform validate` e o workflow pronto para quando houver ambiente.

## Fatos que orientam o desenho

- A allowlist de 125 siglas está em `parser/organs.go` e é usada só por
  `sectionOrgan`. A tabela de nomes precisa ter exatamente as mesmas
  chaves; duas listas separadas divergiriam.
- Na base local (22/09/2026), 80.832 atos têm órgão. 55 siglas têm menos de
  10 atos e várias só aparecem em 2010–2016.
- A mineração automática do nome no texto dá falsos positivos (`SEMFA`
  aparece junto de "SUBSECRETARIA DE TRIBUTOS", que é uma subsecretaria
  dela). O nome de cada sigla foi levantado a partir do próprio Diário, com
  a evidência registrada (edição e trecho). Sigla sem evidência fica sem
  nome e aparece só como sigla.
- O deployer do GitHub Actions tem `roles/run.developer`, que permite
  executar jobs com argumentos sobrescritos, e já age como a conta do
  worker.

## Decisões

1. **Catálogo de órgãos no domínio.** `domain/organ.go` guarda sigla →
   nome (nome vazio quando não confirmado). O parser passa a perguntar ao
   domínio se a sigla é conhecida; a allowlist deixa de existir no parser.
   O domínio é o lugar certo porque API (validação do filtro, nome por
   extenso) e parser usam a mesma lista.
2. **Filtro por sigla exata**, validado contra o catálogo (sigla
   desconhecida é `400`). Aceita minúsculas (normaliza para maiúsculas).
3. **Nome por extenso resolvido na apresentação**, não gravado no banco: a
   tabela muda sem reindexar.
4. **Job de reindexação reaproveita a conta do worker**, que já lê o bucket
   e o segredo do banco. Uma conta nova teria as mesmas permissões.
5. **Execução manual**, por `workflow_dispatch` com `environment`, `from` e
   `to`. Reindexar é decisão de quem mudou o parser, não passo automático
   do deploy: uma reindexação completa leva horas e o deploy não deve
   esperar por ela.
6. **Sem retry no job** (`max_retries = 0`): a reindexação já registra
   falha por edição e segue; repetir tudo por causa de uma edição com PDF
   ausente custaria horas. Quem executa lê o log e roda de novo o período.

## Desenho

### Domínio (`core/domain/organ.go`)

```go
type Organ struct {
	Acronym string
	Name    string
}

func IsKnownOrgan(acronym string) bool
func OrganName(acronym string) string
func NormalizeOrgan(s string) (string, bool)
```

`organNames` é um `map[string]string` com as 125 siglas. `ActFilter` ganha
`Organ string`; `Normalize` passa a sigla para maiúsculas e devolve
`ErrInvalidFilter` quando ela não é conhecida. `ActHit.Act.Organ` já
existe; a busca passa a preenchê-lo.

### Parser

`sectionOrgan` usa `domain.IsKnownOrgan`; `organs.go` sai. Segmentação e
órgão atribuído não mudam (mesmas siglas).

### Dados

Sem migration: `acts.organ` e `acts_organ_idx` existem desde a 005.
`ActRepository` ganha `CountByOrgan(ctx) ([]domain.OrganCount, error)`
(`SELECT organ, count(*) FROM acts WHERE organ <> '' GROUP BY organ`).
`Search` e `CountByMonth` ganham `AND ($N = '' OR a.organ = $N)` e `Search`
passa a selecionar `a.organ`.

### API

- `GET /v1/acts?organ=SEMED` e `GET /v1/stats/acts?organ=SEMED`.
- Cada item da busca ganha `organ` e `organ_name` (vazios quando não há).
- `GET /v1/organs` → `{"items":[{"acronym":"SEMAD","name":"…","acts":24698}]}`,
  ordenado por contagem decrescente, só siglas com atos. `Cache-Control:
  public, max-age=3600`.
- Caso de uso `ListOrgans` junta a contagem do repositório com o nome do
  catálogo.

### Front

- `select` "Órgão" ao lado dos tipos, com as opções de `/v1/organs`
  (`SEMED · Secretaria Municipal de Educação`). Trocar o órgão refaz a
  busca, como os tipos.
- No resultado, a sigla aparece na linha de metadados, com o nome por
  extenso no `title` e no texto quando existir.

### Infra

- `services/api/Dockerfile` compila também `reindex`.
- `infra/stack`: `google_cloud_run_v2_job.reindex` (`${prefix}-reindex`),
  conta do worker, comando `/app/reindex`, `GAZETTE_BUCKET` e
  `DATABASE_URL` (segredo), 1 CPU, 1 GiB, `timeout = 21600s`,
  `max_retries = 0`, imagem ignorada pelo Terraform (quem troca é o
  deploy). Output `reindex_job`.
- `deploy.yml`: `gcloud run jobs update $PREFIX-reindex --image api:$TAG`.
- `.github/workflows/reindex.yml`: `workflow_dispatch` com `environment`
  (dev/prod), `from` e `to`; valida as datas com regex antes de usar;
  executa `gcloud run jobs execute $PREFIX-reindex --args=-from,…,-to,… --wait`.
  As entradas vão por `env`, nunca interpoladas direto no script.

## Testes

- Domínio: `NormalizeOrgan` (minúsculas, espaços, sigla desconhecida);
  `ActFilter.Normalize` com órgão; toda sigla do catálogo em maiúsculas e
  sem espaços; nome, quando existe, sem espaços nas pontas.
- Parser: os testes de órgão existentes passam sem mudança.
- Integração: busca com `organ` devolve só atos da sigla e traz `organ` e
  `organ_name`; `organ` desconhecido dá 400; `/v1/organs` conta certo;
  `stats` respeita o órgão.
- Infra: `terraform fmt -check` e `terraform validate` em dev e prod.
- Front: `npm run typecheck` e conferência visual na base local.

## Riscos

- **Sigla nova** que a prefeitura criar continua precisando entrar no
  catálogo (já era assim com a allowlist).
- **Nomes que mudaram com o tempo** (a mesma sigla com outro nome em outra
  gestão): o catálogo guarda o nome mais recente; a observação fica no
  levantamento, não no código.
- **Timeout de 6 h**: a reindexação local completa (4.301 edições) cabe
  folgado, mas em produção o `pdftotext` roda com 1 CPU; se estourar, a
  execução é retomável por período.
