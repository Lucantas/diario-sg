# Decisões que o código não explica sozinho

O porquê de escolhas que alguém poderia "consertar" sem saber o motivo. O
formato do Diário e as regras do parser estão em `parser-findings.md`; as
decisões de arquitetura, em `adr/`.

## Busca

- Um ato casa pela busca textual (português, sem acento, com stemming) **ou**
  pelo termo literal no corpo (`ILIKE` sem acento). O literal cobre CNPJ,
  número de contrato ou processo e nomes longos que o tokenizador quebra.
- O literal só vale para termos com dígito ou com 8 caracteres ou mais.
  "sus" ou "lei" como substring casariam com quase tudo e inundariam os
  alertas. O teste de integração `e2e_test.go` cobre isso.
- Ordem: atos com a frase exata (sem acento, sem caixa) primeiro, do mais
  recente ao mais antigo; o resto pelo `ts_rank`, que sozinho favorece
  listas longas de nomes com as palavras soltas.
- Aspas pedem frase exata. O `websearch_to_tsquery` entende as aspas, a
  comparação literal não, por isso elas saem antes do `ILIKE`.
- A consulta tem duas etapas. `matched` (CTE `MATERIALIZED`) guarda só id,
  frase exata, `ts_rank` e chaves de ordem de todos os atos encontrados;
  `page` ordena, conta e corta a página; só então entram corpo, destaque e
  entidades. Carregar o corpo de milhares de atos até o `count(*) OVER ()`
  fazia o sort ir para disco e recalcular a frase exata sobre textos
  enormes. O `MATERIALIZED` impede o Postgres de reabrir a CTE e refazer
  essas contas. Sem termo, a frase exata nem é calculada.
- O destaque vem entre `⟦ ⟧` e não em HTML, para o front renderizar sem
  `innerHTML` (sem brecha de XSS). Quando o ato casou só pelo literal, o
  `ts_headline` não marca nada e `highlightFallback` marca o termo, com a
  mesma comparação sem acento e sem caixa do banco.

- `OU` (maiúsculo, fora de aspas) vira `or` antes do
  `websearch_to_tsquery`. Em português `ou` é stopword e sumia:
  `limpeza OU coleta` virava `limpeza E coleta`. Minúsculo continua
  stopword porque "ou" aparece o tempo todo no texto dos atos. Vale na
  busca, nas estatísticas e no casamento de alertas.
- A faixa de valor casa um ato que cite **ao menos um** valor dentro dela.
  Um extrato cita valor global, mensal e unitário; exigir todos na faixa
  esconderia o contrato. Por isso a interface diz "cita valor".
- A API recebe reais com ponto decimal e sem milhar (`1500.50`); o front
  aceita o formato brasileiro e converte. `1.500` na API seria ambíguo, e
  a URL da busca precisa ser permanente.
- Os filtros (tipo, órgão, período, valor) saem de um só construtor,
  `filterSQL`, com parâmetros numerados; a busca, as estatísticas, a
  exportação e o RSS usam o mesmo.
- A URL do site usa nomes em português (`?q=&tipo=&orgao=&de=&ate=&valor_min=&valor_max=&pagina=`);
  a da API continua em inglês.

## Dados

- `NormalizeCNPJ` não valida dígito verificador: o Diário publica CNPJ com
  erro de digitação, e quem investiga precisa achá-lo mesmo assim.
- Valor em reais só é extraído com centavos: `VALOR (R$ 1)` em cabeçalho de
  tabela não é valor.
- `pdftotext` roda sem `-layout`: o Diário tem duas colunas e o `-layout`
  mistura as duas na mesma linha. Texto vazio indica PDF escaneado, o lugar
  para plugar OCR.

- O catálogo de órgãos (sigla → nome) fica no domínio (`domain/organ.go`)
  e é a allowlist do parser: API e parser usam a mesma lista. O nome por
  extenso é resolvido na apresentação, não gravado em `acts`, para a
  tabela mudar sem reindexar. Nome sem evidência no próprio Diário fica
  vazio (o levantamento está em `docs/orgaos.md`); nome errado é pior que
  sigla sozinha.
