# Mural de licitações: plano

**Spec:** `docs/superpowers/specs/2026-09-27-mural-de-licitacoes-design.md`

1. Domínio `procurement.go`: `ProcessKey`, `ParseProcurements(lista,
   html)`, `ParseProcurementContracts(html)`; testes com linhas reais
   das quatro listas.
2. Adaptador `pmsgmural`: `List(ctx, nome)` com a raiz ISRG Root YR
   embutida além das do sistema, pausa de 1 s, User-Agent do projeto.
   Config `PMSG_MURAL_URL`.
3. Migration 028, `postgres.ProcurementRepo` (troca tudo; por CNPJ via
   `municipal_commitments`), caso de uso `LoadMural`, passo no
   `cmd/pncp`.
4. `/v1/entities/cnpj/{cnpj}` e MCP `entidade` com `procurements`, seção
   na página da empresa, teste de integração.
5. Carga real, README, roadmap e "Depois da entrega".
