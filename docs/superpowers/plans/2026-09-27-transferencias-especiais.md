# Transferências especiais: plano

**Spec:** `docs/superpowers/specs/2026-09-27-transferencias-especiais-design.md`

1. Domínio `special_transfer.go`: linhas de cada tabela, datas e valores
   em centavos, `BuildSpecialTransfers`; testes com as linhas reais do
   plano 14371 (pago em duas ordens, relatório final) e de um plano
   impedido sem pagamento.
2. Adaptador `transferegov`: `Rows(ctx, tabela, filtro)` com
   `limit=1000`, pausa de 1 s, erro em status diferente de 200 e em
   1.000 linhas; `URL` para a fonte. Config `TRANSFEREGOV_URL`.
3. Portas, migration 026, `postgres.SpecialTransferRepo`
   (`ReplaceSpecialTransfers`, `SpecialTransfers`), caso de uso
   `LoadSpecialTransfers` (arquiva cada tabela, registra `fetch_runs`),
   passo no `cmd/federal`.
4. `/v1/federal` com `special_transfers`, seção na página `/federal`,
   teste de integração.
5. Carga real, conferência com a CGU, README, roadmap e "Depois da
   entrega".
