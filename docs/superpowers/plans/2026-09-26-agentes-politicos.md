# Agentes políticos e subsídios — plano

**Spec:** `docs/superpowers/specs/2026-09-26-agentes-politicos-design.md`

1. Domínio `political_agents.go`: papéis, `ParsePrefeituraPay`,
   `ParseCamaraPay`, `ParseCouncillors`, `BuildPoliticalAgents`,
   `SubsidyNorms`. Testes com linhas reais recortadas.
2. Adaptador `agentes` (HTTP): folha da Prefeitura por mês, folha da
   Câmara por ano (GET do formulário e POST de exportação), SICAM por ano.
   Testes com `httptest`.
3. Caso de uso `LoadPoliticalAgents` (período, arquivo bruto, fetch run) e
   `GetPoliticalAgents`; portas; repositório Postgres e migration 023.
   Teste de integração.
4. `cmd/agentes`, config, Makefile, Dockerfile, Terraform (job mensal),
   deploy.
5. API `/v1/agentes`, página `/agentes`, ferramenta MCP.
6. Carga real desde 10/2010, conferência de meses contra as fontes, docs
   (roadmap, README, fontes) e "Depois da entrega".
