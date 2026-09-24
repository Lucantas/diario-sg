# Como o Diário Oficial de São Gonçalo estrutura os atos

Achados de 7 edições reais (2024-03-15, 2026-03-31, 2026-06-30, 2026-08-31,
2026-09-16, 2026-09-17, 2026-09-18; 8 a 23 páginas cada), baixadas de
`https://do.pmsg.rj.gov.br/diario/AAAA_MM_DD.pdf` e extraídas com
`pdftotext` (poppler 26.05). Reproduza com `./scripts/fetch-editions.sh`.
Os textos ficam em `services/api/testdata/editions/` e não são versionados
(contêm nomes de pessoas físicas).

## O site

- Home (`/`) tem dois formulários POST. "Busca Rápida" (`BuscaRapida=AAAA-MM-DD`)
  responde com um `<script>window.open('diario/AAAA_MM_DD.pdf','_blank')</script>`
  quando existe edição e com nada quando não existe. "Busca Específica"
  (`DataInicial`, `DataFinal`, `Termo`, `PesquisarTermo=Pesquisar`, action `index`)
  exige um termo e devolve um card por edição com o link
  `href="diario/AAAA_MM_DD.pdf"` e o texto `DD/MM/AAAA`; 5 por página, paginação
  via `GET index?NumeroPagina=N&Termo=...&DataInicial=...&DataFinal=...&PesquisarTermo=Pesquisar`.
- A URL do PDF é determinística por data. `HEAD` responde 200 com
  `Content-Length` quando há edição e **500** (não 404) quando não há
  (sábados, domingos, feriados).
- Não há número de edição na URL nem na listagem; ele só aparece dentro do PDF
  (`EDIÇÃO N°1.771`).
- `https://servicos.pmsg.rj.gov.br/diario_oficial.php` apenas redireciona para o
  portal WordPress `www.pmsg.rj.gov.br`, que linka para `do.pmsg.rj.gov.br`.
  O certificado desse host não envia a cadeia completa (curl falha sem `-k`).
- Nada exige JavaScript. Sem `robots.txt` restritivo observado; sem bloqueio ao
  User-Agent identificado.

## pdftotext: `-layout` quebra as colunas

O miolo do Diário é diagramado em **duas colunas**. Com `-layout` (o padrão do
adapter original) as duas colunas saem lado a lado na mesma linha, e um ato à
esquerda é misturado com outro à direita — títulos como
`DECRETO Nº 441/2026 3.3.90.31.00 127 1.501.0000.0000 0,00 3.000,00`.
Sem `-layout` (modo de leitura) o poppler reconstrói a ordem das colunas e
cada ato sai inteiro. O adapter passou a usar o modo de leitura.

Efeitos colaterais do modo de leitura:

- Linhas justificadas com espaçamento largo saem **uma palavra por linha**
  (`TERMO\nDE\nAPREENSÃO\nADMINISTRATIVA\nNº\n202/SEMMATRAN/...`,
  `ASSOCIAÇÃO\nDO\nFUNDO\nDE`). Acontece com títulos curtos centralizados e
  com texto justificado em coluna estreita.
- Tabelas (anexos de decreto orçamentário, atas de registro de preços, listas
  de nomeação) saem célula por linha, fora de ordem: muitas linhas só com
  dígitos (809 nas 7 edições), `0,00`, códigos `3.3.90.39.00`.
- Cabeçalho de página vira duas linhas fixas: `DIÁRIO OFICIAL ELETRÔNICO DO
  MUNICÍPIO DE SÃO GONÇALO D.O.E. | PODER EXECUTIVO | ANO VII | Nº 1.771 EM 18
  DE SETEMBRO DE 2026` e, no rodapé, o número da página seguido de
  `https://do.pmsg.rj.gov.br/`. Em 2024 o cabeçalho usava `N.º` em vez de `Nº`.
- Anexos com página própria (fotos de animais apreendidos, quadros) trazem
  um cabeçalho diferente (`ESTADO DO RIO DE JANEIRO / PREFEITURA MUNICIPAL DE
  SÃO GONÇALO / SECRETARIA ...` e linhas de `_____`).
- Ocasionalmente um cabeçalho de ato **não sai** no texto (em 2024-03-15 o
  `DECRETO N.º 104/2024` está ausente; a ementa vem colada ao anexo do decreto
  anterior). Sem OCR não há como recuperar.

