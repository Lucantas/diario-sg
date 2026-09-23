# Roadmap — de buscador do Diário a ferramenta de investigação

Público: jornalistas, mandatos e ativistas que investigam a Prefeitura de São
Gonçalo. Nos dados, **nada depende de IA** (sem LLM, embeddings ou
classificação estatística): todo cruzamento é feito por chave determinística
(CNPJ, nº de processo, nº de contrato, CPF parcial + nome) e todo alerta é uma
regra escrita, que mostra a regra e os atos que a acionaram. Perguntas em
linguagem natural ficam com a IA de cada usuário, pelo servidor MCP (ver
`docs/plano-fontes-publicas.md`): nós servimos dados e provas, não texto
gerado.

O plano de fontes (`docs/plano-fontes-publicas.md`, com o levantamento em
`docs/fontes/README.md`) detalha as fontes de cada entrega e adiciona o
servidor MCP desde cedo.

A ideia central: o Diário diz o que foi **anunciado**. A investigação nasce do
cruzamento com o que foi **pago**, com **quem é dono** do fornecedor, com
**quem doou** para quem e com **quem foi punido**. `act_entities` (migration
003) já é o embrião disso; as entregas abaixo transformam as chaves extraídas
no centro do produto.

| Entrega | Tema | Fontes novas | Depende de |
| --- | --- | --- | --- |
| 0 | Base completa | — | — |
| 1 | Citável e exportável | — | 0 |
| 2 | Seguir o dinheiro dentro do Diário | — | 1 |
| 3 | Quem é o fornecedor | Receita (CNPJ), CGU (CEIS/CNEP/CEPIM) | 2 |
| 4 | Anunciado × pago | PNCP, Portal da Transparência de SG | 3 |
| 5 | Recorte político | TSE, Câmara Municipal, transferências federais, TCE-RJ | 3 |

Disponibilidade, formato e licença de cada fonte externa **ainda não foram
verificados**; o primeiro passo das entregas 3 a 5 é um levantamento curto
(`docs/fontes/<fonte>.md`: URL, formato, frequência de atualização, chaves,
limites de uso).

---

## Entrega 0 — Base completa

O scraper perde as **edições extraordinárias**. O site publica a extra do
mesmo dia em `diario/AAAA_MM_DD_1.pdf`, e `editionLinkRe`
(`services/scraper/internal/adapters/source/pmsg/source.go`) só aceita
`AAAA_MM_DD.pdf`; o domínio também assume uma edição por data. Exemplo: em
21/08/2024 a base tem a edição 1196, mas não a extra 1197. Na coleta do
Querido Diário até 30/08/2024 há 46 edições extras de 2020 em diante.

- [x] Aceitar o sufixo `_N` no link e tratar a edição pela URL, não pela
  data (commit `41b9d42`).
- [x] Backfill das extras de 2020 a hoje: 115 edições novas (117 extras
  no total, as mesmas da coleta do Querido Diário).
- [x] Listar pela letra "a" em vez de "Gonçalo": a continuação de
  13/12/2023 não tem cabeçalho e nunca era listada (commit `f76d820`).
  A base de 2020 em diante tem 1740 edições, as mesmas da coleta
  completa do raspador do Querido Diário.
- [x] Número da edição com `N°1. 362` (2025) e `| N.º 190 | em …`
  (2020-2021) no parser (commit `6d22d12`); 319 edições reindexadas.
- [x] 18 a 20/08/2020 (edições 158 a 160): cabeçalho com "em," coberto
  pelo parser.
- [x] `is_extra` em `gazettes`, coluna gerada a partir do sufixo `_N` da
  URL (migration 004), na API (edição e busca) e no front.
- [x] Backfill de 2010 a 2019 (22/09/2026): 2.561 edições encontradas e
  indexadas, nenhuma falha (278, 287, 291, 248, 245, 242, 241, 248, 238 e
  243 por ano), 78.832 atos. A base local vai de 05/01/2010 a hoje, com
  4.301 edições. Os PDFs antigos têm texto extraível e o mesmo layout de
  duas colunas; pesam 646 MB no total (o Postgres cresceu 167 MB).
- [ ] Nenhuma edição extra antes de 2020 foi listada. Não confirmei se o
  site não tinha extras nesse período ou se elas usam outro padrão de URL.
