# SICONFI como total de controle

## Problema

Os pagamentos vêm dos empenhos que o município manda ao TCE-RJ, linha a
linha. Nada confere se a soma bate com o que o próprio município declara
ao Tesouro. O plano de fontes (etapa E) pede o SICONFI como "total de
controle": se os empenhos do TCE somam muito menos que o RREO, falta
dado, e a página tem de dizer.

## Fonte (conferida em 27/09/2026)

`GET https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rreo?an_exercicio=AAAA&nr_periodo=P&co_tipo_demonstrativo=RREO&id_ente=3304904&no_anexo=RREO-Anexo%2001`,
JSON sem chave, 1 pedido por segundo. O Anexo 01 (balanço orçamentário)
traz, na conta `TotalDespesas`, as colunas "DESPESAS EMPENHADAS ATÉ O
BIMESTRE", "DESPESAS LIQUIDADAS ATÉ O BIMESTRE" e "DESPESAS PAGAS ATÉ O
BIMESTRE", iguais de 2017 a 2025. O 6º bimestre fecha o ano; o ano
corrente só tem os bimestres já publicados (em 27/09/2026, nenhum de
2026 acima do 3º).

## Desenho

- **Domínio.** `FiscalTotal{Year, Period, CommittedCents, LiquidatedCents,
  PaidCents}` lido do Anexo 01; ano sem as três colunas é erro.
- **Coleta.** Passo novo do job `tce` (o total só serve para conferir os
  empenhos do TCE): para cada ano do período, o bimestre mais alto
  publicado (6 a 1). Bruto em `raw/siconfi/AAAA/MM/DD/rreo-AAAA-P.json`
  com SHA-256; `fetch_runs` com a fonte `siconfi`.
- **Tabela** `fiscal_totals` (ano, bimestre, empenhado, liquidado, pago,
  URL da consulta): troca o ano inteiro a cada carga.
- **Leitura.** `GET /v1/tce` ganha `fiscal_control`: por ano, o RREO
  (bimestre, empenhado, pago) ao lado da soma dos empenhos do TCE e da
  razão pago TCE / pago RREO. Página `/tce`: tabela "Conferência com o
  RREO" e aviso quando a razão fica abaixo de 90%.

## Fora

- RGF, DCA e MSC; outras contas do RREO.

## Depois da entrega

- O bruto fica em `raw/siconfi/AAAA/MM/DD/rreo_AAAA_P.json.gz`, com o
  manifesto (SHA-256) ao lado, como os outros conjuntos do TCE; o
  arquivamento em gzip com manifesto virou uma função só
  (`archiveJSON`), usada também pelo pessoal e pelo controle do TCE.
- A carga pede sempre de 2017 ao ano corrente (cerca de 15 pedidos, 13
  segundos): o bimestre do ano corrente muda, e os anos fechados podem
  ser retificados. `fiscal_totals` troca a linha do ano.
- Ano sem empenho do TCE carregado (2017 a 2019 na base local) sai com
  `tce_loaded: false` e "sem empenhos carregados", não como incompleto.
- A seção na página se chama "Total de controle" e usa valores
  compactos (R$ 2,5 bi); a cobertura dá a precisão.
- Na carga real, os RREO de 2017 e 2019 declaram empenhado, liquidado e
  pago iguais; é o que a Prefeitura mandou ao Tesouro, não erro de
  leitura (conferido na consulta). Os empenhos de 2020 no TCE-RJ vêm com
  liquidado e pago zerados na fonte, e o controle mostra 0%.
