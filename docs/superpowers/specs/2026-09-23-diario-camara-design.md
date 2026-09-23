# Diário Oficial da Câmara (etapa C1)

Data: 2026-09-23. Primeiro subprojeto da etapa C de
`docs/plano-fontes-publicas.md`. Decisão na ADR 0007.

## Objetivo

Coletar, indexar e tornar pesquisável o Diário Oficial Eletrônico da
Câmara Municipal de São Gonçalo, com a mesma busca, alerta, citação,
exportação, dump e MCP do Diário da Prefeitura.

## Fora do escopo

- Proposições do SICAM, agentes políticos e subsídios (próximos
  subprojetos da etapa C).
- Edições da Câmara anteriores a 2020-10-04 (ver lacunas).
- Alerta filtrado por fonte: o alerta continua valendo para os dois
  diários.
- OCR de PDF só com imagem.

## Fatos que orientam o desenho

- As edições estão em
  `https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/AAAA-MM-DD.pdf`.
  O servidor responde 404 nos dias sem edição e aceita `HEAD`. A primeira
  edição achada por essa URL é de 2020-10-04. As de 2018 a 2020 existem
  ("Ano-01"), mas com nomes de arquivo que não seguem a data.
- A busca do site da Câmara fica atrás de um WAF (ModSecurity) que recusa
  robôs. O coletor não tenta contorná-lo e usa só a URL por data.
- Alguns dias devolvem um PDF de 696 bytes sem texto: a edição vira uma
  edição com zero atos.
- Cabeçalho de toda página: `PODER LEGISLATIVO`, `CÂMARA MUNICIPAL DE SÃO
  GONÇALO`, `São Gonçalo, <data>`, `Ano-08 / Edição – 138` (também
  `Edição - 138` e `Edição 138`), `DIÁRIO OFICIAL ELETRÔNICO – D.O.E` (com
  ou sem travessão), `LEI MUNICIPAL 855/2018 DE 05/07/2018` (com ou sem
  ponto) e uma linha de sublinhados. Rodapé: `Página N de M`.
- Numa amostra de 88 edições (2020-11 a 2026-09), o parser da Prefeitura
  separa bem portarias, resoluções, extratos, avisos e termos. Só falham o
  número da edição, que fica vazio, e 16 atos intitulados
  `PODER LEGISLATIVO`, que são continuação de página.
- O número da edição recomeça a cada ano ("Ano-08").

## Desenho

### Dados (migration 009)

- `gazettes.source text NOT NULL DEFAULT 'diario_prefeitura'
  CHECK (source IN ('diario_prefeitura','diario_camara'))`, e o índice
  `(source, published_at)`.
- `link_diario_acts(gazette)` passa a gravar a fonte da edição em
  `entity_links.source`.

### Domínio (API)

- `domain.SourceDiarioCamara = "diario_camara"`, `domain.ValidSource`,
  `domain.SourceName` ("Diário Oficial do Município de São Gonçalo" /
  "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo").
- `Gazette.Source`, `ActHit.Source`, `ActFilter.Source`.
- `Coverage` passa a ser por fonte: `[]SourceCoverage` com fonte, período,
  edições, atos, última indexação e última coleta.

### Parser

- `parser.ForSource(source) Regex`: a Câmara soma o seu ruído de página e
  o seu padrão de número de edição aos da Prefeitura.
- `TERMO DE HOMOLOGAÇÃO` e `TERMO DE ADJUDICAÇÃO` viram `licitacao` nas
  duas fontes.
- `usecase.IndexGazette` e `ReindexGazettes` recebem um
  `ports.ActParsers` (`For(source) ActParser`).

### Coletor

- Variável `SOURCE` (`diario_prefeitura`, padrão, ou `diario_camara`); a
  `SOURCE_URL` padrão depende dela.
- Adaptador `source/cmsg`: `ListEditions` faz `HEAD` em cada dia do
  período, com 2 s entre pedidos; 200 é edição, 404 é dia sem edição,
  outro status é erro. `Download` faz `GET`.
- `Edition.Source`; o caminho no bucket da Câmara segue a ADR 0005:
  `raw/diario_camara/AAAA/MM/DD/AAAA-MM-DD.pdf`. O da Prefeitura não muda.
- `gazette.fetched.v1` ganha `source` opcional (ausente é a Prefeitura);
  `fetch.completed.v1` aceita `diario_camara`.
- Terraform: job `scraper-camara` com `SOURCE=diario_camara` e agendamento
  próprio; o deploy atualiza a imagem dos dois jobs.

### API, MCP e site

- `GET /v1/acts` e a exportação aceitam `source`; `organ` continua só da
  Prefeitura. Atos e edições trazem `source` e `source_name`.
- MCP:
  - `buscar_atos` aceita `fonte`;
  - todo ato traz a fonte, e a citação usa o nome da fonte;
  - `fontes` lista as duas, com cobertura e última coleta de cada uma,
    e as lacunas da Câmara.
- E-mail do alerta e RSS: o assunto e o item dizem de qual diário é o ato.
- Site:
  - filtro "Fonte" na busca;
  - selo "Câmara" nos resultados da Câmara;
  - citação com o nome da fonte;
  - textos da home e da página de dados citam os dois diários.

## Lacunas declaradas

- Câmara antes de 2020-10-04.
- Mais de uma edição da Câmara no mesmo dia: só a do arquivo `AAAA-MM-DD`
  é coletada.
- PDF sem texto vira edição com zero atos.

## Testes

- Parser: fixture com cabeçalho e rodapé da Câmara em duas páginas: o
  ato continua na página 2 sem ato `PODER LEGISLATIVO`; número da edição
  nas três grafias; a Prefeitura não muda (fixtures atuais).
- Coletor: `cmsg` com servidor de teste (200, 404, 500, cancelamento);
  caminho do bucket da Câmara; `FetchEditions` publica a fonte na edição
  e na coleta; config valida `SOURCE`.
- Worker: evento sem `source` indexa como Prefeitura; com `diario_camara`
  usa o parser da Câmara.
- Integração:
  - edição da Câmara indexada com fonte, ligações com
    `source = 'diario_camara'`;
  - busca com `source`;
  - `fontes` com as duas fontes;
  - citação da Câmara no `ler_ato`.
- Site: citação por fonte; estado da busca com `source`.
- Manual: backfill local da Câmara desde 2020-10-04, com contagem por tipo
  e amostra de atos lidos.
