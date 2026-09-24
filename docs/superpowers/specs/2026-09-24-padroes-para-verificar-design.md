# Padrões para verificar: fracionamento e nomeações antes da eleição (Entrega 2)

Data: 2026-09-24. Parte da Entrega 2 de `docs/roadmap.md` ("Seguir o
dinheiro dentro do Diário"): padrões para verificar, cada um com a regra
em texto e os atos que a acionaram. Linguagem sempre "padrão para
verificar", nunca "irregularidade".

## Objetivo

Uma página `/padroes` e um endpoint `/v1/patterns` que listam, para cada
regra fixa, a regra por extenso e os casos que a acionam, com os atos (ou a
busca que os reúne) para o leitor conferir na edição original.

Este primeiro subprojeto monta a estrutura e as duas regras que os dados
sustentam hoje. As outras duas do roadmap ficam para depois (ver "Fora do
escopo").

## Fatos que orientam o desenho

Medidos na base local (Diário da Prefeitura, 2010 a 2026):

- **Dispensas.** 651 atos do tipo `dispensa` e 700 com modalidade
  `dispensa`; 322 têm CNPJ de fornecedor e valor principal lido do texto.
  A mesma contratação sai várias vezes no Diário (aviso, ratificação,
  extrato do contrato), com o mesmo valor e o mesmo processo: contar atos
  em vez de contratações gera falso positivo (em 2024, a SEMTRAN publicou o
  processo 18.635/2024 três vezes em seis dias).
- **Limite da dispensa por valor** (compras e serviços comuns), conferido
  em 24/09/2026:
  - até 18/07/2018: R$ 8.000,00 (Lei 8.666, art. 24, II);
  - de 19/07/2018: R$ 17.600,00 (Decreto 9.412/2018, vigente 30 dias após
    a publicação de 19/06/2018);
  - Lei 14.133, art. 75, II: R$ 50.000,00 (2021 e 2022), R$ 57.208,33
    (2023, Decreto 11.317/2022), R$ 59.906,02 (2024, Decreto 11.871/2023),
    R$ 62.725,59 (2025, Decreto 12.343/2024), R$ 65.492,11 (2026, Decreto
    12.807/2025).
  - De 01/04/2021 a 29/12/2023 as duas leis conviveram; o texto do ato diz
    qual usou quando cita "14.133".
- **A lei soma por ano**: o art. 75, § 1º, da Lei 14.133 manda somar o
  despendido no exercício financeiro pela unidade gestora com objetos de
  mesma natureza. O Diário não diz a natureza do objeto de forma
  determinística; o mesmo CNPJ é a aproximação possível.
- **Falsos positivos achados na simulação** (contratação = processo, mesmo
  CNPJ, mesmo ano, cada uma abaixo do limite e a soma acima):
  - dispensa que não é por valor: art. 24, XIII e XXII da Lei 8.666
    (pesquisa, energia elétrica). Só 67 dos 276 atos candidatos citam o
    inciso II do art. 24 ou do art. 75;
  - republicação por incorreção com o processo corrigido (em 2020, a SEMAD
    publicou a mesma ratificação com o processo 53.639/2018 e, seis dias
    depois, 56.639/2018, "Republicado por incorreção da PMSG");
  - ato mal segmentado: um edital da SEMHAB de 30/07/2026 traz colada a
    autorização da SMTC do processo 7405/2026 e a de outro fornecedor, do
    processo 07537/2026.
  Com as três correções abaixo, sobra 1 caso: o mesmo fornecedor com duas
  compras em 2020 (SEMMA, R$ 11.862,00; SEMAD, R$ 17.049,60), acima do
  limite de R$ 17.600,00.
- **Nomeações e exonerações.** 23.502 atos de nomeação e 17.330 de
  exoneração. Eleições municipais na base: 07/10/2012, 02/10/2016,
  15/11/2020 e 06/10/2024. Comparando cada um dos seis meses antes do mês
  da eleição com a mediana do mesmo mês nos anos sem eleição, a razão 1,5
  aciona 6 meses (abril de 2012 e 2016, agosto e setembro de 2020,
  setembro de 2016); a razão 2 não aciona nenhum.

## Decisões

1. **Calculado na leitura**, sem migration nem tabela de resultados: as
   consultas passam por algumas centenas de dispensas e uma agregação
   mensal. Quando a base mudar (reindexação, edição nova), o resultado muda
   junto.
2. **Regra no domínio, consulta no Postgres.** O Postgres devolve as
   dispensas candidatas e as contagens mensais; as funções do domínio
   aplicam a regra e são testadas sem banco.
3. **Contratação é o processo**, por CNPJ. Atos do mesmo CNPJ que citam um
   mesmo processo são a mesma contratação, e um ato que cita dois processos
   junta os dois (componentes ligados). Ato sem processo, ou que se diz
   republicado, junta-se à contratação do mesmo CNPJ com o mesmo valor no
   mesmo ano; sem par, é contratação própria. O valor da contratação é o
   maior entre os atos; a data é a da primeira publicação.
4. **Só dispensa por valor de compras e serviços**: a contratação entra se
   algum ato dela cita o inciso II do art. 24 (Lei 8.666) ou do art. 75
   (Lei 14.133). Obras (inciso I) ficam para depois. Fora também: aditivos,
   atos que citam emergência ou calamidade, contratações com valor igual
   ou acima do limite e CNPJs de órgãos públicos (`domain.PublicBody`).
5. **Limite de cada contratação** pela data da primeira publicação e pela
   lei citada; o caso usa o maior limite entre as contratações do grupo
   (o mais favorável ao órgão).
6. **Pico antes da eleição**: mês entre os seis anteriores ao mês da
   eleição com contagem de atos do tipo pelo menos 1,5 vez a mediana do
   mesmo mês nos anos sem eleição municipal (só meses que existem na
   base). Só o Diário da Prefeitura.
7. **O caso de pico não lista os atos** (são centenas): traz a busca que
   os reúne (tipo, período e diário). O caso de fracionamento lista os
   atos de cada contratação.

## Fora do escopo

- Aditivos acima de 25%: o valor do aditivo lido do texto é quase sempre
  o de uma prorrogação, não o acréscimo; precisa de extração própria.
- Contratação emergencial renovada: 23 CNPJs com atos em mais de um mês
  que citam emergência, e "emergência" aparece em contextos variados;
  precisa de regra de texto mais estreita.
- Ferramenta `padroes` no MCP (etapa G do plano de fontes) e alerta por
  padrão.

Fica registrado em `docs/parser-findings.md`: "AUTORIZAÇÃO DA DESPESA E
ADJUDICAÇÃO" não abre ato, e as autorizações de dispensa da SMTC ficam
coladas no ato anterior. Corrigir o parser é outro trabalho (pede
reindexação).

## Desenho

### Domínio

- `pattern.go`: `PatternID` (`fracionamento_dispensa`,
  `pico_pessoal_eleicao`), `Pattern{ID, Title, Rule, Caveat}` e o
  catálogo `Patterns()`; `Finding{Title, Detail, ActIDs []string, Search
  *ActFilter}`; `PatternReport{Pattern, Findings []Finding, Acts
  map[string]ActHit}` para a apresentação.
- `dispensa_limit.go`: `DispensaLimitCents(published time.Time, citesLei14133
  bool) int64` com a tabela acima.
- `dispensa_text.go`: `CitesValueDispensa(body)`, `CitesLei14133(body)`,
  `CitesEmergency(body)` e `IsRepublication(body)`, regras de texto
  testadas com trechos reais.
- `split_dispensa.go`: `DispensaAct{ActID, CNPJ, ProcessKeys []string,
  Organ, PublishedAt, ValueCents, Body}`,
  `SplitDispensa{CNPJ, Year, Contracts []DispensaContract, TotalCents,
  LimitCents}`, `DispensaContract{ProcessKey, FirstPublished, ValueCents,
  LimitCents, Organs, ActIDs}` e `FindSplitDispensas([]DispensaAct)
  []SplitDispensa`, ordenado por ano e CNPJ.
- `election_hiring.go`: `MunicipalElections` (ano → data),
  `MonthlyActCount{Type, Year, Month, Count}`, `HiringPeak{Type, Year,
  Month, Count, BaselineMedian float64, BaselineYears int}` e
  `FindElectionPeaks([]MonthlyActCount) []HiringPeak`.
- Os achados viram `Finding` com títulos e detalhes em português:
  - fracionamento: "CNPJ 53.775.862/0001-52 em 2020: 2 dispensas somam
    R$ 28.911,60, acima do limite de R$ 17.600,00"; detalhe com cada
    contratação (processo, data, valor, órgãos);
  - pico: "Agosto de 2020: 166 atos de nomeação"; detalhe "Mediana de
    agosto nos 13 anos sem eleição municipal: 107. Eleição em
    15/11/2020."

### Portas, Postgres e caso de uso

- `ports.PatternSource`: `DispensaActs(ctx)` e `MonthlyActCounts(ctx,
  types []ActType, source string)`.
- `postgres.PatternRepo`: dispensas com CNPJ ligado e valor principal
  (`type = 'dispensa' OR modality = 'dispensa'`, `type <> 'aditivo'`,
  Diário da Prefeitura), com os processos ligados ao ato e o corpo (a
  regra de texto fica no domínio); contagem por tipo, ano e mês.
- `usecase.ListPatterns.Execute(ctx) ([]domain.PatternReport, error)`:
  roda as duas regras e carrega os atos citados com
  `ActRepository.HitsByIDs` (novo, mesma forma de `ActHit` da busca, sem
  trecho destacado).

### HTTP

`GET /v1/patterns` → `{"items": [{"id", "title", "rule", "caveat",
"findings": [{"title", "detail", "acts": [ato como na busca], "search":
{"type", "from", "to", "source"} | null}]}]}`, com `Cache-Control:
public, max-age=3600`.

### Front

Página `/padroes`: aviso no topo ("padrões para verificar, não
irregularidades; confira sempre a edição original"), e para cada padrão o
título, a regra, a ressalva e os casos. Caso de fracionamento com os atos
(`Result`); caso de pico com o link para a busca (`/?tipo=…&de=…&ate=…
&fonte=diario_prefeitura`). Link "Padrões para verificar" junto de
"Dados abertos" e "MCP" na página inicial.

## Testes

- Domínio:
  - `DispensaLimitCents` nas bordas de cada faixa e com e sem a Lei 14.133
    entre 2021 e 2023;
  - regras de texto com os trechos reais (art. 24, inciso II; art. 75,
    inciso II; art. 24, inciso XIII não conta; "Republicado por
    incorreção");
  - `FindSplitDispensas`: o caso real de 2020 (SEMMA e SEMAD) aciona; a
    republicação da SEMAD não conta como terceira contratação; o caso
    real da SEMTRAN de 2024 (três atos do mesmo processo) não aciona; o
    edital colado de 2026 junta-se ao contrato do mesmo processo e não
    aciona; emergência, CNPJ de órgão público e valor acima do limite
    ficam de fora; anos diferentes não somam;
  - `FindElectionPeaks`: aciona na razão 1,5 e não abaixo; mês fora da
    janela não aciona; ano de eleição não entra na mediana.
- Integração (Postgres real): edições com o texto real dos extratos de
  2020 (SEMMA, SEMAD e a republicação) e dos três atos da SEMTRAN de 2024; `/v1/patterns`
  traz um caso de fracionamento com os dois atos de 2020 e nenhum com os da
  SEMTRAN; contagens mensais por tipo.
- Front (Vitest): URL da busca do caso de pico.
