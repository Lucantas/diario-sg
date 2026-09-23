# Exportação da busca (Entrega 1)

Data: 2026-09-23. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"). O "pronto quando" da entrega é um jornalista sair de uma
busca com um CSV e uma citação que aponta a página exata do PDF
arquivado; a citação veio na 1b, este PR entrega o CSV.

## Objetivo

Qualquer busca do site (termo, tipo, órgão, período, faixa de valor) pode
ser baixada em **CSV** ou **JSON**, com o texto completo de cada ato e os
dados para citar: edição, data, páginas, links do PDF (original e
arquivado, já na página) e SHA-256.

## Fora do escopo

- Dump completo e periódico da base (PR seguinte).
- Exportar a página da empresa.
- Formatos XLSX ou Parquet.

## Fatos que orientam o desenho

- 153.750 atos na base local; corpo médio de 1,1 mil caracteres, mediana
  447, percentil 99 de 11 mil, máximo de 1 milhão (anexos colados).
- `ActFilter.Normalize` limita `Limit` a 100, que é o certo para a busca
  paginada e errado para exportação.
- A API conhece a URL pública do site (`PUBLIC_WEB_URL`, usada nos
  e-mails), então consegue escrever links absolutos para a cópia
  arquivada.

## Decisões

1. **Teto de 10.000 atos por exportação**, na ordem da busca. Cobre
   qualquer recorte razoável (um ano de contratos de uma secretaria) e
   limita a resposta a dezenas de MB. Quando a busca tem mais que isso, a
   resposta diz (`X-Export-Truncated: true`, `X-Total-Count`, e no JSON
   `truncated`), e o front avisa antes do download. A base inteira fica
   para o dump.
2. **CSV para Excel em português**: separador `;`, UTF-8 com BOM, fim de
   linha CRLF, decimal com vírgula. Assim abre direto no Excel e no
   LibreOffice configurados em pt-BR; quem usa pandas passa `sep=";"`.
3. **Proteção contra injeção de fórmula**: célula de texto que começa com
   `=`, `+`, `-`, `@`, tab ou CR ganha um apóstrofo na frente (recomendação
   da OWASP). O texto dos atos vem de fora e vai ser aberto em planilha.
4. **Streaming**: as linhas saem do banco direto para a resposta, sem
   montar a lista na memória. O cabeçalho HTTP (total e truncamento) é
   escrito quando chega a primeira linha, que já traz o total
   (`count(*) OVER ()`).
5. **Consulta própria**, sem `ts_headline`: a exportação leva o corpo
   inteiro, e gerar trecho destacado para 10 mil atos custaria segundos à
   toa. Filtros e ordenação são os mesmos da busca (mesmo `filterSQL` e
   mesma cláusula de ordem).

## Desenho

### Domínio e caso de uso

- `domain.ExportLimit = 10000`.
- `usecase.ExportActs.Execute(ctx, f, yield func(domain.ActHit, total int) error) error`:
  normaliza o filtro, troca `Limit` por `ExportLimit` e `Offset` por 0, e
  repassa ao repositório.
- `ports.ActRepository.Export(ctx, f, yield)`; o `ActHit` exportado vem
  com `Body` preenchido.

### Postgres

`Export` usa o mesmo `WHERE` e a mesma ordem de `Search` (a ordem sai
para uma constante compartilhada), seleciona `a.body`, CNPJs e valores, e
chama `yield` a cada linha. Erro do `yield` (cliente desconectou) encerra
a consulta.

### API

`GET /v1/acts/export?format=csv|json&<filtros da busca>`

- `format` ausente é `csv`; outro valor é 400.
- `Content-Disposition: attachment; filename="diario-sg-busca-AAAA-MM-DD.csv"`
  (ou `.json`), com a data do dia.
- Cabeçalhos `X-Total-Count` e `X-Export-Truncated`.
- CSV, colunas: `data;edicao;extra;tipo;orgao;orgao_nome;titulo;pagina_inicio;pagina_fim;valores_reais;cnpjs;pdf_original;pdf_arquivado;sha256_pdf;texto`.
  `extra` é `sim`/`não`; `valores_reais` é `1200,00 | 300,00`; `cnpjs`
  separados por ` | `; links já com `#page=N` quando há página.
- JSON: `{"total":N,"truncated":bool,"items":[…]}`, cada item com os
  campos da busca (menos `snippet`) mais `body` e `archived_pdf_url`.
- `API.PublicWebURL` vem de `cfg.PublicWebURL`.

### Front

Abaixo da contagem de resultados: "Baixar CSV" e "JSON", links para
`/api/v1/acts/export?…` com os mesmos parâmetros da busca (sem `limit` e
`offset`). Com mais de 10.000 atos, o texto avisa que só os 10.000
primeiros entram.

## Testes

- Unitário (`http`): `csvCell` (fórmulas, texto comum), `reaisBR`
  (centavos → `1200,00`), linha CSV completa de um ato com e sem página.
- Unitário (`usecase`): `ExportActs` passa `Limit = ExportLimit` e
  `Offset = 0` mesmo com filtro de busca paginada.
- Integração: exportar a edição de três contratos em CSV (BOM, cabeçalho,
  3 linhas, `;`, valores, link arquivado absoluto com `#page=1`) e em JSON
  (total, `truncated=false`, `body` completo); filtro de valor respeitado;
  `format=xml` → 400.
- Front (Vitest): `exportUrl(state, "csv")` sem `limit`/`offset` e com os
  filtros.

## Riscos

- **Resposta grande** no Cloud Run: 10 mil atos com anexos podem passar
  de 50 MB; o streaming evita memória, e o limite de 32 MB do Cloud Run
  vale só para resposta sem streaming (HTTP/1 chunked não tem esse teto).
- **Busca pesada**: exportar uma busca textual ampla repete o custo da
  busca (o `ILIKE` sem acento, já lento hoje) uma vez, sem paginação.