- [x] Tipos `corrigenda` e `prestacao_contas` (commit `c2f2397`). De 2010 a
  2019 o `outro` era 10% a 15% dos atos por ano, e 91% dele eram termos de
  aprovação de prestação de contas e corrigendas. Na base inteira, depois
  da reindexação de 22/09/2026, o `outro` caiu de 16.186 para 3.684 atos
  (6.973 corrigendas e 5.543 prestações de contas).

**Pronto quando:** a contagem de edições por ano bate com a listagem do site.

## Entrega 1 — Citável e exportável

Só com os dados atuais. É o que dá credibilidade para quem vai publicar.

- **Proveniência.** ✅ Página de cada ato (1a, commit `5950b8d..1741bf1`),
  link `…pdf#page=N`, SHA-256 do PDF, cópia arquivada servida pela API
  (`/v1/gazettes/{id}/pdf`) e botão "Citar este ato" com edição, data,
  página e os dois links (1b). Produção ganhou o job de reindexação
  (`reindex` no Cloud Run, disparado pelo workflow manual
  `reindex.yml`); ele precisa rodar uma vez depois do deploy, senão os
  atos de lá ficam com página nula.
- **Órgão.** ✅ Allowlist no parser e vazamento de órgão corrigidos
  (commit `6f28706`, base reindexada em 22/09/2026). O órgão vazava para
  seções sem sigla: o bloco de portarias abreviadas do gabinete herdava a
  última sigla vista (em 20/06/2016, `Port. nº 1360` e `1495` a `1498`
  saíam com `SUBCOMP`), e palavras em formato de sigla viravam órgão
  (`EXECUTIVO`, `TOTAL`, `NOME`, sobrenomes: 5.848 atos). Agora a
  portaria abreviada fica sem órgão e zera o órgão corrente,
  `Continuação do D.O.E.` também zera, e uma linha em formato de sigla
  fora da allowlist separa seção sem virar órgão. `RESOLUÇÃO “P” nº` e
  cabeçalhos com `nº` minúsculo passaram a abrir ato: 1.392 atos que
  antes ficavam colados ao anterior. Atos com órgão caíram de 112.543
  para 80.832; fora as portarias abreviadas, 74% dos atos têm órgão.
  ✅ Filtro por órgão na busca e `/v1/organs` (1c). O catálogo de 125
  siglas saiu do parser para `domain/organ.go`, com nome por extenso de
  119 delas; a fonte de cada nome, as siglas que talvez não sejam órgãos
  e as variantes estão em `docs/orgaos.md`. Sigla nova que a prefeitura
  criar precisa entrar no catálogo (e `make reindex`).
- **Busca de investigador.** ✅ Faixa de valor (a partir de
  `act_entities`), operador `OU`, filtros de período e URL permanente
  para cada consulta, com paginação.
- **Exportação.** ✅ CSV (pt-BR, para planilha) e JSON de qualquer busca,
  até 10.000 atos. ✅ Dump semanal da base em CSV compactado, com
  manifesto e página `/dados`. SQLite e Parquet prontos ficaram de fora:
  o LEIAME mostra como carregar o CSV no pandas e no Datasette.
- **RSS** ✅ por consulta (`/v1/feeds/acts`), ao lado do alerta por e-mail.
- **Qualidade visível.** ✅ Avisos em cada ato para os casos conhecidos
  do parser (portaria sem número, ato só com o título, ato com 10 páginas
  ou mais) e botão "Reportar erro", que grava numa fila lida com
  `make reports`. A "tabela quebrada entre páginas"
  (`docs/fase-1-relatorio.md` §2.3) ficou sem aviso: a regra candidata dá
  falsos positivos demais (ver `docs/decisoes-de-codigo.md`).

**Pronto quando:** um jornalista consegue sair de uma busca com um CSV e uma
citação que aponta para a página exata do PDF arquivado. Localmente, já
consegue.

Pendências que ficaram desta entrega:

- [ ] Nada da infra nova (job de reindexação, dump semanal, bucket
  público, leitura do bucket pela API) foi aplicado na nuvem: o
  environment `dev` do GitHub não tem as variáveis, e os workflows de
  infra e deploy falham na autenticação, antes do `terraform plan`. Foi
  validado só com `terraform validate` e `fmt`.