- O filtro por órgão aceita só siglas do catálogo (sigla desconhecida é
  `400`), como o filtro de tipo.

## Citação e cópia arquivada

- A API serve o PDF arquivado (`/v1/gazettes/{id}/pdf`) em vez de abrir o
  bucket: bucket público exporia a listagem e os marcadores `.published`,
  e URL assinada expira, o que não serve para citação.
- O PDF de uma edição não muda (o checksum é a identidade dela), então o
  `ETag` é o SHA-256 e o cache é imutável. O handler busca a edição e
  responde `304` antes de abrir o arquivo: depois de 30 dias os objetos
  vão para a classe ARCHIVE, onde cada leitura é cobrada.
- A citação aponta para a edição e a página, não para o ato: o `id` do
  ato muda a cada reindexação (`ReplaceActs` apaga e insere) e a posição
  pode mudar se a segmentação mudar.
- Página nula (ato ainda não reindexado) sai como `null`, nunca como 1: o
  link abre o PDF no início e a citação omite a página.
- A citação é montada no front (`src/citation.ts`); a API devolve os
  dados, e o formato pode mudar sem mexer no backend.

## Exportação

- Teto de 10.000 atos por exportação, na ordem da busca: cobre qualquer
  recorte razoável e limita a resposta a dezenas de MB. Acima disso os
  cabeçalhos `X-Total-Count` e `X-Export-Truncated` (e `truncated` no
  JSON) avisam; a base inteira fica para o dump.
- CSV feito para abrir direto no Excel e no LibreOffice em pt-BR:
  separador `;`, UTF-8 com BOM, CRLF (o `encoding/csv` também troca as
  quebras de linha dentro dos campos) e decimal com vírgula.
- Célula de texto que começa com `=`, `+`, `-`, `@`, tab ou CR ganha um
  apóstrofo na frente: o texto dos atos vem de fora e vai ser aberto em
  planilha (injeção de fórmula, OWASP).
- Streaming do banco para a resposta; os cabeçalhos são escritos quando
  chega a primeira linha, que traz o total. Resposta em streaming não tem
  o limite de 32 MiB do Cloud Run.
- Consulta própria, sem `ts_headline`: gerar trecho destacado para 10 mil
  atos custaria segundos e a exportação leva o corpo inteiro.

## RSS

- O feed ordena por data (edição mais recente primeiro), não por
  relevância: leitor de RSS espera novidade no topo.
- O `guid` do item é `diario-sg:<edição>:<posição>`, não o `id` do ato:
  o `id` muda a cada reindexação e o leitor mostraria atos antigos como
  novos. Só muda se a segmentação daquela edição mudar.
- 50 itens e `Cache-Control` de 15 minutos: leitores consultam a cada
  poucos minutos e não precisam rodar a busca toda vez.

## Dump da base

- CSV comum (vírgula, UTF-8 sem BOM, gzip) no dump, CSV de planilha
  pt-BR na exportação da busca: o dump é para programas (pandas,
  sqlite-utils), a exportação é para gente abrir no Excel.
- Sem SQLite nem Parquet prontos: o CSV compactado cobre os dois usos sem
  dependência nova no Go, e o LEIAME mostra os comandos.
- Bucket próprio e público para o dump; o de edições é privado e move
  tudo para ARCHIVE. Leitura anônima com `legacyObjectReader`, que lê
  objeto mas não lista o bucket.
- As três tabelas saem de uma transação `REPEATABLE READ` só de leitura,
  senão uma edição indexada no meio do dump apareceria numa tabela e não
  na outra.
- Cada tabela é compactada e enviada por `io.Pipe`, sem arquivo
  temporário: no Cloud Run o `/tmp` é memória.
- O manifesto sobe por último e só se todas as tabelas subiram. Quem lê
  o manifesto encontra os arquivos que ele descreve.
