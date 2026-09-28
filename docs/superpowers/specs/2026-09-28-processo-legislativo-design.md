# Processo legislativo da Câmara (SICAM) e tema ambiental

## Problema

A base sabe quais leis foram sancionadas (`norms`, do SIAPEGOV) e o que a
Prefeitura publica para aplicá-las (licenças, autos de infração, atas do
conselho no Diário), mas não sabe o que acontece antes: que projeto foi
apresentado, por quem, em que comissão está parado, se foi votado,
arquivado ou enviado à sanção. A pergunta que motivou a entrega foi "como
andam as leis de proteção ambiental em São Gonçalo", e ela não tem resposta
sem a tramitação.

O roadmap marcava o SICAM como bloqueado porque a API de busca
(`POST https://api.sicam.app/pesquisar`) responde 403 fora do site. A área
pública, porém, é aberta e o `robots.txt` a libera.

## Fonte (conferida em 28/09/2026)

`https://sg.processolegislativo.com.br`, sistema SICAM da DB Nova.

- `robots.txt`: `Allow: /areapublica/`, `Disallow: /` para o resto, e
  `Sitemap: /sitemap.xml`.
- `sitemap.xml` é um índice; `sitemap-processos-1.xml` e `-2.xml` listam
  50.139 processos (`/areapublica/processo/{numero}-{ano}`) de 2014 a 2026
  (7, 19, 1.509, 3.700, 3.244, 4.296, 3.721, 7.156, 4.542, 6.163, 5.802,
  5.553 e 4.426 por ano). O `lastmod` de todas as entradas é o dia da
  consulta, então não serve para coleta incremental.
- A página do processo é HTML servido direto (≈55 KB, ≈0,2 s): tipo, nº do
  documento (`PROJETO DE LEI Nº 270/2019`), processo (`3865/2019`),
  situação (`Ativo`, `Arquivado`), ementa, autores, data de apresentação,
  órgão ou comissão atual, última movimentação, última atualização, a
  linha do tempo inteira (data e hora, rótulo `Movimentado`/`Finalizado`/
  `Em andamento`/`Andamento`, texto e setor) e os pareceres das comissões
  (resultado, data, comissão, relator). Número sem processo devolve uma
  página sem `Processo:`.
- Tipos: cerca de 10% são projetos (de lei, lei complementar, resolução,
  emenda à Lei Orgânica), mensagens do Executivo e emendas; o resto são
  indicações, moções, títulos, medalhas e requerimentos.
- `GET /integracao/?Processos/{pagina}/{x}/{tipo}/json` lista só o ano
  corrente, 10 por página, com a última movimentação. Não é usado: o
  sitemap e a página do processo bastam.
- `GET /integracao/?Pautas/{ano}/json` e `?Pauta/{id}/Json` trazem as
  sessões e os processos de cada pauta. Ficam para depois: a tramitação do
  processo já registra a leitura e a votação.
