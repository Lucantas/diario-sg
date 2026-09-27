# Empenhos do portal da Prefeitura: plano

**Spec:** `docs/superpowers/specs/2026-09-27-empenhos-do-portal-da-prefeitura-design.md`

1. Domínio `municipal_commitment.go`: valor em texto para centavos,
   data, CNPJ válido, `ParseMunicipalEntities`,
   `ParseMunicipalCommitments` (empenhos e totais); testes com linhas
   reais (CNPJ, CPF descartado, valores com milhar).
2. Adaptador `pmsgportal`: `Entities` e `Commitments(ano, entidade)`,
   pausa de 1 s, User-Agent do projeto, limite de tamanho. Config
   `PMSG_PORTAL_URL`.
3. Migration 027, `postgres.MunicipalCommitmentRepo` (troca o ano com
   COPY; leitura por CNPJ; soma do pago por ano), caso de uso
   `LoadMunicipalCommitments`, passo no `cmd/tce`.
4. `/v1/entities/cnpj/{cnpj}` com `municipal_commitments`, seção na
   página da empresa; `portal_paid_cents` no total de controle e coluna
   em `/tce`; testes de integração.
5. Carga real de 2017 a 2026, conferência com o RREO, README, roadmap e
   "Depois da entrega".
