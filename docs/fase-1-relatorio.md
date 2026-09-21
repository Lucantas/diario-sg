# Fase 1 — relatório

Data: 2026-09-21. Branch `fase-1` (a `main` guarda o baseline recebido; a pasta
não era um repositório git, o `git init` foi feito nesta fase).

## 1. O que foi verificado e o que estava quebrado no início

Verificado rodando de fato (`make up setup migrate`, api e worker em background,
`make lint test test-integration`):

| Item | Resultado |
| --- | --- |
| `make up` (Postgres 16, emulador Pub/Sub, fake-gcs-server) | Subiu. O daemon do Docker estava parado na máquina; foi iniciado via `systemctl start docker` (polkit), sem mudar nada no repositório. |
| `make setup` | Criou bucket, 2 tópicos e 2 assinaturas push. O script silencia erros (`\|\| true`); confirmei os recursos consultando os emuladores. |
| `make migrate` | Aplicou `001_init.sql`. |
| `make lint`, `make test` | Passaram (vet, gofmt, 6 pacotes com testes). |
| `make test-integration` | Passou. Mas rodava no **mesmo banco** do ambiente local e deixava uma edição fictícia gravada; depois de ingerir edições reais o teste passou a falhar (busca por "medicamento" achava atos reais). Corrigido: roda em `diario_test`, criado pelo próprio teste, com `TRUNCATE` no início. |
| Worker recebendo push do emulador | Funcionou na primeira ingestão (`gazette.fetched` → indexação → `gazette.indexed` → casamento de alertas). |
| `services/scraper/.../pmsg/source.go` | O `TODO(ajustar)` estava certo: o parser genérico procurava `<a href="*.pdf">` na home, que não tem nenhum link para PDF. Contra o site real teria encontrado 0 edições. Reescrito na tarefa 7. |
| Parser original nas edições reais | Com `pdftotext -layout`, as duas colunas do Diário saem misturadas na mesma linha: a edição de 18/09/2026 virou **9 atos** (esperado ~50) com títulos como `DECRETO Nº 441/2026 3.3.90.31.00 127 1.501.0000.0000 0,00 3.000,00`. |
| Terraform | `validate` falhava em `envs/dev` e `envs/prod`: `local.resend_enabled` derivava de variável `sensitive` e era usado em `for_each`. `bootstrap` passava. Nenhum erro de schema do provider Google (6.50.0). |
| Front | `npm run typecheck && npm run build` passavam. |
| Ambiente | Todos os arquivos tinham mtime ~2h no futuro (make avisava "Clock skew"); corrigido com `touch`, sem efeito no git. |

Não verificado: nada foi executado na nuvem (sem credenciais GCP/Neon/Resend);
os workflows do GitHub Actions não rodaram.

## 2. Números do parser

Fonte: 9 edições reais (7 baixadas à mão + 2 coletadas pelo scraper), indexadas
pelo pipeline completo; contagens tiradas do banco após a indexação. As mesmas
contagens saem de `go test -run Measure -v ./internal/adapters/parser/` (que
lê `services/api/testdata/editions/*.txt`; `./scripts/fetch-editions.sh` baixa
os arquivos, que não são versionados por conterem nomes de pessoas).

| Edição | Páginas | Atos | `outro` | % outro | nomeação+exoneração | contratos/licitações |
| --- | --- | --- | --- | --- | --- | --- |
| 2024-03-15 (1062) | 12 | 38 | 1 | 3% | 18 | 4 |
| 2026-03-31 (1652) | 12 | 49 | 3 | 6% | 2 | 18 |
| 2026-06-30 (1714) | 8 | 20 | 5 | 25% | 1 | 0 |
| 2026-08-31 (1758) | 21 | 143 | 6 | 4% | 3 | 16 |
| 2026-09-14 (1767) | — | 19 | 2 | 11% | 1 | 2 |
| 2026-09-15 (1768) | — | 65 | 4 | 6% | 3 | 6 |
| 2026-09-16 (1769) | 23 | 58 | 7 | 12% | 4 | 14 |
| 2026-09-17 (1770) | 11 | 54 | 4 | 7% | 7 | 6 |
| 2026-09-18 (1771) | 11 | 51 | 10 | 20% | 4 | 8 |
| **Total** | | **497** | **42** | **8%** | 43 | 74 |

