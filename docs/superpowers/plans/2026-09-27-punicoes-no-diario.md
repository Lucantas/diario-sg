# Punições no Diário: plano

**Spec:** `docs/superpowers/specs/2026-09-27-punicoes-no-diario-design.md`

1. Domínio `diario_sanction.go`: `ClassifyDiarioSanction`, com testes
   dos títulos e fórmulas reais e do ruído (competência de comissão,
   regra geral, anistia).
2. Repositório: `SanctionCandidatesByCNPJ` (id, título e corpo dos atos
   ligados ao CNPJ que passam no filtro largo) e `HitsByIDs`; em
   `CompanySources`, classificar e buscar os atos.
3. `diario_sanctions` em `/v1/entities/cnpj/{cnpj}`, seção na página,
   `punicoes_diario` no MCP; teste de integração.
4. Conferência na base local, README, roadmap e "Depois da entrega".
