# Padrões no MCP (etapa G do plano de fontes)

## Problema

Os padrões para verificar (Entregas 2 a 5) só aparecem na página
`/padroes` e em `GET /v1/patterns`. Quem investiga pela IA não consegue
perguntar "esta empresa aciona algum padrão?" sem baixar a lista inteira,
e cada achado só diz de quem é no texto do título. A etapa G do plano de
fontes pede a ferramenta `padroes`; o plano também pede prompts prontos
que encadeiam as ferramentas.

## O que muda

- **Achado com as chaves.** `Finding` ganha `Entities` (tipo, chave e
  rótulo): o CNPJ do fornecedor em todo padrão de fornecedor, de
  pagamento e do PNCP; todos os CNPJs do grupo em sócio e endereço em
  comum; CNPJ e processos no fracionamento de dispensa; o contrato no
  aditivo acima do limite; o CNPJ na emergencial renovada quando o
  fornecedor é identificado pelo CNPJ. O pico de pessoal na eleição não
  tem entidade. A chave é a mesma da ferramenta `entidade` (CNPJ com 14
  dígitos, processo e contrato normalizados), para a IA seguir de uma para
  a outra.
- **Ferramenta `padroes`.** Argumentos opcionais: `cnpj`, `processo`,
  `contrato` (um de cada vez), `padrao` (o id) e `pular`.
  - Sem nada: o catálogo, cada padrão com id, título, regra, ressalva e
    número de achados, e os 5 primeiros achados de cada.
  - Com `padrao`: até 20 achados daquele padrão, com `pular` para seguir.
  - Com uma entidade: só os achados que citam aquela chave, de todos os
    padrões (ou do `padrao` pedido).
  - Cada achado: título, detalhe, entidades, atos (diário, edição, data,
    página, título e link da página do ato) e a busca ou o link do site.
- **Cache.** O cálculo dos padrões lê a base inteira; o MCP guarda o
  resultado por 10 minutos, como já faz com a cobertura.
- **Prompts.** `investigar_fornecedor` (argumento `cnpj`) e
  `seguir_contrato` (argumento `numero`, de processo ou contrato): texto
  que pede à IA para chamar `entidade`, `padroes`, `buscar_atos` e
  `ler_ato` nessa ordem, citar edição, data e página e tratar padrão como
  pista, não como acusação.

## Cuidados

- Padrão é "para verificar": a descrição da ferramenta e cada resposta
  repetem a ressalva do padrão.
- Nenhuma pessoa física: o sócio em comum já sai sem o nome da pessoa, e
  as entidades são só CNPJ, processo e contrato.

## Fora

- Recurso MCP com o levantamento das fontes: o `docs/fontes/README.md`
  fica fora do módulo Go e a ferramenta `fontes` já diz cobertura e
  lacunas.
- Alerta por e-mail quando uma entidade passa a acionar um padrão.