- [x] Busca textual ampla lenta (resolvido em 23/09/2026): a busca seleciona
  primeiro só ids e chaves de ordem, numa CTE materializada, e busca o
  texto, o destaque e as entidades só dos atos da página. Na base local:
  `contrato` de 3,9 s para 1,2 s, `prefeitura` de 3,1 s para 1,9 s, busca
  vazia de 2,4 s para 0,2 s, com os mesmos resultados. Termos que aparecem
  em atos muito grandes (`prefeitura`, `merenda OU alimentação`) ainda
  levam ~2 s: o custo que sobra é comparar o texto sem acento de cada ato
  encontrado.
- [x] Atualizar o Vite e o Vitest (resolvido em 23/09/2026): Vite 8,
  `@vitejs/plugin-react` 6 e Vitest 5, com Node 22 na imagem e no CI (o
  Vitest 5 exige). `npm audit` sem alertas.
- [x] Unificar as variantes de sigla (`docs/orgaos.md`), resolvido em
  23/09/2026: o filtro por órgão traz a sigla principal e as variantes, e a
  lista de órgãos soma as variantes na principal (de 125 para 107 órgãos).

## Etapa A do plano de fontes — MCP mínimo (entregue)

Servidor MCP em `POST /mcp` da API (`/api/mcp` no site), só leitura, com
`buscar_atos`, `ler_ato`, `entidade` (CNPJ) e `fontes`. Cada pessoa gera
uma chave anônima na página `/mcp`; o uso é contado por chave, dia e
ferramenta. Desenho em
`docs/superpowers/specs/2026-09-23-mcp-minimo-design.md`.

Pendências:

- [ ] OAuth: os conectores do claude.ai e do ChatGPT só aceitam servidor
  com login OAuth. Hoje funcionam Claude Code, Claude Desktop (via
  `mcp-remote`), Cursor e outros clientes que mandam o cabeçalho
  `Authorization`.
- [x] Revogar chave por abuso (resolvido em 23/09/2026): `make keys` lista as
  chaves com o uso dos últimos 30 dias e `make revoke-key PREFIX=…` revoga
  pelo prefixo que aparece no log.
- [ ] Não foi para a nuvem, pelo mesmo motivo da Entrega 1.

## Etapa B do plano de fontes — base comum (entregue)

`entities` e `entity_links` por cima de `act_entities`, com a certeza de
cada ligação; `fetch_runs` gravado pelo evento `fetch.completed.v1`;
convenção do arquivo bruto; ADRs 0004 (entidades), 0005 (coletas) e 0006
(LGPD). A ferramenta `entidade` do MCP aceita processo e contrato. Desenho
em `docs/superpowers/specs/2026-09-23-base-comum-design.md`.

Pendências:

- [ ] Na nuvem, o scraper passa a exigir `TOPIC_FETCH_COMPLETED`: aplicar
  o Terraform (fila nova) antes de publicar a imagem nova do scraper.
- [ ] Páginas de processo e contrato no site ficam para a Entrega 2, já
  lendo de `entity_links`.

## Etapa C1 do plano de fontes — Diário da Câmara (entregue)

O Diário Oficial Eletrônico da Câmara entra nas mesmas tabelas do da
Prefeitura, com `gazettes.source` (ADR 0007). Busca, exportação, RSS,
alertas, citação, dump, `entidade` e MCP valem para os dois; o site ganhou
o filtro "Diário". O job `scraper-camara` coleta pela URL por data. Na base
local, o backfill desde 2020-10-04 trouxe 984 edições e 4.824 atos; 78
edições têm o corpo escaneado e ficam sem atos (sem OCR). Desenho
em `docs/superpowers/specs/2026-09-23-diario-camara-design.md`.

Pendências:

- [ ] Edições de 2018 a 2020-10-03 existem, mas não por URL com a data; a
  busca do site da Câmara recusa robôs (WAF). Falta achar outra lista.
- [ ] Na nuvem: aplicar o Terraform (job e agendamento novos, migration
  009) antes do primeiro deploy com o job da Câmara, e rodar o backfill
  desde 2020-10-04 fora do agendamento (leva mais de uma hora; o job tem
  30 minutos).
- [ ] OCR das edições escaneadas (8% das edições da Câmara).
- [ ] SICAM (proposições) e agentes políticos e subsídios: próximos
  subprojetos da etapa C.
- [ ] Reindexar a produção: `TERMO DE HOMOLOGAÇÃO` e `TERMO DE
  ADJUDICAÇÃO` viraram licitação. Na base local (4.302 edições, 9 min), 166
  atos da Prefeitura passaram de `outro` para `licitacao`; nenhum outro
  tipo nem órgão mudou.

