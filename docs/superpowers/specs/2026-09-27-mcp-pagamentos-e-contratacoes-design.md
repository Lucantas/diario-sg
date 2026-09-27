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