## Estrutura de uma edição

1. **Capa**: 1 a 2 páginas de notícias institucionais (manchetes em caixa alta,
   texto corrido), sem atos.
2. **Expediente**: lista `SECRETARIA MUNICIPAL DE ... / NOME DO SECRETÁRIO`
   (todos os titulares, em toda edição). Não é ato; se indexado, todo nome de
   secretário casaria com todas as edições.
3. **`ATOS DO PREFEITO`**: decretos e portarias assinados pelo prefeito.
4. **Seções por órgão**, cada uma aberta por uma linha só com a **sigla** em
   caixa alta: `SEMAD`, `SEMED`, `SEMCI`, `SMTC`, `SEMAS`, `SEOP`, `FMS`,
   `FUNASG`, `CONGES`, `SEMMATRAN`, `SEMEL`, `SEMCOM`, `SEMCON`, `SEMPAD`,
   `SEMFA`, `SEMTRAN`, `FAELSG`, `CMAS`, `CMDCA`. Às vezes a sigla vem seguida
   do nome por extenso (`SMTC\nSECRETARIA MUNICIPAL DE TURISMO E CULTURA`).
5. **`Continuação do D.O.E. em DD/MM/AAAA`**: anexo de 1 página, coluna única,
   rodapé `D.O.E. - DD/MM/AAAA   1/1`, com as **portarias de pessoal** em
   formato abreviado. Frequentemente começa no meio de uma portaria (o
   cabeçalho ficou fora do PDF) e termina com um `Port. nº N/AAAA` sem corpo.

## Cabeçalhos de ato (linha própria, caixa alta)

| Família | Formas vistas | Ocorrências (7 ed.) |
| --- | --- | --- |
| Despacho | `DESPACHO`, `DESPACHO DO SECRETÁRIO`, `DESPACHO DA PRESIDENTE` | 128 |
| Portaria | `PORTARIA Nº 1490/2026`, `PORTARIA - SEI Nº 1004/SEMAD/SUBRH/CIF/2026`, `PORTARIA – SEI N° ...`, `PORTARIA N. º 12/2026`, `PORTARIA Nº 45/FMS/2026`, `PORTARIA FUNASG N° 049/2026`, `PORTARIA Nº 022 /CONGES/FUNASG/2026`, `PORTARIA Nº 3/2026/SEMPAD/PMSG` | 50 |
| Portaria abreviada (anexo de pessoal) | `Port. nº 1497/2026`, `Port.nº`, `Port nº` | 36 |
| Extrato | `EXTRATO DA ATA DE REGISTRO DE PREÇOS Nº 004/SEMEL/2026`, `EXTRATO DE CONTRATO 015/SEMPAD/2026`, `EXTRATO DO CONTRATO DE COMODATO DE IMÓVEL`, `EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO\nCONTRATO DE LOCAÇÃO 006/2020.`, `EXTRATO DE INEXIGIBILIDADE DE LICITAÇÃO`, `EXTRATO DA HOMOLOGAÇÃO - PREGÃO ELETRÔNICO – SRP FMS Nº 9.2026` | 47 |
| Decreto | `DECRETO Nº 441/2026`, `DECRETO N.º 102/2024`, `DECRETO N. º 12/2026` | 17 |
| Edital | `EDITAL DE CONVOCAÇÃO 05/SUBRH/SEMAD/2026`, `EDITAL DE CONVOCAÇÃO`, `EDITAL DE CONVOCAÇÃO N.º 03/2026/FUNASG` | 20 |
| Resolução | `RESOLUÇÃO Nº 071/SEMMATRAN/2026`, `RESOLUÇÃO Nº 004/2026-SEOP`, `RESOLUÇÃO CMDCA Nº 004, DE 12 DE MARÇO DE 2026.` | 9 |
| Aviso | `AVISO DE LICITAÇÃO SRP`, `AVISO DE LICITAÇÃO`, `AVISO DE DISPENSA ELETRÔNICA N° 19/2026` | 5 |
| Ata | `ATA DA REUNIÃO ORDINÁRIA Nº 3`, `ATA DA SESSÃO PÚBLICA DE RECEBIMENTO DOS ENVELOPES E`, `ATA DA 2ª AUDIÊNCIA PÚBLICA ...` | 5 |
| Termo | `TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS SEM\nRESSALVA`, `TERMO DE COOPERAÇÃO TÉCNICA 01/SEMCON/2026.`, `TERMO DE APREENSÃO ADMINISTRATIVA Nº:` (frequentemente uma palavra por linha) | 14 |
| Outros | `CHAMAMENTO PÚBLICO Nº 04/2026`, `NOTIFICAÇÃO Nº 191/SEMMATRAN/MA/GAB/2026`, `CONCESSÃO DE LICENÇA`, `CONTRATO Nº 12/2026`, `PREGÃO ELETRÔNICO ...`, `CONVOCAÇÃO`, `RESULTADO PRELIMINAR`, `AUTORIZAÇÃO`, `LEI Nº. 1650/2026`, `DESIGNA OS FISCAIS DO CONTRATO ...` | — |