Distribuição por tipo (497 atos): despacho 178, portaria 97, outro 42,
licitação 41, nomeação 25, edital 24, decreto 21, exoneração 18, aditivo 15,
contrato 14, resolução 9, ata 7, dispensa 4, lei 2.

Entidades extraídas por regex: 478 valores, 308 processos, 89 CNPJs, 25 contratos.

**Precisão medida à mão.** Só a edição de 18/09/2026 foi contada ato a ato
(li o texto inteiro): 50 atos reais; o parser produziu 51, todos com a
fronteira certa e o tipo esperado. A diferença é o `CHAMAMENTO PÚBLICO Nº
04/2026` seguido de `ATA DA SESSÃO PÚBLICA...`, que é um ato só e virou dois.
Nas outras edições conferi por amostragem os títulos gerados (todos os
"outro" e os cabeçalhos de cada edição), não o total.

O que cai em `outro` é, de fato, miscelânea sem tipo próprio: termos de
apreensão de animais, notificações ambientais, corrigendas, termos de
aprovação de prestação de contas, concessão de licença ambiental, convocações
e o bloco `Continuação do D.O.E.`. Não vi ato de nomeação, contrato ou
licitação classificado como `outro` nas amostras.

**Erros conhecidos.**

- Cabeçalho ausente no texto extraído (2024-03-15: `DECRETO N.º 104/2024` não
  sai do pdftotext); a ementa fica colada ao anexo do decreto anterior. Sem OCR
  não há como recuperar.
- Anexo `Continuação do D.O.E.` começa no meio de uma portaria (o cabeçalho
  ficou fora do PDF) e termina com `Port. nº N/AAAA` sem corpo: o primeiro
  pedaço vira um ato `outro` com título "Continuação do D.O.E. em ..." e o
  último vira uma portaria só com título.
- `TERMO DE APREENSÃO ADMINISTRATIVA Nº:` com o número na linha seguinte fica
  sem número no título (2026-03-31).
- Sigla de órgão seguida do nome por extenso (`SMTC` + `SECRETARIA MUNICIPAL DE
  TURISMO E CULTURA`) não é reconhecida como seção e entra no fim do ato
  anterior.
- Um ato só com `CHAMAMENTO` + `ATA` vira dois (acima).
- Tabelas (anexos orçamentários, atas de registro de preços, listas de
  nomeação) ficam no corpo célula por linha; a busca acha os nomes, mas o
  trecho fica feio.
- Capa e expediente são descartados de propósito (lista de secretários em toda
  edição faria todo nome de secretário casar com todas as edições).

## 3. O que ficou pronto, com os comandos para reproduzir

Pré-requisitos: Docker rodando, Go 1.22+, Node 20+, `pdftotext`, `terraform`.

```bash
cp .env.example .env
make up setup migrate               # migrations 001, 002 (unaccent/pg_trgm), 003 (act_entities)
make lint test                      # vet, gofmt, unitários (pkg, api, scraper)
make test-integration               # Postgres real, banco diario_test criado pelo teste
make run-api                        # :8080     (terminal 1)
make run-worker                     # :8081     (terminal 2)
```

**Ingestão manual (tarefa 2).**

```bash
./scripts/fetch-editions.sh                     # baixa 7 edições + texto em services/api/testdata/editions/
make ingest FILE=services/api/testdata/editions/2026_09_18.pdf DATE=2026-09-18 EDITION=1771
```

Grava o PDF no fake-gcs, publica `gazette.fetched.v1` pelo mesmo caso de uso do
scraper (`FetchEditions` com um `EditionSource` de arquivo local) e o worker
indexa. `docs/parser-findings.md` documenta a estrutura real do Diário.

**Parser (tarefa 3).** `pdftotext` sem `-layout`; cabeçalhos reais; tipos novos
`despacho`, `resolucao`, `edital`, `ata`; fixtures reais anonimizadas em
`services/api/testdata/fixtures/`; teste de medição nas edições reais
(`PARSER_DUMP=1 go test -run Measure -v ./internal/adapters/parser/` lista os
títulos gerados).

**Busca (tarefa 4).** `002_unaccent_trigram.sql`: configuração
`portuguese_unaccent`, coluna `search` regenerada, índice GIN trigram sobre
`unaccent_immutable(body)`. As consultas casam por `tsquery` **ou** por
substring sem acentos. Verificado com dados reais:

