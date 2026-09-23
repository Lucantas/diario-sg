# ADR 0007 — Diário da Câmara nas mesmas tabelas do Diário da Prefeitura

**Status:** aceito

## Contexto
O Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo tem o mesmo
formato que o da Prefeitura: um PDF por edição, dividido em atos
(portaria, resolução, extrato, aviso de licitação, termo de prestação de
contas da cota parlamentar). O parser da Prefeitura separa e classifica
esses atos quase sem mudança. Só o cabeçalho de página e o número da
edição são diferentes.

A busca, a exportação, o RSS, o alerta por e-mail, a citação, a cópia
arquivada do PDF, o relatório de erro, o dump e o MCP leem de `gazettes`
e `acts`.

## Decisão
- `gazettes` ganha a coluna `source` (`diario_prefeitura`, padrão, ou
  `diario_camara`). Os atos da Câmara ficam em `acts`, como os da
  Prefeitura.
- Um só parser, com o ruído de página e o número da edição escolhidos
  pela fonte (`parser.ForSource`).
- Um só coletor, com a fonte escolhida pela variável `SOURCE`, e um job
  agendado por fonte.
- As ligações de entidades (ADR 0004) levam a fonte da edição.
- Busca, API e MCP aceitam filtrar pela fonte, e todo ato traz a fonte e
  a citação da fonte certa.

## Consequências
- Tudo o que já existe passa a valer para a Câmara sem código novo, e quem
  busca vê os dois diários juntos por padrão.
- Uma fonte que não tenha a forma "edição com atos" (proposições do
  SICAM, folha, despesas) não entra aqui; ela ganha tabelas próprias e
  usa `entity_links`.
- O filtro de órgão é das secretarias da Prefeitura; os atos da Câmara
  ficam sem órgão.
