# RSS por consulta (Entrega 1)

Data: 2026-09-23. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"): RSS por consulta, ao lado do alerta por e-mail.

## Objetivo

Toda busca do site (termo e filtros) tem um feed RSS com os 50 atos mais
recentes que casam com ela. Quem acompanha pelo leitor de RSS não precisa
dar e-mail.

## Fora do escopo

- Atom e JSON Feed.
- Feed por CNPJ (a página da empresa pode ganhar depois; a busca pelo
  CNPJ já cobre o caso).

## Fatos que orientam o desenho

- A busca ordena por relevância; um feed precisa da ordem cronológica.
- O `id` do ato muda a cada reindexação; se o `guid` do item fosse o `id`,
  o leitor mostraria atos antigos como novos depois de cada reindexação.
- A API conhece a URL pública do site (`PUBLIC_WEB_URL`).

## Decisões

1. **Ordem por data** (`ActFilter.Recent`): edição mais recente primeiro,
   posição na edição em seguida. Só o feed usa; a busca continua por
   relevância.
2. **`guid` estável**: `diario-sg:<gazette_id>:<position>`, com
   `isPermaLink="false"`. Muda só se a segmentação daquela edição mudar.
3. **Link do item é a cópia arquivada na página do ato**; o link do canal
   é a própria busca no site (URL permanente do PR da busca).
4. **50 itens**, `Cache-Control: public, max-age=900`: leitores de RSS
   consultam a cada poucos minutos; a busca não precisa rodar a cada vez.
5. **Mesmos filtros e validação da busca** (`filterFromQuery`), sem
   paginação.

## Desenho

- `domain.FeedLimit = 50`; `ActFilter.Recent bool`.
- Postgres: `orderSQL(f)` devolve a ordem por relevância (atual) ou
  `ORDER BY g.published_at DESC, g.source_url DESC, a.position`.
- `usecase.ActFeed.Execute(ctx, f) ([]domain.ActHit, error)`: normaliza,
  liga `Recent`, `Limit = FeedLimit`, `Offset = 0`, chama `Search`.
- `GET /v1/feeds/acts?<filtros>` → RSS 2.0 (`encoding/xml`),
  `application/rss+xml; charset=utf-8`:
  - canal: título `Diário SG: <termo>` (ou `Diário SG: atos publicados`
    sem termo), link para `<site>/?q=…&tipo=…` com os filtros,
    descrição com os filtros em texto, `language pt-br`, `ttl 15`;
  - item: título `<Tipo>: <título do ato>`, link para a cópia arquivada
    com `#page=N`, `guid`, `pubDate` (meio-dia de Brasília do dia da
    edição, RFC 1123), `category` com a sigla do órgão, `description`
    com o trecho sem os marcadores `⟦ ⟧`.
- Front: no bloco do alerta, "Prefere RSS? Assine o feed desta busca"
  com o link; `feedUrl(state)` em `searchState.ts`.

## Testes

- `http` (unitário): `siteSearchURL` monta a URL do site em português com
  valores no formato brasileiro; `stripMarks`; item RSS de um ato com e
  sem página e sem órgão.
- `usecase` (unitário): `ActFeed` liga `Recent`, usa `FeedLimit` e ignora
  `offset`.
- Integração: feed da edição de três contratos é XML válido, com 3
  itens na ordem de posição, `guid` estável, link arquivado; filtro de
  valor respeitado; filtro inválido é 400.
- Front (Vitest): `feedUrl` leva os filtros sem `limit`/`offset`.