```bash
curl -G localhost:8080/v1/acts --data-urlencode 'q=marcio de carvalho ribeiro'   # acha MARCIO DE CARVALHO RIBEIRO
curl -G localhost:8080/v1/acts --data-urlencode 'q=conceicao'                    # acha CONCEIÇÃO
curl -G localhost:8080/v1/acts --data-urlencode 'q=51.903.675'                   # trecho de CNPJ (trigram)
```

Observação: várias palavras são combinadas com E, não como frase; "marcio de
carvalho ribeiro" também acha "LEONARDO RIBEIRO DE CARVALHO". Aspas fazem busca
por frase (`websearch_to_tsquery`).

**Entidades (tarefa 5).** Porta `EntityExtractor` separada do `ActParser`
(justificativa em `services/api/internal/core/ports/ports.go`: segmentar e
extrair evoluem em ritmos diferentes; na fase 2 o extrator pode virar modelo sem
tocar na segmentação). Tabela `act_entities`; endpoints:

```bash
curl localhost:8080/v1/entities/cnpj/32.538.167/0001-05      # ou só dígitos; 400 se não tiver 14 dígitos
curl 'localhost:8080/v1/stats/acts?group=month&type=nomeacao&from=2026-01-01'
```

**Front (tarefa 6).** `/empresa/{cnpj}` (linha do tempo, total citado, contagem
por tipo); resultados da busca trazem `cnpjs` (da tabela de entidades) linkados
para essa página e link para o PDF original. `cd apps/web && npm ci && npm run
typecheck && npm run build`. Verificado em navegador headless (Playwright):
busca "lona viva" → link do CNPJ → página da empresa, sem erros de JS. A
extensão do Chrome não estava conectada nesta sessão, então não houve
verificação visual interativa além das capturas de tela.

**Scraper (tarefa 7).** Estrutura real: `POST index` com `DataInicial`,
`DataFinal`, `Termo` lista as edições (5 por página, `?NumeroPagina=N`); PDFs em
`diario/AAAA_MM_DD.pdf`. Sem JavaScript, sem bloqueio ao User-Agent
identificado, 1 requisição por vez com 2s de pausa. Rodado contra o site real:

```bash
make run-scraper LOOKBACK_DAYS=7
# {"found":5,"stored":5,"skipped":0,"failed":0} — 14 a 18/09/2026; o worker indexou 19, 65, 58, 54 e 51 atos
```

Atenção: `LOOKBACK_DAYS=7 make run-scraper` (variável de shell) **não** funciona,
porque o Makefile inclui o `.env` e exporta `LOOKBACK_DAYS=3` por cima; passe
como variável do make. O número da edição não existe no site; o worker o lê do
texto do PDF (`EDIÇÃO N°1.771`).