- O nginx serve `/dados/` por proxy do bucket, e `location = /dados`
  devolve a página do front: sem ele, o prefixo com `proxy_pass`
  redirecionaria `/dados` para `/dados/`.

## Qualidade visível

- Os avisos de extração (`sem_numero`, `so_titulo`, `muitas_paginas`)
  são calculados na leitura, em `domain.ActWarnings`, e não gravados:
  regra nova vale na hora, sem migration nem reindexação. A API devolve
  códigos e o front escreve o texto.
- `muitas_paginas` começa em 10 páginas: 339 atos na base de 22/09/2026,
  quase todos anexos ou atos que o parser não separou.
- A "tabela quebrada entre páginas" do relatório da fase 1 ficou sem
  aviso. A regra candidata (`Port. nº` cujo corpo não começa com o verbo)
  acha 1.209 atos com muitos falsos positivos, e um aviso errado tira o
  crédito dos certos.
- O reporte identifica o ato por edição, posição e cópia do título, não
  pelo `id`, que muda a cada reindexação. Apagar a edição apaga os
  reportes dela.
- Reporte sem e-mail: não há como responder sem pedir dado pessoal, e a
  correção aparece no próprio site.
- Contra abuso, sem cadastro: campo-isca (quem preenche recebe 202 e nada
  é gravado), mensagem de até 2.000 caracteres e janela fixa de 5 reportes
  por minuto por cliente e 60 por instância. O cliente é o primeiro IP do
  `X-Forwarded-For`, que dá para falsificar; o teto por instância segura
  esse caso. O pior resultado é lixo na fila, que `make close-report
  AS=descartado` limpa.

## Servidor MCP

- No mesmo binário e serviço da API: chama os mesmos casos de uso e não
  pede infra nova. SDK oficial (`modelcontextprotocol/go-sdk`), que exige
  Go 1.25.
- Sem sessão e com resposta JSON (`Stateless`, `JSONResponse`): qualquer
  instância do Cloud Run atende, e o `statusRecorder` do middleware não
  precisa de `Flush`.
- Chave anônima, sem e-mail: identifica o uso, não a pessoa. O banco
  guarda só o SHA-256 e um prefixo de 8 caracteres, que é o que aparece no
  log. Chave vazada se revoga com a própria chave.
- Limites: 3 chaves por hora por cliente e 50 por instância; 60 chamadas
  por minuto por chave e 600 por instância. O registro de uso guarda só a
  contagem por dia e ferramenta, nunca os argumentos.
- Lote JSON-RPC (um array no corpo) é recusado com 400. O SDK aceita lote
  quando o cliente não manda `Mcp-Protocol-Version`, e um único POST com
  centenas de chamadas passaria pelo limite, que conta requisições.
- Antes da chave, um limite por cliente (120 por minuto por IP, 1.200 por
  instância) segura quem manda chave inventada, que custaria uma consulta
  ao banco por requisição. A revogação tem o seu (10 por minuto).
- Erro de entrada volta como erro da ferramenta, com a mensagem do
  domínio, para a IA corrigir a chamada; erro interno vira "erro interno"
  e fica no log.
- A cobertura (período, edições, atos) vai em toda resposta e fica em
  cache por 5 minutos: a contagem de atos percorre um índice inteiro (~0,27 s na base local).
- Citação ABNT reescrita em Go com o mesmo teste do front: as duas precisam
  continuar iguais.

## Entrega de mensagens e idempotência

- Pub/Sub entrega pelo menos uma vez. A edição é identificada pelo
  checksum do PDF: se já foi indexada, o worker só republica o evento.
- O scraper grava o PDF, publica o evento e só então grava o marcador
  `.published`. Se a publicação falhar, a próxima coleta tenta de novo; o
  consumidor ignora a repetição pelo checksum.
- `notifications_sent` garante no máximo um alerta por inscrição e edição.
  Por isso o caso de uso pode devolver erro e deixar o Pub/Sub reentregar.
