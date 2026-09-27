# Ferramentas `pagamentos` e `contratacoes` do MCP (e a decisão sobre `empresa`)

## Problema

O plano de fontes lista três ferramentas que o MCP ainda não tem:
`pagamentos` (empenhos por credor, órgão e período, com totais),
`contratacoes` (licitações e contratos por fornecedor ou órgão) e
`empresa`. Hoje os empenhos e o mural só aparecem dentro de `entidade`,
e só por CNPJ: não dá para perguntar "quanto a Saúde pagou em 2025" nem
"que licitações de merenda saíram".

## Desenho

- **`pagamentos`** lê `municipal_commitments` (portal da Prefeitura,
  2017 em diante, todas as entidades). Filtros, todos opcionais e
  combináveis: `cnpj`, `entidade` (parte do nome, sem acento: "saude"),
  `processo` (mesma chave do mural), `texto` (no objeto e no nome do
  credor), `de` e `ate` (anos). Devolve os totais (empenhos, empenhado,
  liquidado, pago), `por_ano` (com o pago declarado no RREO quando não
  há outro filtro além dos anos, como total de controle), `por_entidade`
  e `por_credor` (os 10 maiores pelo pago) e `empenhos` (20, os mais
  recentes, com `pular`). Sem filtro nenhum, pede ao menos um.
- **`contratacoes`** lê o mural (`procurements`, `procurement_contracts`)
  e o PNCP. Filtros: `cnpj` (processos dos empenhos da empresa, como na
  página, mais os contratos do PNCP do fornecedor), `processo`, `texto`
  (no objeto e no fornecedor), `orgao` (o sufixo do edital, como PMSG,
  FMS, SEMED) e `de`/`ate` (ano da abertura). Devolve licitações (até
  20, com situação, abertura e link) e contratos (até 20, com valor,
  fornecedor e documento), com os totais encontrados, e a sugestão de
  `buscar_atos` para os extratos no Diário.
- **`empresa`**: não vira ferramenta. `entidade` com `tipo=cnpj` já traz
  cadastro, sanções (CGU e Diário), pagamentos (TCE e portal), mural,
  PNCP, obras e emendas; o prompt `investigar_fornecedor` encadeia. A
  descrição de `entidade` passa a dizer que ela é a ficha da empresa.

## Fora

- Pagamentos do TCE-RJ em `pagamentos`: o portal cobre mais anos e é
  mais completo (ver o total de controle); o TCE continua em `entidade`.

## Depois da entrega

- Na base local, `pagamentos` com só os anos 2023 a 2025 soma 19.760
  empenhos; o pago do portal fica a menos de 3,1% do RREO em cada ano
  (2025: R$ 2,536 bi contra R$ 2,540 bi; 2024: R$ 2,869 bi contra
  R$ 2,839 bi; 2023: R$ 2,682 bi contra R$ 2,766 bi). "saude" em 2025
  junta três entidades (fundo, fundação de saúde e a dos servidores),
  1.508 empenhos.
- O processo 17943/2024 tem 14 empenhos no portal e, no mural, uma
  licitação e dois contratos (R$ 13,84 mi).
- Os filtros respondem em menos de 100 ms, menos o de texto, que passa
  por `unaccent` no objeto e no nome de todos os empenhos (1,5 s na
  base local); um índice de trigramas resolveria se virar problema.
- A sigla SEMED não aparece nos editais do mural: os números usam PMSG
  (745), FMS (440), FAESG (42) e FMAS (6). A descrição da ferramenta cita
  essas.
