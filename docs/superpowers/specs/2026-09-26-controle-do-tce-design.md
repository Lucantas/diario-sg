# Controle do TCE-RJ (Entrega 5, parte 1)

Data: 2026-09-26. Primeiro item da Entrega 5 ("TCE-RJ: apontamentos sobre
contratos já presentes no Diário"). Traz o que o TCE-RJ publica sobre o
município em dados abertos: o parecer prévio das contas de governo, os
débitos e multas aplicados às unidades de São Gonçalo e as obras
paralisadas, e mostra tudo numa página `/tce`.

## Fatos que orientam o desenho

Consultados em 26/09/2026 na API `https://dados.tcerj.tc.br/api/v1/`
(OpenAPI em `/api/v1/openapi.json`):

- `prestacao_contas_municipio`: 637 linhas, todos os municípios;
  São Gonçalo tem 7 (2019 a 2025) com ano, parecer ("FAVORÁVEL", "EM
  ANALISE"…), processo do TCE e responsável (o prefeito, agente
  político).
- `penalidades_ressarcimento_municipio`: 5.243 linhas, todos os
  municípios (o filtro `municipio` é ignorado); São Gonçalo tem 132
  condenações em 50 processos, de 2022 a 2026, somando R$ 10,4 milhões.
  Cada linha traz processo, condenação, ano, valor, órgão, natureza do
  processo e data da sessão. Não traz o nome de quem foi condenado.
- `obras_paralisadas`: 431 obras; São Gonçalo tem 5 (paralisadas de 2016
  a 2019), com número do contrato, CNPJ e nome da contratada, valor do
  contrato, valor pago, motivo e situação do contrato.
- Cerca de 10 dos 50 processos com condenação são citados no Diário (por
  exemplo, portarias de tomada de contas especial que citam "Processo
  TCE-RJ n° 214.824-1/2014"), sempre com o número escrito com ponto.

## Decisões

1. **Carga no job `tce`**, depois do pessoal: um caso de uso lê os três
   conjuntos inteiros, fica com as linhas de São Gonçalo e troca as três
   tabelas numa transação. Fonte `tce_controle` em `fetch_runs`.
2. **Tabelas** (migration 019): `tce_accounts` (ano, parecer, processo,
   responsável), `tce_penalties` (condenação como chave, processo, ano,
   valor, órgão, natureza, sessão) e `tce_stalled_works` (contrato,
   CNPJ, contratada, órgão, função, valores, datas, motivo, situação).
3. **Arquivo bruto:** a resposta inteira de cada conjunto em
   `raw/tce_controle/AAAA/MM/DD/<conjunto>.json.gz`, com
   `<conjunto>.manifest.json` (SHA-256 e linhas de São Gonçalo).
4. **Ligações:** a obra paralisada liga à empresa pelo CNPJ (`exata`).
5. **Página `/tce`** ("TCE-RJ"): contas de governo por ano, obras
   paralisadas (com link para a página da empresa) e débitos e multas por
   processo (órgão, natureza, condenações, valor, sessão), cada processo
   com o link para buscar o número no Diário. Link na página inicial.
6. **Página da empresa:** "Obras paralisadas (TCE-RJ)" quando a empresa
   tem obra na lista. `GET /v1/entities/cnpj/{cnpj}` traz `stalled_works`;
   a ferramenta `entidade` traz `obras_paralisadas_tce`.
7. **API:** `GET /v1/tce` devolve os três conjuntos.

## Fora do escopo

- Nome dos condenados (o TCE não publica nos dados abertos).
- Ligar a condenação ao contrato do Diário (o processo do TCE não é o
  processo administrativo da Prefeitura).
- Contratos, licitações e compras diretas do TCE (`contratos_municipio`,
  `licitacoes`): ficam para uma parte própria.

## Testes

- Domínio: leitura de uma linha real de cada conjunto; filtro de São
  Gonçalo; rótulo de busca do processo (`214.824-1/2014` → `"214.824"`).
- Caso de uso com fakes: troca das três tabelas, bruto e manifesto, falha
  num conjunto não grava nada.
- Integração: carga, `GET /v1/tce`, empresa com obra paralisada.
- Web: agrupamento das condenações por processo.
- Carga real local e conferência contra o JSON.

## Depois da entrega

- Carga local de 26/09/2026 em 3,4 segundos, dentro do job `tce`: 7
  pareceres, 132 condenações em 50 processos (R$ 10.370.821,01) e 5 obras
  paralisadas, os mesmos números do JSON.
- 3 das 5 obras são de empresas citadas no Diário (R.C Vieira Engenharia,
  com duas, e Fox Engenharia) e ganharam ligação `exata`.
- A busca do processo no Diário pelo número com ponto trazia números
  iguais de outras coisas (autos de infração, editais); o link passou a
  buscar o número junto com o termo "TCE" (`"214.824" TCE`).
- A carga roda com os dados inteiros (não depende dos anos do job), e a
  lista de débitos da página é longa no celular: 50 processos, um por
  item.

**Pronto quando:** `/tce` mostra os pareceres, as obras paralisadas e os
débitos e multas de São Gonçalo, e a página da empresa com obra
paralisada mostra a obra.
