# `pagamentos` e `contratacoes`: plano

**Spec:** `docs/superpowers/specs/2026-09-27-mcp-pagamentos-e-contratacoes-design.md`

1. Domínio: `PaymentQuery` e `ProcurementQuery` com validação (ao menos
   um filtro, anos em ordem, `pular` não negativo) e os resultados
   (`PaymentReport`, `ProcurementReport`); testes.
2. Postgres: `MunicipalCommitmentRepo.QueryPayments` (totais, por ano,
   por entidade, por credor, página de empenhos, com filtro montado por
   parâmetros) e `ProcurementRepo.QueryProcurements`; RREO por ano do
   `FiscalRepo`.
3. Casos de uso `QueryPayments` e `QueryProcurements`; ferramentas
   `pagamentos` e `contratacoes` no MCP; descrição de `entidade` como
   ficha da empresa; lista de ferramentas no site e no teste.
4. Testes de integração, README, roadmap e "Depois da entrega".