- Voto nominal: a página chama `https://api.sicam.app/votacao/v2`, a mesma
  API bloqueada. O placar sai da tramitação ("Aprovado - Votação única por
  22 votos"), não o voto de cada vereador.
- A sanção não aparece no SICAM: a última etapa de um projeto aprovado é
  "Enviado para PREFEITURA … Ofício nº". A lei vem da tabela `norms`, cujo
  autor, em ~770 leis, cita o projeto ("PROJETO DE LEI Nº 0133/2019
  VEREADOR …").

Na amostra de 2026, 103 dos 173 projetos de lei estão na Comissão de
Justiça e Redação aguardando parecer. Existe uma Comissão de Defesa do
Meio Ambiente.

## Desenho

### Domínio

- `Bill`: processo (número, ano), tipo como o SICAM escreve, rótulo do
  documento e seu número/ano, ementa, autores, apresentação, situação,
  órgão atual, última movimentação, última atualização na fonte, URL e
  quando foi lido. `BillEvent` (posição, quando, rótulo, texto, setor) e
  `BillOpinion` (posição, resultado, data, comissão, relator).
- `ParseBillPage(html)` lê a página; devolve `ErrBillNotFound` quando não
  há `Processo:`. As datas por extenso ("TERÇA-FEIRA, 6 DE JANEIRO DE
  2026 - 11:27") viram horário de São Paulo.
- `ParseProcessSitemap(xml)` devolve as chaves `numero-ano`.
- Tipos normativos (os que andam para virar norma): projeto de lei, de lei
  complementar, substitutivo, de resolução, de emenda à Lei Orgânica,
  mensagem e emenda. São os atualizados todo dia e o padrão da ferramenta.
- Fase, calculada na leitura a partir da situação e da tramitação, da mais
  forte para a mais fraca: `arquivado` (situação ou "Processo Arquivado"),
  `retirado`, `rejeitado`, `enviado_ao_executivo` ("Enviado para
  PREFEITURA", "Ao Executivo"), `aprovado` ("Aprovado - Votação"),
  `em_votacao` ("Para Votação", pauta), `em_comissao` (recebido,
  encaminhado ou aguardando parecer em comissão), `apresentado`.
  `dias_sem_movimentacao` conta do último evento até hoje.
- Projeto → lei: `BillReference(author)` lê "PROJETO DE LEI [COMPLEMENTAR]
  Nº 0133/2019" do autor da norma; a ligação é pelo tipo e pelo nº/ano do
  documento (não do processo), com certeza `provavel`.

### Coleta (job `sicam`)

- Adaptador `sicam`: sitemap e página do processo, 1 pedido por segundo,
  `User-Agent` do projeto, limite de tamanho de resposta.
- `LoadBills` com dois modos:
  - completo (`-full`): todas as chaves do sitemap;
  - diário (padrão): chaves do sitemap que não estão na base, mais os
    processos de tipo normativo que não estão arquivados, dos lidos há mais
    tempo para os mais recentes, até `-max` páginas (padrão 3.000).
- Grava em lotes de 500 páginas: upsert de `bills` e troca dos eventos e
  pareceres de cada processo do lote, numa transação; o bruto do lote
  (a parte `<main>` de cada página, em JSONL) vai para
  `raw/sicam_processos/AAAA/MM/DD/<run>-<lote>.jsonl.gz` com manifesto. Um
  processo que falha no parser conta em `failed` e não para a coleta; um
  número sem processo conta em `skipped`.
- `fetch_runs` com a fonte `sicam_processos`; `fontes` passa a mostrá-la.
- `make sicam [FULL=1] [MAX=n]`; na nuvem, job diário às 05:00 (o completo
  roda uma vez, fora do agendamento, com tempo máximo de 24 h).

### Tabelas (migration 030)

- `bills` (PK `process_number, process_year`), com índice de texto na
  ementa e nos autores (`portuguese_unaccent`) e índice por tipo.
- `bill_events` e `bill_opinions` (PK processo + posição, `ON DELETE
  CASCADE`).

### Tema ambiental

`domain/theme.go` com o tema `meio_ambiente`, escrito como regra e
devolvido em texto pela ferramenta:

- termos (sem acento, por raiz): meio ambiente, ambiental, arboriz, árvore,
  poda, supressão vegetal, floresta, reflorest, mata atlântica, manguez,
  área de proteção, APP, unidade de conservação, parque natural, fauna,
  flora, proteção/bem-estar/maus-tratos de animais, resíduo, lixo,
  reciclag, coleta seletiva, compostag, entulho, saneamento, esgoto,
  drenagem, recurso hídrico, nascente, poluição, ruído/poluição sonora,
  licenciamento ambiental, clima, sustentáv, educação ambiental,
  agrotóxico, queimada, desmatamento, energia solar, plástico descartável;
- processos: ementa com um dos termos, ou tramitação/parecer na Comissão
  de Defesa do Meio Ambiente;
- normas: ementa com um dos termos;
- atos do Diário: órgão ambiental (SEMMA, SEMA, SEMMADU, COMMADS, PROMEA);
  SEMMATRAN só quando o texto não é de trânsito (CORIM, táxi, trânsito,
  transporte, estacionamento, interdição de via, ônibus); ou título com um
  dos termos.

O tema é filtro de leitura, sem coluna nova, e a ferramenta devolve a
regra aplicada.

### Leitura

- `GET /v1/bills` (lista com filtros) e `GET /v1/bills/{numero}-{ano}`
  (processo com tramitação, pareceres e a lei ligada).
- Ferramenta `proposicoes` no MCP: por processo (`5564/2025`) ou por
  documento (`tipo` + `numero` 270/2019); ou lista com `texto`, `autor`,
  `tipo` (padrão: normativos; `todos` inclui indicações e moções),
  `situacao`/`fase`, `parado_ha_dias` (mínimo sem movimentação), `tema`,
  `de`/`ate` (apresentação), paginada (até 20). Cada item traz fase, dias
  sem movimentação, órgão atual, última movimentação, link da página e,
  quando houver, a lei que resultou. O processo sozinho traz a
  tramitação e os pareceres. Contagem por fase no topo da lista.
- `norma` ganha `tema` na busca por texto e devolve o projeto de origem
  quando a ligação existe.
- `buscar_atos` e `agrupar` ganham `tema`.
- Página `/proposicoes` no site: lista com os mesmos filtros e página do
  processo com a linha do tempo, os pareceres, a lei e a busca da lei no
  Diário.
- Recurso `diario-sg://fontes` e `docs/fontes/README.md`: SICAM passa para
  "na base", com o que ficou de fora.

## Fora

- Voto de cada vereador (API bloqueada; pedir acesso à Câmara ou à DB
  Nova).
- Texto integral e anexos (PDF) dos projetos.
- Pautas e atas das sessões como tabela própria.
- Perfil de vereador com as proposições (a ferramenta filtra por autor).
- Contornar o bloqueio da `api.sicam.app` imitando o navegador.

## Testes

- Parser com páginas reais gravadas: projeto arquivado por fim de mandato,
  mensagem aprovada e enviada ao Executivo, indicação, número inexistente.
- Sitemap real reduzido.
- Fase: um caso real por fase.
- Tema: ementas reais que entram e que não entram ("ambiente escolar",
  "ambiente laborativo", "produtos de origem animal" não entram).
- Ligação projeto → lei com autores reais do SIAPEGOV.
- Integração: carga de um lote, troca de eventos, filtros da lista.
