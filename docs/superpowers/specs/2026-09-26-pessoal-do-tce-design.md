# Pessoal do TCE-RJ (Entrega 4, parte 4)

Data: 2026-09-26. Último item da Entrega 4: "folha de pagamento só em
agregados por cargo/órgão". Traz o número de servidores e a remuneração
por mês, unidade e situação funcional que o município informa ao TCE-RJ,
e põe ao lado as nomeações e exonerações publicadas no Diário.

## Fatos que orientam o desenho

- `https://dados.tcerj.tc.br/api/v1/situacao_funcional?ano=AAAA&municipio=SAO%20GONCALO&inicio=0&limite=100000`
  devolve JSON já agregado: `Anomes` (`AAAA/MM`), `UnidadeGestora`,
  `SituacaoFuncional`, `Grupo`, `Quantidade` e `Remuneracao` (reais, ponto
  decimal). Nenhum dado de pessoa.
- Só há dados de janeiro de 2024 em diante (2021 a 2023 vêm vazios); em
  26/09/2026, até julho de 2026. São 6 unidades (Prefeitura, Câmara,
  IPASG, as duas fundações de saúde e a de artes e esporte) e cerca de
  200 linhas por ano; sem repetição de mês, unidade e situação.
- A classificação muda com o que o município informa: de abril de 2024 a
  abril de 2025, os cerca de 3 mil comissionados da Prefeitura aparecem
  como "Outras funções públicas remuneradas" (grupo "Outros") e os agentes
  políticos somem; em maio de 2025 voltam a "Comissionado Extraquadro". O
  grupo "Efetivo (com cargo ou função)" vem às vezes com espaço no fim.
- Em janeiro de 2026: 15.623 vínculos e R$ 72,4 milhões no mês, contando
  inativos e pensionistas.

## Decisões

1. **Carga no job `tce`**, depois dos empenhos: `cmd/tce` roda os dois
   casos de uso, cada um com a sua fonte em `fetch_runs` (`tce_pessoal`
   para este). Os anos são os mesmos do job, limitados a 2024 em diante.
2. **Tabela `tce_staff`** (migration 018): mês, unidade, situação, grupo
   (sem espaço nas pontas), quantidade e remuneração em centavos. A carga
   troca, numa transação, todos os meses dos anos lidos. Ano sem nenhuma
   linha não apaga o que já existe (a API às vezes responde vazio).
3. **Arquivo bruto:** a resposta de cada ano em
   `raw/tce_pessoal/AAAA/MM/DD/<ano>.json.gz` e um `manifest.json` com o
   SHA-256.
4. **Página `/pessoal`** ("Pessoal"): tabela por mês com vínculos e
   remuneração por grupo (efetivos, comissionados, contratados, agentes
   políticos, outros, inativos e pensionistas) e o total; ao lado, as
   nomeações e exonerações publicadas no Diário da Prefeitura no mês.
   Filtro por unidade. Aviso da mudança de classificação e de que a
   remuneração é a informada ao TCE. Link na página inicial.
5. **API:** `GET /v1/panels/staff?unit=` devolve os meses, as unidades e,
   por mês, os grupos, o total e as contagens do Diário. Sem ferramenta no
   MCP nesta parte.

## Fora do escopo

- Folha nominal (por pessoa), por cargo ou por lotação: o TCE não publica
  e a LGPD desaconselha (ADR 0006).
- Padrão novo a partir da folha: a classificação instável daria falsos
  alarmes; fica para quando houver mais meses estáveis.

## Testes

- Domínio: leitura de uma linha real, grupo com espaço, remuneração em
  centavos; montagem do painel por grupo e total.
- Adapter contra `httptest`: JSON, resposta vazia.
- Caso de uso com fakes: anos limitados a 2024, ano vazio não troca,
  bruto e manifesto.
- Integração: duas cargas, `GET /v1/panels/staff` com e sem unidade.
- Web: rótulo do mês e soma dos grupos.
- Carga real local e conferência de 3 meses contra o JSON do TCE.

**Pronto quando:** `/pessoal` mostra, mês a mês desde 2024, os vínculos e
a remuneração por grupo com as nomeações e exonerações do Diário, e o job
`tce` grava os dois conjuntos.