Observações:

- O número vem como `Nº`, `N°`, `N.º`, `N. º`, `Nº.`, `nº`, `no` (sem símbolo)
  ou nada. Formato `NNN/AAAA` ou `NNN/SIGLA/AAAA` ou `NNN/AAAA/SIGLA`.
- Cabeçalhos longos quebram em 2 linhas (`EXTRATO DO QUINTO TERMO ADITIVO DE
  PRORROGAÇÃO AO` + `CONTRATO DE LOCAÇÃO 006/2020.`).
- `ANEXO DECRETO Nº 441/2026` e `ANEXO I` **não** abrem ato novo: pertencem ao
  ato anterior.
- Palavras de cabeçalho também aparecem no meio do corpo, mas em caixa
  baixa/mista (`Portaria nº 240/SUBRH/SEMAD/2018`, `Decreto nº 151`), o que
  permite exigir caixa alta no início da linha.

### A portaria abreviada vem no fim do ato, não no início

No anexo de pessoal a ordem é **verbo → corpo → número**:

```
Nomeia:
a contar de 18 de setembro de 2026, FULANA ... para exercer o
cargo em comissão de ...
Port. nº 1499/2026
Exonera a pedido:
a contar de 17 de setembro de 2026, ...
Port. nº 1500/2026
```

Três evidências, conferidas em 2020, 2021 e 2026 (inclusive com `-layout`):
o primeiro ato do bloco vem logo após `GABINETE DO PREFEITO`/`ATOS DO
PREFEITO`/`Continuação do D.O.E.` sem número antes dele; o último `Port. nº`
do bloco é seguido só do rodapé da página; e `Continuação do D.O.E. em
DD/MM/AAAA` está sempre no topo de uma página, seguido de um verbo.
Tratar `Port. nº` como cabeçalho (como a primeira versão do parser fazia)
atribuía a cada número o corpo da portaria **seguinte** e deixava um ato só
com título no fim de cada bloco.

### Notificações da Defesa Civil não têm título

As notificações de interdição da COMDEC não têm cabeçalho: cada uma começa
com `A COORDENADORIA MUNICIPAL DE DEFESA CIVIL DE SÃO GONÇALO, de acordo
com sua competência legal, ... vem pelo presente NOTIFICAR`, e vêm em blocos
de dezenas, uma por imóvel. Sem cabeçalho, o parser as colava no ato
anterior: em setembro de 2026, 1.099 notificações estavam dentro de 131 atos
alheios (extratos de dispensa, corrigendas, editais). Um exemplo: o extrato de
ratificação de dispensa da edição 1233.

Essa frase de abertura passou a iniciar um ato com o título `NOTIFICAÇÃO DA
DEFESA CIVIL` e o órgão COMDEC, e o tipo fica `outro`. A sigla `COMDEC` antes
do bloco nem sempre é seguida da primeira notificação, porque o
`pdftotext` põe antes dela a assinatura da coluna ao lado. Por isso algumas
linhas soltas (`Subsecretário Municipal de Defesa Civil`, `Mat. …`) ainda
podem ficar no fim do ato anterior.

### Títulos sem palavra de cabeçalho clássica

