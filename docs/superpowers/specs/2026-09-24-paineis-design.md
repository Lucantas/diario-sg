# Painéis: maiores fornecedores por valor, por secretaria e por ano (Entrega 2)

Data: 2026-09-24. Último item da Entrega 2 de `docs/roadmap.md` ("Seguir o
dinheiro dentro do Diário"): "Painéis. Maiores fornecedores por valor, por
secretaria e por ano."

## Objetivo

Uma página `/paineis` e um endpoint `GET /v1/panels/suppliers` com o
ranking dos fornecedores pelo valor declarado nos extratos do Diário. O
ranking pode ser filtrado por ano e por secretaria, e vem acompanhado dos
totais por ano e por secretaria, para o leitor trocar de recorte. Cada
fornecedor leva à página da empresa e ao ato que deu o maior valor, para
conferir na edição original.

## Fatos que orientam o desenho

Medidos na base local (Diário da Prefeitura, 2010 a 2026):

- **O mesmo dinheiro aparece várias vezes.** A homologação do pregão, a ata,
  o extrato do contrato e os aditivos saem com o mesmo processo e valores
  parecidos. Em 2025, o contrato 010/SEMED/2025 (R$ 106,3 milhões) saiu no
  mesmo dia da homologação com o mesmo valor. Somar atos conta o mesmo
  dinheiro duas ou três vezes.
- **A homologação traz o valor do certame, não o do fornecedor.** A
  homologação do pregão 90008/2025 da SEMED (merenda) diz R$ 53,2 milhões,
  e a ata do único fornecedor ligado a ela, R$ 5,9 milhões. Muitas
  homologações não dizem "homologação" no título: começam por "PREGÃO
  ELETRÔNICO Nº …" e seguem com "Homologo o resultado" ou "Nos termos do
  relatório final apresentado pelo Pregoeiro".
- **A ata de registro de preços é um teto, não uma compra.** Das 1.739
  publicações com valor e um só fornecedor nos tipos contrato, dispensa,
  licitação e ata, 933 são atas de registro de preços (ou o extrato
  trimestral delas), e somam mais que os contratos. A maior é a ata
  072/FMS/2026, com R$ 173,0 milhões no lote. Misturar teto com contrato
  põe no topo quem só registrou preço.
- **Aditivo tem valor ambíguo.** Pode ser o valor da prorrogação por mais
  12 meses, o do acréscimo ou o novo total (é o mesmo motivo que adiou a
  regra de aditivos acima de 25% dos padrões).
- **Ato com mais de um fornecedor.** O valor do extrato de uma ata com
  vários vencedores não é de nenhum deles sozinho.
- **O Diário não traz o nome da empresa de forma estruturada.** Nome vem da
  Receita Federal, na Entrega 3; o painel mostra o CNPJ.

Com as regras abaixo, a base local fica com 1.373 contratações de 906
fornecedores: R$ 904,7 milhões contratados e R$ 1,22 bilhão registrados em
atas. O maior contratado é o CNPJ 14.180.324/0001-63 (R$ 126,1 milhões,
dos quais R$ 106,3 milhões do contrato 010/SEMED/2025).

## Decisões

1. **Calculado na leitura**, como os padrões: o Postgres devolve os atos
   candidatos, o domínio monta as contratações e o ranking. Sem migration.
   `Cache-Control: public, max-age=3600`.
2. **Atos candidatos:** tipos `contrato`, `dispensa`, `licitacao` e `ata`,
   com valor principal lido do texto, e exatamente um CNPJ que não é de
   órgão público (`domain.PublicBody`). Aditivos ficam de fora.
3. **Contratação** por fornecedor, pela mesma junção dos padrões: atos que
   citam um mesmo processo ou um mesmo contrato são a mesma contratação;
   ato sem processo nem contrato junta-se ao do mesmo CNPJ com o mesmo
   valor no mesmo ano.
4. **O valor de cada ato** tem um de três papéis:
   - **nenhum**: fase de homologação, aditivo, rescisão, ajuste de contas ou
     fiscal (`domain.PhaseOf`), ou texto de homologação, adjudicação, aviso,
     resultado, multa, sanção, penalidade, notificação ou cancelamento no
     começo do ato (a revisão achou uma multa de R$ 16 milhões de 2018
     contada como contrato);
   - **registrado**: ato que começa por "ata de registro de preços" ou
     "extrato (de publicação) trimestral", ou com fase de ata de registro de
     preços sem
     partes nem número de contrato no começo do texto;
   - **contratado**: todos os outros (contrato, dispensa, adesão a ata,
     chamamento com as partes, contrato de gestão).
5. **O valor da contratação** é o maior valor contratado entre os atos dela.
   Sem valor contratado, é o maior valor registrado, contado à parte. Sem
   nenhum dos dois, a contratação não entra.
6. **Ano** é o da primeira publicação da contratação; **secretaria** é a
   sigla principal (`domain.PrincipalOrgan`) dos órgãos dos atos dela. Uma
   contratação com atos de dois órgãos conta nos dois.
7. **Ranking** pelo valor contratado, depois pelo registrado, depois pelo
   CNPJ; os 50 primeiros. Os totais por ano respeitam o filtro de
   secretaria, e os totais por secretaria respeitam o filtro de ano.
8. **Só o Diário da Prefeitura por padrão**; `source` aceita o da Câmara.

## Fora do escopo

- Nome do fornecedor (Entrega 3, Receita Federal).
- Valor de aditivos e de prorrogações: precisa da mesma extração própria
  adiada nos padrões.
- Painel por objeto ou por modalidade.
- Ferramenta no MCP.

## Desenho

### Domínio

- `panel_value.go`: `PanelValueRole` (`PanelValueNone`,
  `PanelValueContracted`, `PanelValueRegistered`) e
  `PanelValueRoleOf(t ActType, title, head string)`, com as regras de
  texto da decisão 4 testadas com trechos reais.
- `supplier_panel.go`:
  - `PanelAct{ActID, CNPJ, OtherCNPJs, Refs, Organ, PublishedAt,
    ValueCents, Type, Title, Head}` (`Refs` são chaves `processo:…` e
    `contrato:…`);
  - `PanelFilter{Year int, Organ string}`;
  - `SupplierRow{CNPJ, Contracts, ContractedCents, RegisteredCents, First,
    Last, Organs, LargestActID}`;
  - `PanelTotal{Key string, Contracts, ContractedCents, RegisteredCents}`;
  - `SupplierPanel{Filter, Contracts, Suppliers, ContractedCents,
    RegisteredCents, Rows, Years, Organs}`;
  - `BuildSupplierPanel(acts []PanelAct, f PanelFilter) SupplierPanel`.
- A junção por processo reaproveita o `unionFind` dos padrões.

### Portas, Postgres e caso de uso

- `ports.PanelSource`: `PanelActs(ctx, source)` e `HitsByIDs(ctx, ids)`.
- `postgres.PanelRepo.PanelActs`: atos dos quatro tipos com valor, os CNPJs
  e as referências (processo e contrato) ligados em `entity_links` da mesma
  fonte, título e os 400 primeiros caracteres do corpo. `HitsByIDs` vira
  função do pacote, usada pelos dois repositórios.
- `usecase.GetSupplierPanel.Execute(ctx, source, filter)`: valida a fonte,
  monta o painel e carrega os atos de maior valor de cada linha.

### HTTP

`GET /v1/panels/suppliers?year=2024&organ=SEMED&source=diario_prefeitura`
→

```json
{"source": "diario_prefeitura", "year": 2024, "organ": "SEMED", "organ_name": "…",
 "contracts": 10, "suppliers": 8, "contracted_cents": 0, "registered_cents": 0,
 "items": [{"cnpj": "…", "contracts": 2, "contracted_cents": 0, "registered_cents": 0,
            "first": "2024-01-02", "last": "2024-05-03", "organs": ["SEMED"],
            "largest": {ato como na busca}}],
 "years": [{"year": 2024, "contracts": 10, "contracted_cents": 0, "registered_cents": 0}],
 "organs": [{"organ": "SEMED", "organ_name": "…", "contracts": 10, "contracted_cents": 0, "registered_cents": 0}]}
```

`year` fora de 2000 a 2100 ou não numérico, e `source` desconhecido, dão
400. `year` sem filtro sai `null`.

### Front

Página `/paineis`: aviso do que o valor é (declarado nos extratos, não
pagamento; cada contratação uma vez; atas à parte; o que fica de fora),
filtros de ano e secretaria (no endereço, `?ano=&orgao=`), a tabela dos
maiores fornecedores (CNPJ com link para `/empresa/…`, contratações, valor
contratado, valor em atas, período, secretarias e o link do ato de maior
valor) e as tabelas por ano e por secretaria, em que cada linha aplica o
filtro. Link "Painéis" no rodapé da página inicial.

## Ajustes depois da entrega (25/09/2026)

Uma auditoria dos 4.358 atos com valor que o painel lê, rodando a regra do
painel sobre a base local, achou atos que não são contratação nova contados
como contratados. O título nem sempre diz o que o ato é ("CONTRATO N°
005/2019 … OBJETO: O presente termo aditivo tem por objeto a prorrogação"),
então o painel lê o começo do corpo:

- nos primeiros 400 caracteres: termo aditivo, aditamento, prorrogação do
  contrato, do prazo ou da ata, rerratificação, "Onde se lê"/"Leia-se",
  corrigenda, revogação, readequação, reajuste, designação de fiscal,
  editais e publicação de chamamento, termo de permissão de uso e ata de
  reunião ou assembleia ficam sem valor;
- nos primeiros 800: "HOMOLOGO", "HOMOLOGANDO", "fica a homologação" ficam
  sem valor, salvo em dispensa e inexigibilidade, em que a homologação é a
  própria contratação;
- extrato de ata numerada ("EXTRATO DE ATA 024/SEMDUR/2021 DE REGISTRO DE
  PREÇOS") é ata.

O valor principal também deixou de pegar o preço unitário das tabelas de
preços (ver `docs/parser-findings.md`). Na base local, as contratações com
valor passaram de 1.373 para 1.299, o contratado de R$ 904,7 milhões para R$
833,6 milhões e as atas de R$ 1,22 bilhão para R$ 1,30 bilhão.

## Testes

- Domínio:
  - `PanelValueRoleOf` com trechos reais: extrato de contrato, ratificação
    de dispensa, adesão a ata e contrato de pregão SRP contam como
    contratado; ata de registro de preços e extrato trimestral, como
    registrado; homologação com e sem a palavra no título, aviso e aditivo,
    como nenhum;
  - `BuildSupplierPanel`: homologação e contrato do mesmo processo contam
    uma vez pelo contrato; ata e contrato do mesmo processo contam como
    contratado; ata sozinha conta como registrado; ato com dois fornecedores
    e CNPJ de órgão público ficam de fora; filtro de ano e de secretaria;
    totais por ano respeitam a secretaria e por secretaria respeitam o ano;
    variante de sigla vira a principal; ordem e limite de 50.
- Integração (Postgres real): edições com os extratos reais da
  homologação e do contrato 010/SEMED/2025 e de uma ata de registro de
  preços; `/v1/panels/suppliers` conta o contrato uma vez, põe a ata em
  "registrado", filtra por ano e secretaria e recusa ano e fonte inválidos.
- Front (Vitest): leitura e escrita dos filtros no endereço.
