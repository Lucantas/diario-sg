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
