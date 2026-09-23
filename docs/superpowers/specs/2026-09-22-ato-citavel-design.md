# Ato citável — página, PDF arquivado e citação (Entrega 1b)

Data: 2026-09-22. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"). Usa a página e o órgão gravados na 1a e a reindexação em
produção do PR anterior.

## Objetivo

Quem acha um ato consegue:

- abrir o PDF **na página do ato**, tanto no site da prefeitura quanto na
  **cópia arquivada** servida por nós (a prefeitura pode tirar o arquivo do
  ar);
- conferir que a cópia é a mesma coletada, pelo **SHA-256 do PDF**;
- copiar uma **citação** pronta, com edição, data, página, título, os dois
  links e o hash.

## Fora do escopo

- Link permanente para o ato no nosso site. O `id` do ato muda a cada
  reindexação (`ReplaceActs` apaga e insere), e a posição também pode
  mudar se o parser mudar a segmentação. O que é estável é a edição (`id`
  da edição e checksum) e a página; a citação aponta para eles.
- Página de edição no front (a API já tem `GET /v1/gazettes/{id}`).
- Exportação (PR próprio).

## Fatos que orientam o desenho

- O scraper grava o PDF no bucket (`storage_path`) e o SHA-256 dele em
  `gazettes.checksum` (hex, 64 caracteres); o worker já lê do bucket.
- O bucket é privado (`public_access_prevention = "enforced"`) e move os
  objetos para a classe ARCHIVE depois de 30 dias. Leitura em ARCHIVE é
  imediata, mas cobra por GB lido.
- Atos indexados antes da migration 005 e ainda não reindexados têm página
  nula.
- Visualizadores de PDF de navegador (Chrome, Firefox, Edge) respeitam
  `#page=N`.

## Decisões

1. **A API serve o PDF**, em `GET /v1/gazettes/{id}/pdf`, lendo do bucket.
   Alternativas descartadas: bucket público (expõe a listagem e os
   marcadores `.published`, e a URL muda se o bucket mudar) e URL assinada
   (expira, e uma citação não pode expirar).
2. **Cache agressivo**: o conteúdo é imutável para a mesma edição (o
   checksum é o `ETag`), então `Cache-Control: public, max-age=31536000,
   immutable` e resposta `304` para `If-None-Match`. Diminui leituras
   cobradas na classe ARCHIVE.
3. **Página nula não vira página 1**: sem página, o link abre o PDF no
   início e a citação omite a página.
4. **Citação montada no front**, a partir dos campos da busca, em formato
   próximo da ABNT para documento eletrônico. A API devolve os dados, não o
   texto; o formato pode mudar sem mexer no backend.

## Desenho

### Domínio e portas

- `ActHit` ganha `Checksum string` (da edição). `Act.PageStart/PageEnd`
  continuam `int`, com 0 significando "sem página".
- Caso de uso `GetGazettePDF` com dois passos: `Gazette(ctx, id)`
  (`FindByID`) e `Open(ctx, g)` (`FileStorage.Get(storage_path)`). Dois
  passos para o handler responder `304` sem ler o bucket. Edição
  inexistente é `ErrNotFound`; falha no bucket é erro interno.

### Dados

Sem migration. `Search`, `ReportByEntity` e `ListByGazette` passam a ler
`page_start`, `page_end` (nulos viram 0) e `g.checksum`.

### API

- Itens de `/v1/acts` e `/v1/entities/cnpj/{cnpj}` ganham `page_start` e
  `page_end` (`null` quando não há) e `pdf_sha256`.
- `/v1/gazettes/{id}` ganha `pdf_sha256`; cada ato ganha `page_start`,
  `page_end`, `organ` e `organ_name`.
- `GET /v1/gazettes/{id}/pdf`: `Content-Type: application/pdf`,
  `Content-Disposition: inline; filename="diario-sg-AAAA-MM-DD-<edição>.pdf"`
  (`-extra` quando for extra; `s-n` quando sem número), `ETag` com o
  SHA-256, `Cache-Control` imutável, `304` com `If-None-Match` igual.
- Config: a API passa a exigir `GAZETTE_BUCKET` (e usa
  `STORAGE_EMULATOR_HOST` no ambiente local).

### Infra

- A conta da API ganha `roles/storage.objectViewer` no bucket de edições.
- O serviço da API ganha `GAZETTE_BUCKET`.

### Front

- Linha de metadados do resultado: o link da edição continua indo ao PDF
  da prefeitura, agora com `#page=N`; ao lado, "p. 3" (ou "p. 3–4") e o
  link "cópia arquivada" (`/api/v1/gazettes/{id}/pdf#page=N`).
- Botão "Citar" abre um bloco com o texto da citação e um botão "Copiar"
  (`navigator.clipboard`; sem ele, o texto fica selecionado para
  Ctrl+C).
- `src/citation.ts` com funções puras `pageFragment`, `pageLabel` e
  `formatCitation`, testadas com Vitest (novo no projeto, com `npm test`
  no CI).

Formato da citação:

```
SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, ed. 1771 (extra), 18 set. 2026, p. 3-4. PORTARIA Nº 10/2026. Disponível em: <https://do.pmsg.rj.gov.br/…pdf#page=3>. Cópia arquivada em: <https://…/api/v1/gazettes/…/pdf#page=3>. SHA-256 do PDF: 9f2c…. Acesso em: 22 set. 2026.
```

Edição sem número vira "ed. s/n"; sem página, o trecho "p. …" sai e os
links vão sem `#page`.

## Testes

- Unitário (`usecase`): `GetGazettePDF` devolve o conteúdo do storage;
  edição inexistente dá `ErrNotFound`.
- Unitário (`http`): nome do arquivo (`pdfFilename`) para edição normal,
  extra e sem número.
- Integração: `/v1/acts` traz `page_start`, `page_end` e `pdf_sha256`;
  ato com página nula vem com `null`; `/v1/gazettes/{id}/pdf` devolve os
  bytes, `ETag`, `Content-Disposition` e `304` com `If-None-Match`;
  edição inexistente dá 404.
- Front (Vitest): `formatCitation` com e sem página, com edição extra e
  sem número; `pageLabel` para uma e várias páginas.
- Manual na base local: abrir o link de três atos e conferir que o PDF
  abre na página certa.

## Riscos

- **Custo de leitura em ARCHIVE**: um robô baixando todos os PDFs (cerca
  de 650 MB) custa centavos por passada, mas repetido pode pesar. O cache
  imutável ajuda; se virar problema, trocar o lifecycle para NEARLINE ou
  COLDLINE.
- **Atos sem página em produção** até a reindexação rodar lá: o front
  funciona, só sem página.
