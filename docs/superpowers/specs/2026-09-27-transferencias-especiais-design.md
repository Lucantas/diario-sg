# Transferências especiais ("emendas Pix")

## Problema

A emenda individual na modalidade transferência especial cai direto no
caixa do município, sem convênio. O Portal da Transparência só mostra o
valor, somado por mês ("TRANSFERENCIAS ESPECIAIS" em
`federal_transfers`: R$ 5,57 mi, sem autor). O arquivo de emendas da CGU
também não traz essas emendas como destinadas a São Gonçalo. Faltam o
parlamentar, o objeto, a execução e o relatório de gestão.

## Fonte (conferida em 27/09/2026)

API pública do Transferegov (PostgREST, sem chave):
`https://api.transferegov.gestao.gov.br/transferenciasespeciais/`.

- `plano_acao_especial?cnpj_beneficiario_plano_acao=eq.28636579000100`:
  8 planos de ação (2022 a 2024): código, ano, situação (`CIENTE`,
  `IMPEDIDO`…), parlamentar, número da emenda, área, custeio e
  investimento. Traz a conta bancária do município, que não guardamos.
- `executor_especial`, `plano_trabalho_especial` e
  `relatorio_gestao_novo_especial`, por `id_plano_acao=in.(…)`: o
  executor (fundo ou município), o objeto, o prazo e o relatório de
  gestão (executado e pendente).
- `empenho_especial` (por plano) → `documento_habil_especial` (por
  empenho) → `ordem_pagamento_ordem_bancaria_especial` (por documento):
  o pago é a soma dos documentos com ordem bancária, e a data é a da
  última ordem. Na carga de conferência, o pago soma R$ 5,57 mi, igual ao
  total da CGU.

PostgREST devolve no máximo 1.000 linhas; cada pedido leva `limit=1000`,
e 1.000 linhas de volta é erro (resposta cortada). 1 pedido por segundo.

## Desenho

- **Domínio.** `SpecialTransfer` (plano, código, ano, situação, autor,
  emenda, área, valor, executores, empenhado, pago, última ordem
  bancária, situação e fim do plano de trabalho, relatório mais recente
  com tipo, data, executado e pendente) montado por
  `BuildSpecialTransfers` a partir das linhas de cada tabela.
- **Coleta.** Passo novo do job `federal` (`LoadSpecialTransfers`): os
  planos, depois as tabelas ligadas por `in.(…)`. Bruto de cada tabela
  em `raw/transferegov_especiais/AAAA/MM/DD/<tabela>.json.gz` com
  manifesto; `fetch_runs` com a fonte `transferegov_especiais`. Troca
  tudo a cada carga (são poucas linhas).
- **Tabelas** (migration 026): `special_transfers` e
  `special_transfer_executors`.
- **Leitura.** `GET /v1/federal` ganha `special_transfers`. A página
  `/federal` ganha a seção "Transferências especiais (emendas Pix)": por
  plano, o ano, o autor, o objeto, a situação, o valor, o pago e o que o
  relatório de gestão diz que foi executado.

## Fora

- Metas (`meta_especial`) e histórico da ordem de pagamento: o objeto e
  o relatório bastam para conferir.
- Ligação do executor com a página de empresa: o executor é o município
  ou um fundo municipal, não fornecedor.