- `AUTO DE INFRAÇÃO` (Fazenda, Meio Ambiente) e `DESIGNAÇÃO DE FISCAL` /
  `DESIGNAÇÃO DE FISCAIS` (Transportes) abrem atos. Antes ficavam dentro do
  ato anterior: na edição 1257, a designação de fiscais do contrato da LM
  Cursos ficou dentro de uma prestação de contas da SEMED, e somou o valor
  errado ao CNPJ da empresa. `DESIGNAÇÃO` sozinha na linha não abre ato,
  porque aparece quebrada no meio de títulos de portaria ("DISPÕE SOBRE A /
  DESIGNAÇÃO / DO GESTOR…").
- A sigla do órgão procura o próximo texto sem limite de linhas em branco.
  Quando a sigla fica no pé da página, como `SEMED` na edição 1257, o
  rodapé e o cabeçalho removidos deixam três linhas em branco até o ato. O
  limite antigo, de duas linhas, perdia a seção.

### Edições até abril de 2021

- Não existe `ATOS DO PREFEITO`; o anexo de pessoal começa logo após
  `GABINETE DO PREFEITO`, sem capa de notícias.
- O número da edição não aparece no texto extraído (`EDIÇÃO N°` só existe a
  partir de 08/04/2021), então `edition_number` fica vazio nessas edições.
- O resto (verbos, `Port. nº`, siglas de órgão, extratos) é igual ao formato
  atual.

### Edições de 2010 a 2013

- O bloco de pessoal abre com `GABINETE DA PREFEITA` (e não `DO
  PREFEITO`), que não está em `preambleMarkers`; o corte do preâmbulo cai
  no primeiro início de ato. Não conferi se isso perde algum texto antes do
  primeiro ato nessas edições.
- As portarias abreviadas do gabinete vêm sem sigla de órgão antes delas.
- Aparecem cabeçalhos de tabela sozinhos na linha (`SUPERVISOR`, `CARGO`,
  `NOME`, `DESPESA`, `TOTAL`), que casam com o formato de sigla.

## Como aparecem nomes, cargos, matrículas

- **Portaria do prefeito (formal)**: `PORTARIA Nº 1490/2026` / `O PREFEITO
  MUNICIPAL DE SÃO GONÇALO, no uso das atribuições...` / `RESOLVE:` /
  `Exonerar a pedido, a contar de 16 de setembro de 2026, NOME EM CAIXA ALTA –
  Mat.: 70432, do cargo em comissão Chefe de Departamento de Esportes - Símbolo
  CC-1, da (o) Fundação de ...`.
- **Portaria abreviada (anexo)**: `Port. nº 1498/2026` / `Nomeia:` /
  `a contar de 18 de setembro de 2026, NOME EM CAIXA ALTA - CPF: 943.***.***-20,
  para exercer o cargo em comissão de ... - Símbolo CC-1, na(o) Secretaria ...`.
  Verbos vistos após `Port. nº`: `Nomeia:` (18), `Exonera:` (8), `Exonera a
  pedido:` (1), `Torna sem efeito:` (2), `Designa`. Nomeações coletivas vêm em
  tabela `MAT. / NOME / CARGO / SIMB.` (célula por linha).
- **CPF sempre mascarado** (`153.***.***-60`); **matrícula** como `Mat.: 131360`,
  `MAT.: 24886`, `matrícula 22.685`, `matrícula nº 131.252`.
- Portarias da SEMAD (readaptação, averbação, licença) escrevem o nome em caixa
  mista no meio da frase: `Readaptar, pelo período de 01 (um) ano, Nome Sobrenome,
  matrícula 22.685, ocupante do cargo efetivo Professor Docente II`.
- **Assinatura** fecha quase todo ato: `São Gonçalo, 17 de setembro de 2026.` /
  `NOME DO SIGNATÁRIO` / `Cargo`. Serve como fim de ato quando o próximo
  cabeçalho não é reconhecido.
- Nomes de secretários repetem-se em todo ato assinado (assinatura) e no
  expediente.

## CNPJ, valores, contratos, processos

| Campo | Formatos (com frequência nas 7 edições) | Regularidade |
| --- | --- | --- |
| CNPJ | `CNPJ: 12.345.678/0001-90` (20), `CNPJ n° ...` (7), `CNPJ nº ...` (3), `CNPJ/CPF: ...` (3), `CNPJ ...` (2), `inscrita sob o CNPJ n° ...`. Variações: `12.345.678/0001- 90` (espaço), `12.345678/0001-90` (ponto faltando), `-901` (dígito a mais por erro). 57 ocorrências no formato padrão. | Alta: regex funciona. Não há CNPJ sem pontuação. |
| Valor | `R$ 59.571,78` (dominante), `R$59,00` (sem espaço), `R$ 1` (cabeçalho `VALOR (R$ 1)` de tabela: **não** é valor), valores por extenso entre parênteses. Em atas de registro de preços há dezenas de valores unitários por ato. | Alta para o número; a **semântica** (mensal, global, unitário, por exercício) só está no rótulo anterior (`VALOR MENSAL:`, `VALOR GLOBAL:`). |
| Contrato | `CONTRATO N° 12/2026`, `CONTRATO Nº 12/2026.`, `Contrato SEMCOM Nº 07/2023`, `CONTRATO 015/SEMPAD/2026`, `CONTRATO DE LOCAÇÃO 006/2020`, `TERMO ADITIVO ... AO CONTRATO ...` | Média: sigla opcional no meio; número pode não ter `Nº`. |
| Processo | `Processo SEI! nº 03.05511/2026-8` (97), `PROCESSO nº: 1613/2026` (16), `Processo nº 9841/2026` (15), `Processo Administrativo nº 8.189/2025` (13), `PROCESSO SEI: 03.01508/2026-9` (11), `Processo no 9841/2026`, `Processo: 1613/2026`, `PROCEDIMENTO ADMINISTRATIVO Nº 9720/2026`, `SEI53.01067/2026-4` (sem espaço, em tabela) | Alta para os dois formatos (`NN.NNNNN/AAAA-D` do SEI e `NNNN/AAAA` legado). |

## Decisões tomadas no parser (ver `services/api/internal/adapters/parser`)

1. Extrair sem `-layout`.
2. Remover ruído de página (cabeçalho/rodapé, número de página seguido da URL).
3. Descartar capa e expediente: tudo antes de `ATOS DO PREFEITO` (até 2020,
   `GABINETE DO PREFEITO`; na falta dos dois, do primeiro início de ato).
4. Linhas só com sigla de órgão em caixa alta abrem seção: encerram o ato
   anterior e não entram no próximo.
5. Juntar sequências de linhas de uma palavra em caixa alta que começam com uma
   palavra de cabeçalho (`TERMO` `DE` `APREENSÃO` ...) antes de casar os regex.
6. Classificar pelo cabeçalho e, para portarias, pelo verbo nas primeiras
   linhas do corpo (`Nomeia`/`Nomear` → nomeação; `Exonera`/`Exonerar` →
   exoneração; `Torna sem efeito` continua portaria).
7. Tipos novos no domínio porque a taxonomia original não cobria o que existe
   de fato: `despacho`, `resolucao`, `edital`, `ata`.
8. `Port. nº N/AAAA` **fecha** o segmento corrente e vira o título dele; o
   corpo é o texto desde o verbo (`Exonera:`, `Nomeia:`, `Designa`...). Um
   `Port. nº` sem nada antes vira ato só com o número, para não perdê-lo.
9. `Continuação do D.O.E.` é fronteira de seção, como a sigla de órgão: encerra
   o ato anterior e não entra em nenhum ato.

Os números do parser (atos por edição, % em `outro`, erros conhecidos) estão em
`docs/fase-1-relatorio.md`.

## Diário Oficial Eletrônico da Câmara

Amostra: 88 edições de 2020-11 a 2026-09, baixadas de
`https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/AAAA-MM-DD.pdf`.

- **URL por data.** 404 nos dias sem edição; o servidor aceita `HEAD`. A
  primeira edição achada assim é de 2020-10-04. As edições de 2018 a 2020
  ("Ano-01") existem, mas com nomes de arquivo que não seguem a data, e a
  busca do site fica atrás de um WAF que recusa robôs.
- **PDF vazio.** Alguns dias devolvem um PDF de 696 bytes sem texto
  (2025-02-24, por exemplo). Vira uma edição com zero atos.
- **Cabeçalho de toda página**, sempre antes do primeiro texto da página:
  `PODER LEGISLATIVO`, `CÂMARA MUNICIPAL DE SÃO GONÇALO`,
  `São Gonçalo, 3 de novembro de 2025`, `Ano-08 / Edição – 138` (também
  `Edição - 138` e `Edição 138`), `DIÁRIO OFICIAL ELETRÔNICO – D.O.E`
  (com ou sem travessão), `LEI MUNICIPAL 855/2018 DE 05/07/2018` (com ou
  sem ponto) e uma linha de sublinhados.
- **Rodapé:** `Página N de M`, que o `pdftotext` às vezes põe no topo da
  página seguinte.
- **Por isso o cabeçalho só sai do topo da página.** `São Gonçalo, <data>`
  e `CÂMARA MUNICIPAL DE SÃO GONÇALO` também aparecem dentro dos atos, na
  assinatura ("GABINETE DO PRESIDENTE DA CÂMARA MUNICIPAL DE / SÃO
  GONÇALO", "São Gonçalo, 30 de outubro de 2025."). Na amostra, 538 linhas
  com cara de cabeçalho estavam no meio de atos. O parser tira as linhas
  de cabeçalho enquanto a página ainda não teve texto, e tira
  `Página N de M` e `Ano-NN / Edição` em qualquer lugar.
- **Número da edição** recomeça a cada ano (`Ano-08 / Edição 138`); o
  parser guarda só o número, e a data diferencia.
- **Atos.** Os cabeçalhos são os mesmos da Prefeitura: `PORTARIA Nº`,
  `RESOLUÇÃO Nº` (títulos de cidadania, moções, regimento), `EXTRATO`,
  `AVISO DE LICITAÇÃO`, `TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS` (cota
  parlamentar, CEAPM, um por vereador e mês), `DESPACHO`, `CORRIGENDA`.
  `TERMO DE HOMOLOGAÇÃO` e `TERMO DE ADJUDICAÇÃO` passaram a ser licitação
  nas duas fontes.
- **Lei promulgada pela Câmara** vem com o título duas vezes: `LEI Nº 1219
  DE 28 DE DEZEMBRO DE 2020`, a cláusula "PROMULGO A SEGUINTE LEI:" e o
  título de novo, seguido da ementa. O parser da Câmara junta o título
  repetido ao ato quando o que veio antes é curto e termina em dois-pontos.
  Sem essa condição, os termos de prestação de contas de vereadores
  diferentes, que têm o mesmo título, virariam um ato só.
- **PDF escaneado.** Muitas edições têm só o cabeçalho em texto e o corpo
  em imagem (cerca de 170 caracteres por página). Ficam com zero atos,
  porque o parser não faz OCR.
- **Backfill local (2020-10-04 a 2026-09-23):**
  - 984 edições, 4.824 atos e 1.896 ligações de entidades; nenhuma falha
    de coleta;
  - 78 edições (8%) ficaram sem atos: 5 PDFs em branco de 696 bytes e 73
    com o corpo escaneado;
  - por tipo: resolução 2.732, nomeação 370, prestação de contas 367,
    exoneração 311, portaria 167, licitação 144, despacho 143, aditivo 126,
    contrato 110, corrigenda 92, outro 84, edital 82, dispensa 56, lei 21,
    ata 15, decreto 4;
  - um ato (extrato de 2020-10-28) ficou com o cabeçalho partido no meio da
    página ("DIÁRIO OFICIAL ELETRÔNICO" e "D.O.E" em linhas separadas). O
    parser só tira o cabeçalho do topo da página.
- **Sem órgão.** Os atos da Câmara ficam sem órgão; o filtro de órgão é
  das secretarias da Prefeitura.
- **Duas colunas.** Os termos de prestação de contas da cota parlamentar
  (e alguns extratos) vêm em duas colunas lado a lado. Aqui o modo de
  leitura do `pdftotext` falha, ao contrário do que acontece na Prefeitura:
  intercala as colunas linha a linha (título A, título B, vereador A,
  vereador B, processo A, processo B…). O resultado era um ato com dois ou
  três vereadores e outro só com o título: 57 atos misturados nas 984
  edições locais. Em algumas edições as colunas ainda vêm defasadas meia
  linha.
- **Por isso, na Câmara, o texto sai também com `-layout`.** Uma página
  conta como de duas colunas quando várias linhas têm texto começando na
  mesma posição depois de um vão de dois espaços ou mais, e no máximo 25%
  das linhas com texto atravessam esse vão. Numa página assim, a coluna
  da esquerda vem antes da da direita, e uma linha que atravessa o vão
  descarrega as duas colunas antes de entrar. As outras páginas ficam com
  o texto do modo de leitura. Nas 984 edições: 329 mudaram de texto,
  os atos misturados foram de 57 para 0, e o total de atos foi de 4.849
  para 4.853. As diferenças conferidas foram todas para melhor:
  - resoluções que estavam coladas em outras voltaram a ser atos próprios
    (2020-11-04);
  - a cláusula "RATIFICAÇÃO: Ficam mantidas…" deixou de virar um ato de
    dispensa falso (2024-01-29).
