# Fontes do Diário SG

O que o servidor sabe, de onde vem, que ferramenta devolve e o que falta.
Para o período coberto e a última coleta de cada Diário, chame `fontes`.

## Diários oficiais

- **Diário Oficial da Prefeitura de São Gonçalo**: edições em PDF,
  quebradas em atos. Ferramentas: `buscar_atos`, `ler_ato`, `entidade`,
  `agrupar`, `pagina_original`, `padroes`.
- **Diário Oficial Eletrônico da Câmara Municipal**: mesmas ferramentas,
  com `diario` = camara. As edições de 2018 a 3/10/2020 existem, mas não
  estão na base: o site não as lista por data e a busca antiga tem
  captcha. Edições escaneadas são lidas por OCR e o ato avisa.

## Empresa (ficha em `entidade` com tipo cnpj)

- **Receita Federal**: cadastro do CNPJ (razão social, situação,
  abertura, capital, atividade, endereço, sócios com CPF mascarado), do
  arquivo mensal público.
- **CGU**: sanções do CEIS e do CNEP e impedimentos do CEPIM.
- **Punições publicadas no Diário**: advertência, multa, suspensão,
  impedimento e inidoneidade aplicadas pela Prefeitura ou pela Câmara,
  lidas dos atos ligados ao CNPJ.
- **PNCP**: contratos do município com a empresa (Lei 14.133), quase
  todos de 2024 em diante.
- **Emendas parlamentares federais** pagas à empresa (Portal da
  Transparência).

## Dinheiro do município

- **Portal da transparência da Prefeitura**: empenhos de 2017 em diante,
  de todas as entidades (Prefeitura, fundos, fundações, SG-PREVI, Câmara),
  com credor, objeto, processo e modalidade. Ferramentas: `pagamentos` e,
  por empresa, `entidade`.
- **TCE-RJ**: empenhado, liquidado e pago por credor e ano
  (`pagamentos_tce` em `entidade`, com o período carregado), obras
  paralisadas, parecer prévio das contas e débitos e multas.
- **SICONFI (Tesouro)**: RREO do município, usado como total de controle
  do pago por ano (`pago_rreo_centavos` em `pagamentos`).
- **Mural de licitações e contratos da Prefeitura**: licitações,
  dispensas, inexigibilidades, contratos e atas, com o link do documento.
  Sem CNPJ: a ligação com a empresa é pelo processo dos empenhos.
  Ferramenta: `contratacoes`.
- **Consulta de leis (SIAPEGOV)**: leis, leis complementares, Lei Orgânica
  e decretos. Ferramenta: `norma`.

## Pessoas públicas

- **Folhas da Prefeitura e da Câmara**: remuneração de prefeito, vice,
  secretários, Procurador-Geral e vereadores, mais partido e nome
  parlamentar do SICAM. Ferramenta: `agentes_politicos`. Servidores não
  têm perfil: aparecem só nos atos.

## União

- **Transferências da União, emendas com aplicação na cidade e
  transferências especiais (Transferegov)**: na página `/federal` do site.

## O que não está na base

- **TSE** (candidatos, bens, doações): o CDN e a API de dados abertos
  negam acesso à máquina da coleta; o espelho da Base dos Dados pede conta
  na plataforma ou projeto com cobrança no BigQuery.
- **SICAM** (proposições e votações da Câmara): a API recusa pedidos de
  fora da página da Câmara.
- **Portal antigo da Prefeitura (portaltp)**: o WAF recusa o robô; os
  mesmos anos estão no portal novo.

Nenhuma fonte é lida contornando captcha, WAF ou controle de acesso, e a
base não guarda CPF nem monta perfil de pessoa física.
