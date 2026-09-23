# Plano: todas as fontes públicas de São Gonçalo no Diário SG

Data: 23/09/2026. Complementa `docs/roadmap.md` (Entregas 2 a 5) e usa o
levantamento de `docs/fontes/README.md`.

## Objetivo

Uma pessoa investigando a Prefeitura ou a Câmara faz uma pergunta ("quanto
a empresa X recebeu, por quais contratos, e quem são os sócios?") e recebe a
resposta **com a prova**: o ato no Diário, a linha do empenho, o registro do
PNCP, o documento da Receita. Hoje isso exige abrir seis portais que não
conversam entre si, três deles sem exportação.

Duas portas para as mesmas respostas:

- **O site**, com páginas por entidade (empresa, processo, contrato, órgão,
  agente político) e perguntas prontas.
- **Um servidor MCP**, que cada pessoa pluga na IA que já usa (Claude,
  ChatGPT, Cursor…). Não temos chat próprio: a IA é do usuário, os dados e
  as provas são nossos.

## Princípios

1. **Proveniência em tudo.** Cada registro guarda fonte, URL, data da coleta
   e o SHA-256 do arquivo bruto, que fica arquivado no bucket, como já é
   feito com os PDFs do Diário. Uma resposta sempre aponta para o
   documento original e para a nossa cópia.
2. **Cada fonte tem adapter e tabelas próprias.** Nada de forçar tudo num
   modelo único: um empenho não é um ato. O que une as fontes é a tabela de
   entidades.
3. **Ligação por chave, com grau de certeza visível.** CNPJ é ligação exata;
   processo + órgão + ano é forte; nome é fraca e nunca afirma identidade.
4. **Cobertura declarada.** Toda resposta diz o período e as lacunas da
   fonte ("despesas com CNPJ só de 2017 a 2022"). Ausência de dado não é
   evidência de ausência de fato.
5. **Determinístico por baixo.** Padrões para verificar são regras escritas
   com teste. A IA do usuário interpreta; nossos números não dependem dela.
6. **LGPD.** Agentes políticos e secretários têm página. Servidor aparece
   no ato em que é citado, sem perfil. Folha com nome só de agentes
   políticos; dos demais, só agregados por órgão e cargo. Sócio de empresa só dentro da página da
   empresa.

## Arquitetura

```
fonte ──► coletor (Cloud Run Job, agendado) ──► bruto no bucket (hash)
                                                   │
                                                   ▼
                                   normalizador ──► tabelas da fonte
                                                   │
                                                   ▼
                                   ligador ──► entities + entity_links
                                                   │
                         ┌─────────────────────────┼──────────────┐
                         ▼                         ▼              ▼
                     API HTTP              servidor MCP       padrões
                  (site e terceiros)     (IA do usuário)    (regras)
```

- **Coletor por fonte** no serviço de ingestão (mesmo padrão do scraper:
  caminho determinístico no bucket, marcador de publicado, evento no
  Pub/Sub). Guarda a resposta bruta (JSON, CSV, HTML, PDF) antes de
  interpretar, para poder reprocessar quando o normalizador mudar, como o
  `make reindex` faz com o Diário.
- **Normalizador** transforma o bruto em linhas da tabela da fonte, de forma
  idempotente (chave natural da fonte, como `numeroControlePNCP` ou
  unidade + empenho + ano).
- **Ligador** extrai chaves de cada linha, normaliza (CNPJ só dígitos;
  processo no formato `órgão/número/ano`; contrato `número/ano/órgão`) e
  grava em `entity_links(entity_id, fonte, registro, papel, certeza,
  evidência)`.
- **Tabela `fetch_runs`** registra cada coleta (fonte, período pedido,
  linhas, erros, hash) e alimenta a cobertura declarada.

### Tabelas

| Tabela | Fonte | Chave natural |
| --- | --- | --- |
| `gazettes`, `acts`, `act_entities` (existem) | D.O. da Prefeitura | edição, posição |
| `legislative_gazettes`, `legislative_acts` | D.O.E. da Câmara (mesmo pipeline de PDF, outro parser) | edição, posição |
| `bills` | SICAM (proposições e tramitação) | tipo, número, ano |
| `expenses` | TCE-RJ (`empenho_municipio`, CNPJ e valores por empenho e mês; 2025 e 2026 conferidos, ano inicial a medir), portaltp 2017–2022 (empenho, liquidação e pagamento) e, se a API for descoberta, portal Embras 2023– | unidade, empenho, ano, fase (empenho, liquidação, pagamento) |
| `procurements`, `procurement_contracts` | Mural de licitações e PNCP | processo; `numeroControlePNCP` |
| `laws` | SIAPEGOV (leis e decretos) | tipo, número, ano |
| `payroll_aggregates` | portais de transparência (Prefeitura e Câmara) | órgão, cargo, mês |
| `political_agents`, `political_agent_pay` | Câmara (vereadores), Prefeitura (prefeito, vice, secretários) | nome + mandato |
| `companies`, `company_partners` | Receita (só CNPJs que aparecem em alguma fonte) | CNPJ |
| `sanctions` | CGU (CEIS, CNEP, CEPIM) e penalidades do TCE-RJ | CNPJ/CPF + processo da sanção |
| `candidates`, `campaign_donations` | TSE | eleição, candidato; doador |
| `transfers` | Transferências e emendas da CGU (incluem FNDE e FNS), Transferegov | convênio, emenda ou repasse |
| `fiscal_reports` | SICONFI (RREO, RGF, DCA) | ente, período, anexo, conta |
| `entities`, `entity_links` | todas | tipo + chave normalizada |

Tipos de entidade: `cnpj`, `pessoa` (nome + CPF parcial quando houver),
`processo`, `contrato`, `empenho`, `norma` (lei, decreto, resolução),
`proposicao`, `orgao` (sigla do Diário ↔ CNPJ ↔ unidade do PNCP ↔ UASG) e
`agente_politico`.

## Como as fontes se cruzam

| Chave | Onde aparece | Pergunta que destrava |
| --- | --- | --- |
| **CNPJ** | Diário (extratos), empenhos (TCE-RJ, portaltp), dispensas (TCE-RJ), PNCP, planilha de obras, Receita, CEIS/CNEP, TSE (doação de empresa só até as eleições de 2014; o STF a proibiu em 2015), D.O.E. da Câmara | Quanto a empresa recebeu, de quem, por quê; quem são os sócios; se foi punida |
| **Processo** | Diário, mural de licitações, PNCP, SEI, planilha de obras, empenhos (quando trazem) | Da licitação ao pagamento de uma compra |
| **Contrato** | Diário, mural, PNCP | Aditivos, prorrogações e quanto já foi pago |
| **Órgão** | Diário (sigla), PNCP (CNPJ e unidade), Compras.gov (UASG), portais (unidade gestora) | Gastos por secretaria e fundação |
| **Norma** | Diário (lei sancionada), SICAM (projeto que virou lei), D.O.E. da Câmara (resolução), SIAPEGOV | Quem propôs, como votou a Câmara, quando entrou em vigor |
| **Pessoa** | Diário (nomeação, exoneração), Receita (sócio, CPF mascarado), TSE (candidato, doador), folha (agentes políticos) | Sócio de fornecedor que doou para candidato eleito; nomeado que é sócio de fornecedor |

A ligação de pessoa é sempre fraca: a Receita mascara o CPF dos sócios, e o
Diário só traz o nome. A página e o MCP mostram a evidência do casamento
(nome igual, dígitos visíveis iguais) e dizem "possível", nunca "é".

## Como uma pergunta é respondida

Exemplos, com o caminho pelos dados:

- **"Quanto os vereadores ganham e desde quando?"** → `norma` (Resolução
  2.156/2024 no D.O.E. da Câmara) + `political_agent_pay` (folha da Câmara,
  mês a mês) + a resolução anterior para comparar.
- **"Quanto a empresa X recebeu da Prefeitura?"** → `cnpj` → `expenses`
  (empenhado, liquidado e pago por ano e órgão, do TCE-RJ) + contratos no
  Diário e no PNCP + aviso de cobertura.
- **"Esse contrato foi pago acima do valor?"** → `contrato` → extrato e
  aditivos no Diário → empenhos ligados por processo ou CNPJ + órgão →
  padrão "pago acima do contratado mais aditivos".
- **"Algum fornecedor punido foi contratado?"** → `sanctions` ∩ `cnpj` com
  contrato ou pagamento depois do início da sanção.
- **"Quem votou a favor do aumento?"** → hoje o SICAM só dá o placar (19 a
  3). Lacuna declarada: votação nominal não está publicada em formato
  estruturado.

## Servidor MCP

Servidor remoto (HTTP "streamable"), só leitura, com chave por usuário, no
Cloud Run ao lado da API e chamando os mesmos casos de uso.

**Ferramentas**, cada uma com resposta pequena, estruturada e com
`fontes: [{nome, url, pagina, sha256, coletado_em}]` e `cobertura`:

| Ferramenta | O que faz |
| --- | --- |
| `buscar_atos` | Busca textual no Diário da Prefeitura e no da Câmara, com os filtros atuais |
| `ler_ato` | Texto completo de um ato, com citação pronta |
| `entidade` | Tudo o que liga a um CNPJ, processo, contrato, norma ou órgão, agrupado por fonte |
| `pagamentos` | Empenhos, liquidações e pagamentos filtrados por credor, órgão e período, com totais |
| `contratacoes` | Licitações e contratos (Diário, mural, PNCP) de um fornecedor ou órgão |
| `empresa` | Receita + sanções + resumo do que recebeu |
| `agente_politico` | Mandato, subsídio (norma e folha), proposições, votações quando houver |
| `padroes` | Padrões para verificar que uma entidade aciona, com a regra em texto |
| `fontes` | Lista das fontes, cobertura, data da última coleta e lacunas conhecidas |

**Recursos e prompts:** a descrição de cada fonte (este levantamento) como
recurso, para a IA saber o que existe e o que falta; prompts prontos como
"investigar fornecedor" e "seguir um contrato", que encadeiam as
ferramentas.

**Sem SQL livre no começo.** SQL somente leitura sobre visões selecionadas é
tentador para IAs, mas facilita puxar dado pessoal em massa. Reavaliar
depois, com visões sem dado de pessoa, limite de linhas e `statement_timeout`.

**Limites:** limite de requisições por cliente, como no reporte de erro;
nenhuma ferramenta devolve lista de pessoas físicas fora de agentes
políticos; paginação obrigatória.

**Por que MCP e não chat próprio:** sem custo de LLM do nosso lado, sem
responsabilidade por texto gerado, e quem investiga cruza os nossos dados
com o que já tem. O site continua sendo a porta principal para quem não usa
IA; as perguntas prontas do site e as ferramentas do MCP chamam os mesmos
casos de uso.

## Etapas

As Entregas 2 a 5 do roadmap continuam valendo; este plano detalha as
fontes de cada uma e adiciona o MCP desde cedo.

| Etapa | Conteúdo | Fontes |
| --- | --- | --- |
| **A. MCP mínimo** (entregue em 23/09/2026) | Servidor MCP sobre a API atual: `buscar_atos`, `ler_ato`, `entidade` (CNPJ), `fontes` | Diário |
| **B. Base comum** (abre a Entrega 3; entregue em 23/09/2026) | ADR do modelo de entidades; `fetch_runs`, bruto no bucket, `entities`/`entity_links` por cima de `act_entities` (ADR 0004); ADR de LGPD | — |
| **C. Diário da Câmara** | D.O.E. da Câmara no mesmo pipeline de PDF, com parser próprio; SICAM (proposições); agentes políticos e subsídios | Câmara |
| **D. Quem é o fornecedor** (Entrega 3) | Receita e sanções da CGU; página e ferramenta `empresa` | Receita, CGU |
| **E. Anunciado × pago** (Entrega 4) | Empenhos e dispensas do TCE-RJ (com CNPJ, até o mês corrente); portaltp 2017–2022 para liquidação e pagamento detalhados; mural de licitações; PNCP; SICONFI como total de controle; ferramenta `pagamentos` | TCE-RJ, Prefeitura, PNCP, Tesouro |
| **F. Recorte político** (Entrega 5) | TSE (pelo espelho da Base dos Dados, se o TSE continuar bloqueando), transferências e emendas (CGU, Transferegov), pareceres e penalidades do TCE-RJ; folha agregada | Estado e União |
| **G. Padrões nas fontes novas** | Os padrões das Entregas 3 a 5 viram ferramenta `padroes` | todas |

O Diário da Câmara vem antes da Receita porque é a lacuna que já apareceu
(o salário dos vereadores não está no Diário da Prefeitura) e reaproveita
quase todo o pipeline existente.

## Riscos

- **Portais frágeis.** Cadeia TLS incompleta, 403 sem `User-Agent`,
  aplicações só em JavaScript com API não documentada, exportação por
  postback. Cada coletor precisa de teste com a resposta real gravada e de
  alerta quando a coleta volta vazia.
- **Troca de fornecedor de portal.** A Prefeitura trocou o portal (EL →
  Embras) e a API antiga parou em 2022. Vai acontecer de novo; o bruto
  arquivado e a cobertura declarada evitam que a troca apague histórico.
  Fontes de fora da Prefeitura (TCE-RJ, SICONFI, CGU) servem de base
  estável e de conferência dos portais municipais.
- **Chave do município diferente em cada fonte.** IBGE 3304904, SIAFI 5897
  (CGU), código próprio da Receita, texto no TCE-RJ. Tabela de-para desde a
  etapa B.
- **Fontes instáveis ou bloqueadas.** O PNCP alterna 500 a 504; o TSE
  bloqueia com 403; a API de empenhos do TCE-RJ leva minutos por ano. Coleta
  com retry, incremental e fora do horário comercial.
- **Carga nos servidores municipais.** Coleta incremental, fora do horário
  comercial, com intervalo entre requisições.
- **Volume da Receita.** A base de CNPJ tem ~7,8 GB compactados por mês;
  baixar, filtrar e guardar só os CNPJs que aparecem em alguma fonte.
- **Poucas punições municipais nas bases federais.** O CEIS não tem
  nenhuma sanção aplicada pela Prefeitura; as punições da Prefeitura saem
  no próprio Diário e precisam de extração própria.
- **Ligação errada.** Processo e contrato têm formatos diferentes em cada
  fonte. Toda ligação guarda a evidência e o grau de certeza, e os testes
  usam casos reais que devem e que não devem ligar.
- **Custo.** O Neon gratuito tem limite de armazenamento; despesas e
  empenhos de dez anos devem caber, mas medir antes da carga completa.

## Decisões

1. **MCP com chave por usuário** (decidido em 23/09/2026). Cada pessoa gera
   uma chave no site; o limite de requisições e o registro de uso são por
   chave. A chave identifica o uso, não libera dado a mais: as mesmas regras
   de LGPD valem para todos.
2. **Ordem: Câmara (C) antes da Receita (D)** (decidido em 23/09/2026).

3. **Folha com nome só de agentes políticos** (decidido em 23/09/2026).
   Prefeito, vice, secretários e vereadores aparecem com nome e
   remuneração, mês a mês. Os demais servidores entram só em agregados por
   órgão e cargo (quantidade e total pago). Os portais oficiais publicam a
   folha nominal de todos, mas reunir essa folha com o Diário, a Receita e o
   TSE num só lugar facilitaria montar perfil de pessoas, o que a LGPD pede
   para evitar.
