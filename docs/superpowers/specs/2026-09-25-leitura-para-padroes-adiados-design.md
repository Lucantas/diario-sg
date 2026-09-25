# Leitura dos textos para os três padrões adiados (Entrega 2)

Data: 2026-09-25. Fecha os três padrões que ficaram em aberto em
`docs/superpowers/specs/2026-09-24-padroes-para-verificar-design.md`:
aditivos acima de 25%, contratação emergencial renovada e dispensa por
valor de obras (inciso I) no fracionamento. Linguagem sempre "padrão para
verificar", nunca "irregularidade".

## Fatos que orientam o desenho

Medidos na base local em 25/09/2026:

- **Regex no corpo, na hora da consulta, não serve.** Um `body ~* …` com
  as citações de artigo e inciso sobre os atos da Prefeitura leva cerca de
  2 minutos no Postgres. As consultas dos padrões atuais só funcionam
  porque filtram por tipo antes e aplicam a regra de texto em Go. Para os
  padrões novos, os candidatos não se acham pelo tipo: é preciso ler o
  texto na indexação e guardar o fato no ato, como já acontece com a
  modalidade e o valor principal (migrations 010 e 011).
- **Aditivos.** Dos 4.571 aditivos da Prefeitura, 161 falam em acréscimo.
  O valor em reais do acréscimo não serve para calcular o percentual: muitas
  vezes é o novo total do contrato (o 3/SEMDUR/2024 daria 101%). O
  percentual declarado no texto ("acréscimo … que equivale a 24,88% do
  valor contratado") serve, com três cuidados:
  - reajuste não conta no limite (art. 65, § 8º, da Lei 8.666) e o texto do
    reajuste traz o percentual acumulado: os aditivos do contrato
    001/SEMCON/2022 dizem "reajuste de 26,92%" e depois "acréscimo de
    26,92% do valor original", no aditivo "COM REAJUSTE";
  - a retificação ("Onde se lê") repete o aditivo;
  - o mesmo aditivo sai mais de uma vez; o ordinal ("SEGUNDO TERMO
    ADITIVO") diz se é o mesmo.
  Com isso, 75 aditivos têm percentual de acréscimo declarado, 26 contratos
  ligados. O contrato FMS 007/2015 (compra de equipamentos) teve acréscimo
  de 46,37%, acima de 25%; o 019/2016 da SEMED (reforma de escola) teve
  49,13%, abaixo do limite de 50% para reforma.
- **Emergencial.** 111 atos da Prefeitura citam o art. 24, IV, da Lei
  8.666 ou o art. 75, VIII, da Lei 14.133. Só 27 trazem o CNPJ de um
  fornecedor; os extratos de contrato da PGM, por exemplo, dizem "Partes:
  Procuradoria Geral do Município de São Gonçalo X Lógica Tecnologia Ltda",
  sem CNPJ. As ratificações não dizem o fornecedor, mas trazem o processo,
  que o extrato do contrato também traz. Lendo o nome do fornecedor e
  juntando os atos pelo processo, aparecem 10 sequências de contratação
  emergencial da mesma empresa no mesmo órgão, por exemplo a Lógica
  Tecnologia na PGM em janeiro de 2021, agosto de 2021, janeiro de 2022 e
  julho de 2022.
- **A mesma empresa com duas chaves.** O Centro Fluminense de
  Oxigenoterapia Hiperbárica aparece pelo CNPJ em uns atos e só pelo nome em
  outros.
- **Obras.** O fracionamento atual só olha o inciso II. O inciso I tem
  limite próprio, conferido em 25/09/2026:
  - Lei 8.666, art. 24, I: R$ 15.000,00 até 18/07/2018; R$ 33.000,00 a
    partir de 19/07/2018 (Decreto 9.412/2018);
  - Lei 14.133, art. 75, I: R$ 100.000,00 (2021 e 2022), R$ 114.416,65
    (2023, Decreto 11.317/2022), R$ 119.812,02 (2024, Decreto 11.871/2023),
    R$ 125.451,15 (2025, Decreto 12.343/2024), R$ 130.984,20 (2026,
    Decreto 12.807/2025).

## Decisões

1. **Dois fatos novos por ato, lidos na indexação** (migration 013, sem
   editar as anteriores):
   - `acts.legal_basis text[]`: os incisos de dispensa citados, entre
     `art24:I`, `art24:II`, `art24:IV`, `art75:I`, `art75:II` e
     `art75:VIII`, nas duas ordens ("art. 24, inciso IV" e "inciso IV do
     art. 24");
   - `acts.declared_increase_bp integer`: o acréscimo declarado num aditivo,
     em centésimos de ponto percentual (46,37% = 4637), zero quando não há,
     quando o percentual é de reajuste, quando é o limite da lei ("até
     25%") e quando o ato é retificação.
   O `make reindex` preenche as colunas das edições já indexadas.
   `CitesValueDispensa` passa a usar a mesma leitura de incisos.
2. **Nome do fornecedor lido na consulta**, só dos atos candidatos (não
   precisa de coluna): `SupplierNameOf(body)` lê "Partes: … X …", "em favor
   da empresa …", "Contratado:", "Favorecida:" e para no primeiro CNPJ,
   vírgula ou palavra de fim ("objeto", "valor", "para", "inscrita"…).
   `SupplierNameKey(name)` tira acentos, pontuação e sufixos societários
   (LTDA, EIRELI, ME, EPP, S/A…). Um nome que aparece antes de um CNPJ
   (`PartiesOf`) em algum ato candidato vira esse CNPJ.
3. **Aditivos acima do limite:** aditivos do mesmo contrato (número do
   contrato ligado em `entity_links` e sigla principal do órgão) somam os
   acréscimos declarados, contando cada ordinal uma vez (sem ordinal, cada
   percentual distinto uma vez). O limite é 25%, ou 50% quando algum aditivo
   do contrato fala em reforma (art. 65, § 1º, da Lei 8.666; art. 125 da
   Lei 14.133). Aciona quando a soma passa do limite.
4. **Emergencial renovada:** os atos com `art24:IV` ou `art75:VIII` se
   juntam em contratações por processo ou contrato dentro do órgão. O
   fornecedor da contratação é o CNPJ ou o nome dos atos dela (só um; com
   dois, fica de fora). Aciona quando a mesma empresa tem duas ou mais
   contratações emergenciais no mesmo órgão, cada uma começando entre 30
   dias e 24 meses depois da anterior (contratações no mesmo mês, para
   objetos diferentes, não são renovação).
5. **Obras no fracionamento:** a contratação por dispensa de valor passa a
   ter categoria, obras (inciso I) ou compras e serviços (inciso II), cada
   uma com seu limite; o caso soma dentro da mesma categoria. O título do
   caso diz qual.
6. Os dois padrões novos entram em `PatternCatalog`, em `/v1/patterns` e na
   página `/padroes`, que já mostra qualquer padrão do catálogo.

## Ajustes da revisão

A revisão independente achou, com casos reais da base:

- nomes de fornecedor lidos de dentro do nome da secretaria ("Secretaria
  Municipal de Turismo e Cultura de São Gonçalo/RJ" virava a empresa
  "Cultura de São Gonçalo/RJ") e de texto solto ("menos favorecidos
  economicamente"): o " e " que fica dentro do nome de um órgão é pulado,
  "favorecido" só conta com dois-pontos e nome com palavra em minúscula
  (fora de, da, do, e) é recusado;
- acréscimo que pegava o percentual de decréscimo ou de desconto: o aditivo
  do contrato 05/2016 da FMS lista "46,61% … de acréscimo" e "15,77% … de
  acréscimo" entre dois decréscimos. Agora a forma "N% de acréscimo" soma
  todos os itens, e a forma "acréscimo … N%" é recusada quando há
  decréscimo, supressão, desconto ou redução no meio. Decimal com ponto
  ("1.72 %") passa a ser lido;
- inciso de outra lei ("art. 24, I" da LDB, do Código de Trânsito) e
  citação de proibição ("vedada a recontratação … com base no inciso VIII
  do art. 75", "na hipótese de contratação direta fundamentada no art. 75,
  VIII"): a citação só conta com a Lei 8.666, a 14.133 ou "Lei de
  Licitações" por perto e sem "vedad", "hipótese" ou "recontrata" logo
  antes; a emergencial só olha dispensa, contrato, licitação e "outro";
- número de processo ou contrato repetido em órgãos diferentes: a junção é
  dentro do órgão; o ato sem órgão (ratificação do prefeito) entra pelo
  processo quando o número só aparece num órgão; caso sem órgão fica de
  fora nos dois padrões;
- o mesmo aditivo publicado com e sem ordinal contava duas vezes;
- "reforma" em outro ato colado no mesmo corpo subia o limite para 50%:
  agora só no título e no começo do texto;
- nome ligado a mais de um CNPJ (matriz e filial) não vira alias.
- o mesmo aditivo republicado dias depois saía em outra seção do Diário,
  com outro órgão (o 05/2016 da KF Engenharia sob a SEMCOMP e depois sob
  a SEMDUR): aditivo com o mesmo número de contrato, o mesmo ordinal e o
  mesmo percentual até 30 dias depois de outro conta como o mesmo.

Resultado na base local depois do reindex: fracionamento, 1 caso (compras;
nenhum de obras); aditivos acima do limite, 2 casos; emergencial renovada,
9 casos. Tipos e órgãos dos atos não mudaram com o reindex.

## Fora do escopo

- Incisos citados juntos ("art. 24, incisos II e IV", "art. 24, I e II"):
  só o primeiro é lido.
- Aditivo que só declara o valor em reais, sem percentual.
- Guardar o nome do fornecedor no banco ou mostrá-lo nos painéis (a Entrega
  3 traz o nome da Receita).
- Diário da Câmara.

## Testes

- Domínio, com trechos reais:
  - `LegalBasisOf`: art. 24, inciso IV; "inc. IV"; "inciso VIII do art.
    75"; art. 24, XIII não entra; os casos atuais de `CitesValueDispensa`
    continuam passando;
  - `DeclaredIncreaseBasisPoints`: 24,88%, "acréscimo equivalente a
    46,37%", "em 8,63% … do valor global" (depois de "Lei 8.666/93"),
    reajuste de 12,06% = 0, "COM REAJUSTE … acréscimo de 26,92%" = 0,
    retificação = 0, "até 25%" = 0;
  - `SupplierNameOf` e `SupplierNameKey`: PGM "X Lógica Tecnologia Ltda",
    "LOGICA TECNOLOGIA EIRELI" com a mesma chave, "em favor da empresa …",
    "Partes: Fundação de Artes, Esporte e Lazer … e a empresa …";
  - `FindExcessiveAddenda`: FMS 007/2015 aciona; SEMED 019/2016 (reforma)
    não; ordinal repetido conta uma vez; dois aditivos de 15% somam 30%;
  - `FindRenewedEmergencies`: Lógica Tecnologia na PGM aciona com a
    ratificação juntada ao extrato pelo processo; DBNOVA com cinco
    contratações no mesmo dia na SEMAS não aciona; nome e CNPJ da mesma
    empresa se juntam;
  - `DispensaLimitCents` de obras nas bordas; fracionamento de obras aciona
    e obras não somam com compras.
- Integração: indexação grava `legal_basis` e `declared_increase_bp`;
  `/v1/patterns` traz os quatro padrões com os casos dos textos reais.
- Base local: `make reindex` de 2010 a 2026 e comparação de tipos e órgãos
  antes e depois (não devem mudar).
