# ADR 0004 — Entidades e ligações entre fontes

**Status:** aceito

## Contexto
O plano de fontes (`docs/plano-fontes-publicas.md`) junta o Diário, a
Câmara, o TCE-RJ, o PNCP, a Receita e outras fontes. O que une um ato, um
empenho e um registro da Receita são chaves: CNPJ, processo, contrato,
órgão, norma, pessoa. Hoje só existe `act_entities`, que guarda o que o
parser extrai de cada ato do Diário (CNPJ, valor, contrato, processo) e
alimenta busca, filtros, dump público e a página do CNPJ.

## Decisão
- **Duas tabelas novas por cima, sem substituir `act_entities`.**
  - `entities (kind, key)` é o catálogo de chaves.
  - `entity_links` liga uma entidade a um registro de qualquer fonte:
    fonte, tipo de registro, id do registro, papel, certeza e evidência.
  - `act_entities` continua sendo a extração do Diário. A busca, o dump e
    a página do CNPJ não mudam.
- **Valor não é entidade:** é atributo do registro, não chave de
  cruzamento. Continua em `act_entities`.
- **A regra vive no banco**, em funções SQL criadas pela migration 008:
  `entity_key`, `link_certainty` e `link_diario_acts` (liga os atos de uma
  edição). O ligador da indexação e o preenchimento inicial chamam a mesma
  função. O domínio em Go espelha a regra (`domain.EntityKey`,
  `domain.LinkCertainty`) só para validar o que a pessoa digita, e um
  teste de integração confere que Go e SQL concordam.
- **Chave normalizada:**
  - CNPJ só com dígitos;
  - processo só com dígitos (como o extrator já faz);
  - contrato sem espaços, em maiúsculas e sem zeros à esquerda no número
    (`001/2017` e `1/2017` são a mesma chave).
- **Certeza da ligação**, sempre visível para quem lê:
  - `exata`: CNPJ;
  - `forte`: processo, e contrato com a sigla do órgão no número;
  - `fraca`: contrato só com número e ano, que se repete entre órgãos
    (cada secretaria tem o seu `1/2017`). A resposta avisa que pode juntar
    contratos diferentes.
  - Nome de pessoa, quando entrar, será sempre `fraca` e dito como
    "possível", nunca como "é".
- **Evidência:** o trecho como apareceu na fonte (`Processo SEI nº
  06.10981/2025-7`), para quem lê conferir o casamento.
- **Registro do Diário** é o ato (`record_kind = ato`, `record_id` = id do
  ato). Na indexação e na reindexação, as ligações do Diário são
  refeitas na mesma transação que troca os atos.
- **Tipos** entram no `CHECK` quando uma fonte passa a gravá-los. Hoje:
  `cnpj`, `processo`, `contrato`. Os próximos já previstos no plano são
  `orgao`, `norma`, `pessoa`, `agente_politico`, `empenho` e `proposicao`.

## Consequências
- Duas tabelas com informação parecida para o Diário: a extração e a
  ligação. A migration 008 preenche `entity_links` a partir de
  `act_entities` chamando `link_diario_acts` para cada edição.
- Fonte nova grava só nas suas tabelas e em `entity_links`, sem tocar no
  Diário.
- Mudar a regra de chave ou de certeza exige migration que troque as
  funções e refaça as ligações, e o espelho em Go muda junto.