- Resposta ao push: 2xx confirma (inclusive mensagem inválida de forma
  permanente, que não adianta repetir); 5xx faz o Pub/Sub reentregar com
  backoff e, depois de N tentativas, mandar para a DLQ.
- O scraper sai com código 1 em falha para o Cloud Run Job aplicar
  `max_retries`.

## Alertas

- Inscrição com confirmação dupla e link de cancelamento em todo e-mail
  (LGPD e reputação de envio).
- Confirmar e cancelar são `POST` disparados por um botão na página, não
  `GET` no link do e-mail: robôs que pré-visualizam links não podem
  confirmar nem cancelar nada.
- Uma busca por inscrição a cada edição. Serve para milhares de inscrições;
  depois disso, busca reversa (ADR 0003).

## Infra e clientes

- Clientes do Google Cloud em REST com a biblioteca padrão, sem os SDKs:
  imagem menor e o mesmo código fala com os emuladores locais.
- Pool de conexões pequeno: o plano gratuito do Neon limita conexões e cada
  instância do Cloud Run tem o seu pool.
- O servidor HTTP faz shutdown gracioso ao receber SIGTERM (o Cloud Run dá
  10 s).
- `migrate` usa advisory lock para duas execuções simultâneas não
  conflitarem.

## Terraform, CI e ambiente local

- `infra/bootstrap` é aplicado uma vez, à mão, por alguém com papel Owner:
  cria o bucket do state, o Workload Identity Federation (o GitHub Actions
  autentica sem chave JSON, e só este repositório pode trocar token) e as
  contas do pipeline. Os outputs viram Variables dos Environments no GitHub.
- O Terraform cuida de configuração, escala e IAM dos serviços Cloud Run; a
  imagem é do workflow de deploy (`ignore_changes` na imagem). O deploy só
  troca a imagem.
- A URL do Cloud Run é montada de forma determinística para evitar
  dependência circular entre api e web.
- Worker com concorrência 4: extração de PDF é pesada. `cpu_idle` para só
  pagar CPU durante requisição.
- Cada fila tem retry com backoff, DLQ e uma assinatura pull na DLQ para
  inspecionar e reprocessar; a DLQ guarda as mensagens por 7 dias.
- Artifact Registry apaga imagens com mais de 14 dias para caber no free
  tier (0,5 GB).
- Uma conta de serviço por componente, com o mínimo de permissão; a conta
  do deploy roda as migrations e age como as contas dos serviços.
- O workflow de infra nunca cancela um `apply` em andamento
  (`cancel-in-progress: false`).
- A reindexação na nuvem é um Cloud Run Job com a imagem da API e a conta
  do worker (que já lê o bucket e o segredo do banco), executado à mão
  pelo workflow Reindex. Não é passo do deploy: leva horas e é decisão de
  quem mudou o parser. Sem retry: a falha é registrada por edição e o
  período pode ser executado de novo, já que a reindexação é idempotente.
- Os Dockerfiles são construídos a partir da raiz do repositório
  (`docker build -f services/api/Dockerfile .`): os `COPY` usam caminhos
  a partir dela, e os serviços Go copiam `pkg/`.
- O nginx faz proxy de `/api` para a API na mesma origem (sem CORS); o
  template passa pelo `envsubst` da imagem oficial, que só troca variáveis
  definidas no ambiente.
- No `docker-compose.yml`, `extra_hosts: host.docker.internal` deixa o
  emulador do Pub/Sub entregar push ao worker que roda fora do Docker.
- `make test-integration` usa o banco `diario_test`, criado pelo próprio
  teste, para não misturar com as edições ingeridas localmente.

## Scraper

- A listagem usa a busca do site com o termo "a", que aparece em qualquer
  edição. "Gonçalo" não serve: a continuação de 13/12/2023 não tem
  cabeçalho.
- Uma conexão nova por requisição: em backfill longo o site deixa conexões
  ociosas mortas sem fechar, e reaproveitá-las travava o download seguinte.
- Teto de 200 páginas de listagem por execução (5 edições por página, cerca
  de 1.000 edições). Backfill longo roda por ano.
