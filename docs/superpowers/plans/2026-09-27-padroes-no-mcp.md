# Padrões no MCP — plano

**Spec:** `docs/superpowers/specs/2026-09-27-padroes-no-mcp-design.md`

1. Domínio: `Finding.Entities` preenchido por cada construtor de achado;
   `FindingMentions(f, kind, key)` e testes por padrão.
2. MCP: ferramenta `padroes` (catálogo, por padrão com `pular`, por
   entidade), cache de 10 minutos, `ListPatterns` no `Deps`; teste de
   integração com dados que acionam um padrão de fornecedor.
3. MCP: prompts `investigar_fornecedor` e `seguir_contrato`; teste.
4. Página `/mcp` e README com a ferramenta nova; roadmap e plano de
   fontes (etapa G); "Depois da entrega".
