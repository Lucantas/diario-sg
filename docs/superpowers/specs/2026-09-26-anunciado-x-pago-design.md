# Anunciado × pago (Entrega 4, parte 2)

Data: 2026-09-26. Os três padrões da Entrega 4, que cruzam as contratações
publicadas no Diário com os pagamentos do TCE-RJ (parte 1). Entram em
`/padroes` e `GET /v1/patterns`, com a mesma linguagem ("padrão para
verificar").

## Fatos que orientam o desenho

Medidos na base local em 26/09/2026:

- 515 CNPJs receberam pagamento da Prefeitura (fora a Câmara) de 2021 a
  2026 sem aparecer em nenhum ato do Diário, somando R$ 1 bilhão. A maior
  parte é de "encargos especiais" (dívida, precatórios, PASEP) e
  "previdência social" (INSS): tirando essas funções, os órgãos públicos
  e os pagamentos abaixo de R$ 100 mil, sobram 95 credores e R$ 584
  milhões.
- Contratações de pelo menos R$ 100 mil publicadas de 2021 em diante sem
  nenhum pagamento à empresa no ano da publicação nem no seguinte: 9.
- O TCE não traz o número do contrato, então o pago só se compara com o
  anunciado pela empresa inteira. Muitos extratos não trazem o CNPJ: os
  aditivos da coleta de lixo (Força Ambiental, R$ 70 a 90 milhões por
  ano) só têm o nome. Somando só os atos com CNPJ, 103 empresas recebem
  mais do que o Diário anuncia; atribuindo pelo nome os 3.045 atos com
  valor e sem CNPJ (396 atribuídos), e exigindo pago de pelo menos o dobro
  e R$ 1 milhão acima do anunciado, 68.

## Decisões

1. **Pago sem publicação** (`pago_sem_publicacao`): credor pessoa
   jurídica que recebeu da Prefeitura, de fundos e de fundações (fora a
   Câmara) pelo menos R$ 100 mil de 2021 em diante, sem nenhum ato do
   Diário que cite o CNPJ. Fora: funções "encargos especiais" e
   "previdência social", órgãos públicos (natureza jurídica de direito
   público no cadastro da Receita e os CNPJs municipais já conhecidos).
2. **Contratação sem pagamento** (`contrato_sem_pagamento`): contratação
   de pelo menos R$ 100 mil publicada de 2021 em diante (dentro da
   cobertura do pago), sem pagamento à empresa no ano da publicação nem no
   seguinte.
3. **Pago acima do anunciado** (`pago_acima_do_anunciado`): a empresa
   recebeu de 2021 em diante pelo menos o dobro de tudo o que o Diário
   anunciou para ela desde 2010 (contratações, atas e aditivos, como nos
   painéis), com diferença de pelo menos R$ 1 milhão. Atos com valor e sem
   CNPJ contam quando o nome do fornecedor lido do texto
   (`SupplierNameOf`) é igual à razão social de um único credor.
4. **Nomes dos credores**: a carga da Receita passa a incluir os CNPJs que
   aparecem nos pagamentos, além dos citados no Diário.
5. **Leitura na consulta**, na mesma `SupplierPatternSource` da Entrega 3
   (pagamentos por credor, CNPJs citados, cobertura, atos sem CNPJ).

## Fora do escopo

- Ligar pagamento a contrato (o TCE não traz o número).
- Pago pela Câmara (as contratações da Câmara têm painel próprio; fica
  para quando a Câmara tiver padrões).

## Testes

- Domínio: cada finder com casos que acionam e parecidos que não (órgão
  público, encargos, abaixo do piso, contrato fora da cobertura, pagamento
  no ano seguinte, anunciado por nome, nome ambíguo).
- Integração: `/v1/patterns` com um caso de cada.

## Depois da entrega

- Com a Receita recarregada incluindo os credores (26/09/2026, 1h52:
  4.919 CNPJs, 4.720 no cadastro), "pago sem publicação" caiu de 95 para
  75 casos: 20 credores de direito público saíram pela natureza jurídica,
  e todos os que ficaram têm nome. "Contratação sem pagamento" tem 9
  casos e "pago acima do anunciado", 68.
- Conferência de um caso de cada: Hashimoto, Perfil X Construtora e
  Consórcio Ônix/F.P. Vieira (pago sem publicação) aparecem no Diário só
  pelo nome, sem CNPJ, em extratos, portarias de fiscal e decretos, às
  vezes com outra razão social ("HASHIMOTO MANUTENÇÃO ELÉTRICA"); a
  ressalva do padrão diz isso, e cada caso ganhou o link "Procurar o nome
  no Diário" (a razão social atual, sem o sufixo). CEJOM (contratação sem
  pagamento, termo de adesão de R$ 5 milhões em 28/12/2022) não tem
  nenhum empenho no TCE em nenhum ano. Ampla (pago acima do anunciado) é
  concessionária de energia, caso previsto na ressalva.
- `/padroes` em 1280 e 390 px sem rolagem horizontal nem erro no console.

**Pronto quando:** `/padroes` mostra os três padrões com os casos da base
local, e a página explica que o TCE não liga pagamento a contrato.