## Entrega 2 — Seguir o dinheiro dentro do Diário

Ainda sem fonte externa.

- **Página de processo e de contrato.** `/processo/{n}` e `/contrato/{n}`
  juntam, em ordem, licitação → homologação → extrato → aditivos → rescisão,
  pelo nº extraído.
- **Alerta por entidade.** Inscrição em CNPJ, processo ou contrato, além de
  termo livre.
- **Painéis.** Maiores fornecedores por valor, por secretaria e por ano.
- **Padrões para verificar** (regras fixas, cada uma com a regra em texto e
  os atos que a acionaram):
  - várias dispensas para o mesmo fornecedor/objeto abaixo do limite legal
    numa janela curta (possível fracionamento);
  - aditivos que somam mais de 25% do valor original do contrato;
  - contratação emergencial renovada seguidamente;
  - picos de nomeação e exoneração nos meses antes das eleições (2020 e 2024
    estão na base, dá para comparar com anos sem eleição).

Linguagem: sempre "padrão para verificar", nunca "irregularidade".

**Pronto quando:** cada regra tem teste com atos reais que a acionam e atos
reais parecidos que não acionam.

## Entrega 3 — Quem é o fornecedor

Primeira fonte externa. Começa por uma mudança de modelo: uma tabela de
**entidades** (CNPJ, processo, contrato e, na entrega 5, agente político) que
funciona como ponto de ligação; cada fonte tem adapter e tabelas próprias,
ligadas às entidades. O Diário continua sendo documento com busca textual; as
outras fontes são registros estruturados. Registrar em ADR.

- **Receita Federal — CNPJ (dados abertos).** Situação, data de abertura,
  CNAE, capital social, endereço, quadro de sócios. Carga mensal só dos CNPJs
  que aparecem no Diário.
- **CGU — CEIS, CNEP, CEPIM.** Sanções vigentes e históricas.
- **Página da empresa** enriquecida com tudo acima.
- **Novos padrões:** empresa aberta pouco antes do primeiro contrato; capital
  social muito menor que o valor contratado; fornecedores com sócio ou
  endereço em comum; fornecedor sancionado contratado.
- **ADR de LGPD** antes de publicar sócios: agentes políticos e secretários
  podem ter página própria; servidores são encontráveis na busca, sem perfil;
  sócios aparecem só dentro da página da empresa.

## Entrega 4 — Anunciado × pago

- **PNCP** (contratações do município, pela Lei 14.133) e **Portal da
  Transparência de SG** (empenho, liquidação, pagamento).
- Ligação por CNPJ + nº de contrato/processo, com o grau de certeza de cada
  ligação visível.
- **Novos padrões:** pagamento a fornecedor sem contrato publicado no Diário;
  contrato publicado sem nenhum pagamento; pago acima do contratado mais
  aditivos.
- Folha de pagamento só em agregados por cargo/órgão (LGPD).

## Entrega 5 — Recorte político

- **TSE:** candidatos, bens e prestação de contas (doações). Cruzamento sócio
  de fornecedor × doador. A Receita mascara parte do CPF dos sócios, então o
  casamento é nome + dígitos visíveis: determinístico, mas a página mostra a
  evidência do casamento e nunca afirma identidade.
- **Câmara Municipal:** leis, projetos e votações; perfil de vereador.
- **Transferências federais e emendas** destinadas ao município.
- **TCE-RJ:** apontamentos sobre contratos já presentes no Diário.

---

## Fora do produto: Querido Diário

O Querido Diário (Open Knowledge Brasil) não coleta São Gonçalo. O PR
[okfn-brasil/querido-diario#1256](https://github.com/okfn-brasil/querido-diario/pull/1256)
(2024) traz um spider aprovado por um revisor voluntário, mas nunca foi
mergeado e ficou desatualizado em relação ao upstream (pasta
`data_collection/` virou `querido_diario_raspadores/`, spiders migraram para
`async def start()`) e o número da edição deixou de ser extraído porque o site
passou a escrever `N°1.771` sem espaço. Contribuição com a versão atualizada:
[okfn-brasil/querido-diario#1550](https://github.com/okfn-brasil/querido-diario/pull/1550),
com crédito ao autor original e aviso no #1256. A coleta completa do
raspador serviu de gabarito para a nossa base (ver Entrega 0).