Alerta de ponta a ponta com dados reais: `POST /v1/subscriptions` ("lona
viva") → token no log da API → `POST /v1/subscriptions/confirm` → reingestão da
edição 1771 → e-mail (modo log) no worker com 1 resultado.

**Infra (tarefa 8).** `terraform fmt -check -recursive infra` limpo;
`terraform -chdir=<dir> init -backend=false && validate` passa em `bootstrap`,
`envs/dev` e `envs/prod` (google 6.50.0, neon 0.18.0; `.terraform.lock.hcl`
versionados). `lifecycle_rule { age = 30 → SetStorageClass ARCHIVE }` no bucket
de gazetas. Correção: `nonsensitive(var.resend_api_key != "")`.

## 4. O que NÃO ficou pronto e por quê

| Item | Motivo | Próximo passo concreto |
| --- | --- | --- |
| `terraform plan` em dev | Sem credenciais GCP/Neon nesta máquina. | Rodar `terraform -chdir=infra/envs/dev init -backend-config=bucket=... && terraform plan` com o `terraform.tfvars` preenchido e conferir se a lifecycle rule e o `nonsensitive` são aceitos pela API. |
| Precisão do parser medida em mais de uma edição | Contar à mão 6.000 linhas por edição não coube; só a de 18/09 foi contada inteira. | Contar mais duas edições (uma de 2024 e a de 31/08, com 143 atos) e registrar em `expectedActs` no `measure_test.go`. |
| Órgão (secretaria) por ato | Fora do escopo pedido; o parser já detecta a sigla como seção mas não a guarda. | Campo `organ` em `acts` (migration + `domain.Act`) preenchido pelo parser; filtro `?organ=` na API. |
| Semântica dos valores | A soma na página da empresa junta mensal, global, unitário e por exercício. | Guardar o rótulo anterior (`VALOR MENSAL:`, `VALOR GLOBAL:`) em `act_entities.value` e somar só globais, ou expor a lista para o usuário. |
| Validação de dígito verificador do CNPJ | Decisão: o Diário publica CNPJ com erro e o jornalista precisa achá-lo mesmo assim. | Se virar problema, marcar (não descartar) CNPJs inválidos. |
| Fallback do scraper se a busca do site cair | Só há um caminho (busca por termo). | Implementar `HEAD diario/AAAA_MM_DD.pdf` por dia da janela quando a listagem falhar (200 = existe, 500 = não). |
| Textos reais das edições no repositório | Contêm nomes de pessoas físicas; ficaram fora do git. | Manter fora; o script de download reproduz em ~30s. |
| Trecho da página da empresa corta no meio da palavra | `substr` fixo de 120 caracteres em volta do CNPJ. | Ajustar às fronteiras de palavra em `ReportByEntity` ou usar `ts_headline` com o CNPJ como consulta. |
| `make setup` esconde erros dos emuladores | `\|\| true` no script. | Trocar por verificação explícita (GET do recurso após criar). |
| Deploy e CI na nuvem | Não há projeto GCP. | Depois do bootstrap, o CI (já passa localmente: lint, test, integration, web, infra) pode rodar como está. |

## 5. Riscos para a fase 2 (extração com IA)

Regular o bastante para regex (já feito ou trivial):

- **CNPJ**: sempre `NN.NNN.NNN/NNNN-NN` (57/57 no formato padrão; 2 variações com
  espaço ou ponto faltando).
- **Valor em reais**: `R$ N.NNN,NN`; o único ruído é `VALOR (R$ 1)` em
  cabeçalho de tabela (tratado exigindo centavos).
- **Número de processo**: dois formatos estáveis (`NN.NNNNN/AAAA-D` do SEI e
  `NNNN/AAAA` legado), 308 extraídos.
- **Número de contrato/ata/pregão/portaria/decreto**, **matrícula** (`Mat.:
  131360`, `matrícula 22.685`), **datas por extenso** (`a contar de 17 de
  setembro de 2026`), **símbolo do cargo** (`CC-1`, `SSM`), **número da edição**.
- **Verbo do ato de pessoal** (Nomeia/Exonera/Designa/Torna sem efeito): vem
  em linha própria ou logo após `RESOLVE:`.

Precisa de modelo (ou de muito mais regra do que vale a pena):

- **Nome + cargo + órgão em nomeações coletivas**: vêm em tabela que o
  pdftotext devolve célula por linha e fora de ordem (`MAT. / 130118 / NOME /
  CARGO / PAULA ... / 131198 CARLA ...`). Ligar matrícula a nome a cargo exige
  reconstruir a tabela; um modelo com o layout (ou `-layout` só nesse bloco) é
  o caminho.
- **Fronteira do nome de pessoa física** em texto corrido: `Readaptar, pelo
  período de 01 (um) ano, Nome Sobrenome, matrícula 22.685, ocupante do cargo
  efetivo Professor Docente II QD SUP-22H`; o nome está em caixa mista e o cargo
  vem depois de "ocupante do cargo". Regex por âncoras (`, matrícula`) cobre
  parte; um NER treinado no Diário cobre o resto.
- **Semântica dos valores** (mensal × global × unitário × por exercício) e
  **partes do contrato** (quem contrata, quem é contratado): dependem do rótulo
  e do contexto (`PARTES: MUNICÍPIO ... e EMPRESA ...`).
- **Objeto do contrato/licitação**: texto livre após `OBJETO:`; extração
  simples por regex, mas normalização/sumário pede modelo.
- **Atas de registro de preços**: dezenas de itens com marca, unidade,
  quantidade e preços em tabela quebrada; recompor exige layout.
- **Cabeçalhos que faltam** no texto extraído (decreto sem título) e
  **títulos quebrados** em uma palavra por linha: um classificador de
  "início de ato" seria mais robusto que a lista de regex.
- **Dados pessoais**: CPF sai mascarado no Diário, mas nomes e matrículas não.
  Qualquer extração estruturada de pessoas físicas precisa respeitar a regra
  já registrada no README (mostrar só o publicado, não criar perfis).
