# MCP mínimo (etapa A)

Data: 2026-09-23. Etapa A de `docs/plano-fontes-publicas.md`: servidor MCP
sobre a API atual, com chave por usuário (decisão 1 do plano).

## Objetivo

Quem usa uma IA com suporte a MCP (Claude Code, Claude Desktop, Cursor…)
gera uma chave no site, pluga o servidor do Diário SG e pergunta em
linguagem natural. A IA busca e lê os atos pelas ferramentas e cita a
edição, a página e a cópia arquivada.

## Fora do escopo

- OAuth. Os conectores do claude.ai e do ChatGPT exigem OAuth; nesta
  etapa só funcionam clientes que mandam cabeçalho `Authorization`.
- Fontes além do Diário da Prefeitura, ferramentas `pagamentos`,
  `contratacoes`, `empresa`, `agente_politico` e `padroes` (etapas B a G).
- Recursos e prompts do MCP.
- Conta de usuário, e-mail na chave, painel de uso.
- CLI de administração das chaves (revogar por id fica por SQL até
  aparecer abuso).

## Fatos que orientam o desenho

- O SDK oficial em Go (`github.com/modelcontextprotocol/go-sdk` v1.8.0)
  tem transporte HTTP "streamable" com modo sem sessão (`Stateless`) e
  resposta JSON sem SSE (`JSONResponse`), e gera o JSON Schema das
  entradas e saídas a partir de structs. Ele pede Go 1.25.
- O middleware da API envolve a resposta num `statusRecorder` sem
  `Flush`; resposta JSON dispensa streaming.
- O Cloud Run escala a mais de uma instância; sessão em memória não
  sobreviveria. Sem sessão, qualquer instância atende.
- O `id` do ato muda na reindexação; edição + posição é estável.
- `usecase.SearchActs`, `GetGazette` e `GetCompany` já fazem o que as
  ferramentas precisam; a API pública já tem limitador de janela fixa.
- O site fica atrás do nginx, que manda `/api/` para a API.

## Decisões

1. **No mesmo binário e serviço da API**, em `POST /mcp` (no site,
   `https://<site>/api/mcp`). Sem infra nova. Chama os mesmos casos de
   uso que a API.
2. **Sem sessão e com resposta JSON** (`Stateless`, `JSONResponse`).
3. **Chave anônima gerada no site.** `POST /v1/mcp/keys` devolve a chave
   uma única vez: `dsg_` + 32 bytes aleatórios em base64url. O banco
   guarda só o SHA-256 e um prefixo de 8 caracteres para identificar a
   chave em log. Sem e-mail: a chave identifica o uso, não a pessoa
   (decisão 1 do plano). Criação limitada a 3 por hora por cliente e 50
   por hora por instância, com campo isca como no reporte.
4. **Revogação pela própria chave**: `DELETE /v1/mcp/keys` com a chave no
   `Authorization`. O site tem o formulário.
5. **Limite por chave**: 60 chamadas por minuto por chave e 600 por
   instância; chave ausente, inválida ou revogada é 401.
6. **Registro de uso por chave, dia e ferramenta** (`api_key_usage`),
   mais `last_used_at` na chave. Só contagem; nada dos argumentos.
7. **Ferramentas pequenas e com prova.** Nomes e campos em português.
   Todo ato traz `fontes` (edição, URL oficial com `#page=N`, cópia
   arquivada, página, SHA-256) e toda resposta traz `cobertura` (período
   coberto e lacunas conhecidas). Paginação obrigatória, até 20 atos por
   chamada.
8. **Citação pronta em Go**, no mesmo formato ABNT do botão "Citar este
   ato" do site.
9. **Go 1.25** no módulo `services/api`, no `go.work` e na imagem da API.

## Ferramentas

| Ferramenta | Entrada | Saída |
| --- | --- | --- |
| `buscar_atos` | `consulta`, `tipo`, `orgao`, `de`, `ate` (AAAA-MM-DD), `valor_min`, `valor_max` (reais), `limite` (1–20, padrão 10), `deslocamento` | `total`, `atos[]` (edição, posição, data, tipo, órgão, título, trecho, CNPJs, valores, avisos, fontes), `cobertura` |
| `ler_ato` | `edicao_id`, `posicao` | ato com texto completo, `citacao`, `fontes`, `cobertura` |
| `entidade` | `cnpj` | total citado, contagem por tipo, até 20 atos mais recentes, `cobertura` |
| `fontes` | — | fontes com período, número de edições e atos, última coleta e lacunas |

Todas com `readOnlyHint`. Erro de entrada (filtro inválido, CNPJ
inválido, ato inexistente) volta como resultado de ferramenta com
`isError`, com a mensagem do domínio.

Lacunas declaradas do Diário da Prefeitura: PDF só com imagem não é lido;
a separação em atos pode errar (os avisos de cada ato dizem onde); o
Diário da Câmara ainda não é coletado.

## Desenho

- Migration `007_api_keys.sql`: `api_keys (id, key_hash unique,
  key_prefix, created_at, revoked_at, last_used_at)` e `api_key_usage
  (key_id, day, tool, calls, PK (key_id, day, tool))`.
- `domain`: `APIKey`, `NewAPIKeySecret()`, `HashAPIKey()`,
  `ErrUnauthorized`; `Coverage` (primeira e última edição, edições, atos,
  última coleta).
- `ports`: `APIKeyRepository` (`Create`, `FindActive`, `Revoke`,
  `RecordUse`) e `CoverageReader` (`Coverage`).
- `usecase`: `APIKeys` (`Issue`, `Authenticate`, `Revoke`, `RecordUse`),
  `ReadAct` (edição + posição → edição e ato), `SourceCoverage`.
- `postgres`: `APIKeyRepo`; `GazetteRepo.Coverage`.
- `presentation/ratelimit`: o limitador sai do pacote `http` para ser
  usado pelos dois.
- `presentation/mcp`: servidor do SDK com as quatro ferramentas,
  middleware de chave e limite, citação e fontes.
- `presentation/http`: `POST` e `DELETE /v1/mcp/keys`; `/mcp` montado
  no roteador.
- Front: página `/mcp` com o que é, botão de gerar chave (mostrada uma
  vez, com botão de copiar), configuração pronta para Claude Code, Claude
  Desktop e Cursor, e formulário de revogar. Link no rodapé da busca.

## Testes

- `domain`: segredo com prefixo e tamanho certos, dois segredos
  diferentes, hash estável.
- `usecase`: `Issue` guarda só o hash; `Authenticate` recusa chave
  desconhecida e revogada; `ReadAct` acha pela posição e dá `ErrNotFound`
  para posição inexistente.
- `ratelimit`: os testes atuais, movidos.
- `mcp` (unitário): citação igual à do site para um ato com e sem
  página; fontes com `#page=N`; conversão de reais para centavos.
- Integração (Postgres real, cliente MCP do SDK): sem chave é 401; gerar
  chave, listar as quatro ferramentas, buscar, ler o ato achado, consultar
  CNPJ, `fontes` com o período da base; uso registrado por ferramenta;
  revogar e receber 401; a quarta chave na hora, do mesmo cliente, é 429.
- Front (Vitest): trechos de configuração levam a URL e a chave.
