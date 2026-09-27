# Empenhos do portal da Prefeitura

## Problema

Os pagamentos vêm do TCE-RJ, que cobre de 2020 em diante, não traz o
processo e, pelo total de controle do SICONFI, ficou com 89% do pago em
2024 e 87% em 2025 (2020 vem com o pago zerado). O plano de fontes pedia
o portal antigo (portaltp) para 2017–2022 e a descoberta da API do
portal novo para 2023 em diante.

## O que a checagem mostrou (27/09/2026)

- **portaltp** (`saogoncalo-rj.portaltp.com.br/api/transparencia.asmx`):
  o WAF responde 403 ("Request forbidden by administrative rules") ao
  User-Agent do projeto (`diario-sg-bot/…`) e ao do curl, e 200 a um
  User-Agent genérico. Trocar o User-Agent para passar seria contornar a
  regra; não fazemos.
- **Portal novo** (Embras/SIAPEGOV): a aplicação Angular lê
  `https://sistema.pmsg.rj.gov.br/portal-transparencia/api/`, sem chave.
  `sis_entidade` lista 29 entidades (Prefeitura, Câmara, fundações,
  fundos, SG-PREVI). `execucao/empenhos/empenhos?ano=AAAA&id_entidade=N`
  (com `dt_inicio_mes`, `dt_final_mes`, `nome_razao` e `nr_empenho`
  vazios) devolve `{cliente, empenhos, totais}`, o ano inteiro, sem
  paginação: 6.196 empenhos da Prefeitura em 2023. Cada empenho traz
  favorecido (nome e CPF/CNPJ formatado), objeto, tipo e número do
  processo, modalidade, número e data do empenho, e empenhado, liquidado
  e pago acumulados (texto "R$ 1.234,56"). Cobre de 2017 a 2026; todo
  empenho tem processo. O pago de 2025 somado nas 13 entidades com
  empenho dá cerca de R$ 2,57 bi, contra R$ 2,54 bi do RREO e R$ 2,21 bi
  do TCE-RJ.

## Desenho

- **Domínio.** `MunicipalCommitment` (entidade, ano, id e número do
  empenho, data, CNPJ, nome, objeto, tipo e número/ano do processo,
  modalidade, empenhado, liquidado, pago) e `MunicipalTotal` (ano,
  entidade, os três totais de `totais`, que incluem os favorecidos pessoa
  física). `ParseMunicipalCommitments` descarta quem não tem CNPJ válido
  (CPF não entra, ADR 0006).
- **Coleta.** Passo novo do job `tce` (o mesmo período, `-from` e
  `-to`): lista as entidades e, para cada ano e entidade, pede os
  empenhos. 1 pedido por segundo, com o User-Agent do projeto. Bruto em
  `raw/pmsg_empenhos/AAAA/MM/DD/<ano>-<entidade>.json.gz` com manifesto;
  `fetch_runs` com a fonte `pmsg_empenhos`. Troca o ano inteiro.
- **Tabelas** (migration 027): `municipal_commitments` e
  `municipal_totals`.
- **Leitura.**
  - `/v1/entities/cnpj/{cnpj}` ganha `municipal_commitments`: pago e
    empenhado por ano e os 50 empenhos mais recentes, com processo,
    modalidade e objeto. A página da empresa mostra a seção "Empenhos no
    portal da Prefeitura".
  - O total de controle em `/v1/tce` ganha a soma do portal por ano
    (`portal_paid_cents`), ao lado do TCE-RJ.

## Fora

- Liquidações e pagamentos por empenho (`execucao/empenhos/liquidacoes`
  e `pagamentos`, um pedido por empenho): o acumulado do empenho basta.
- Ligar o processo do empenho ao processo do Diário: o portal dá número
  e ano sem o prefixo do órgão nem o dígito; a ligação fica para a
  ferramenta `pagamentos` do MCP.

## Depois da entrega

- Carga real de 2017 a 2026: 42.641 empenhos com CNPJ em 5 minutos e meio.
  O pago do portal (somando os totais das entidades) fica perto do RREO em
  todos os anos: 2024, portal R$ 2,88 bi, RREO R$ 2,84 bi, TCE-RJ R$ 2,54
  bi; 2025, R$ 2,57 bi, R$ 2,54 bi e R$ 2,21 bi.
- O SG-PREVI repete em 2017 o mesmo empenho em duas linhas iguais, só com
  o objeto diferente (uma é a inscrição em restos a pagar). A carga fica
  com a primeira e conta as outras em `skipped` (9 na carga real).
- Os empenhos do TCE-RJ passaram a pular os anos antes de 2020, como o
  pessoal já fazia com os anteriores a 2024, para `make tce FROM=2017`
  carregar o portal inteiro sem falhar no passo do TCE.
- A página da empresa mostra os 10 empenhos mais recentes; a API devolve
  50.
- Objetos de sentença judicial (RPV) citam o nome do autor da ação. É o
  texto que a Prefeitura publica, como nos atos do Diário; CPF não entra.
- `empresas_penalizadas` do portal volta vazio para todas as entidades
  (conferido em 27/09/2026): a Prefeitura não publica ali as punições.
