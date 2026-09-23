# Nomes por extenso dos órgãos

`domain.organNames` (`services/api/internal/core/domain/organ.go`) tem as
125 siglas que o parser aceita como cabeçalho de órgão. Este arquivo diz
de onde veio o nome de cada uma. A busca e a API mostram o nome ao lado da
sigla; sigla sem nome aparece sozinha.

## Como os nomes foram escolhidos

Levantamento feito na base local (2010 a 2026-09, 153.750 atos) em
22/09/2026, procurando cada sigla no texto dos atos.

- **Evidência direta** (99 siglas): o Diário escreve a sigla e o nome lado
  a lado, como em "Secretaria Municipal de Educação (SEMED)".
- **Evidência indireta** (20 siglas entraram, 1 ficou de fora): a sigla é
  cabeçalho e o ato logo abaixo nomeia o órgão, sem os dois lado a lado.
  SMC ficou sem nome porque o Diário também usa a sigla para o Sistema
  Municipal de Cultura e para a Secretaria Municipal de Cultura.
- **Sem nome** (6 siglas): IPIIBA, SEMAIMPD, SEMIURME, SMAP, SMC e SSM.

Cada sigla tem um nome só, que vale para todos os anos. Quando o órgão
mudou de nome, a coluna de observação registra os outros nomes
encontrados.

## Siglas que talvez não sejam órgãos

- **IPIIBA**: "IPIIBA:" é o cemitério de Ipiíba nos editais de exumação;
  os 6 atos atribuídos (20/01/2021) são corrigendas de portarias.
- **SSM**: símbolo de cargo em comissão ("Símbolo SSM"); os 4 atos
  atribuídos (2013) são corrigendas.
- **PROMEA**: é um programa, não um órgão; os atos são da comissão dele.
- **PGE** e **GM**: usadas como cabeçalho da Procuradoria Geral do
  Município e da Guarda Municipal, mas no texto também significam
  Procuradoria Geral do Estado e prefixo de cargo.

Tirar uma sigla da lista não muda a segmentação (a linha continua
separando seção), mas os atos abaixo dela ficam sem órgão, e isso só
chega à base com `make reindex`. Fica para quando houver mais evidência.

## Variantes da mesma sigla

Erros de digitação ou grafias antigas que o Diário usou por pouco tempo.
Cada variante tem o nome do órgão principal. A busca por qualquer uma das
siglas traz os atos de todas, e a lista de órgãos (`/v1/organs`) mostra só
a principal, com os atos das variantes somados. O ato continua com a sigla
que saiu no Diário. A tabela vive em `domain/organ_variants.go`.

| Variante | Sigla principal |
| --- | --- |
| CODEPE | COPEDE |
| COMINSG | COMIRSG |
| CONDEC | COMDEC |
| FAESLG | FAELSG |
| FUNPARJ | FUMPARJ |
| SAMSADC, SEMSAD | SEMSADC |
| SEMIND | SEMIMD |
| SMPPE | SEMPPE |
| SEMGOVCON, SEMGOVCOMS | SEMGOVCOM |
| SEMHABITA | SECHABITA |
| SEMIDC | SEMINDECON |
| CGGMSG | CGMSG |
| FMSSG | FMS |
| SMAS | SEMAS |
| SMSP | SEMSEP |
| SECGOV | SEMGOV |

## Siglas

| Sigla | Nome | Evidência | Observação |
| --- | --- | --- | --- |
| CAISAN | Câmara Intersecretarias de Segurança Alimentar e Nutricional | 2026-07-02: "Câmara Intersecretarias de Segurança Alimentar e Nutricional (CAISAN)" | também "Câmara Intersetorial (Municipal) de Segurança Alimentar e Nutricional (CAISAN)" (2018–2026-04-13, Decreto 137/2022); "Câmara Interministerial" é a CAISAN federal |
| CEC | Comissão Especial de Contratação | 2025-07-25: cabeçalho CEC → "…pela Comissão Especial de Contratação, instituída por meio da Portaria Sei nº 42/SEMAD…" | evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado; a única expansão literal de CEC no Diário é "Comissão de Ética e Conduta (CEC)" (2021-12-21), outro órgão, fora de cabeçalho |
| CGGMSG | Corregedoria-Geral da Guarda Municipal de São Gonçalo | 2020-03-06: "Corregedor-Geral – CGGMSG"; 2020-03-19: "…/CGGMSG … CORREGEDORIA DA GUARDA MUNICIPAL DE SÃO GONÇALO" | sigla anterior de CGMSG (2019–2020) |
| CGMSG | Corregedoria-Geral da Guarda Municipal de São Gonçalo | 2026-03-12 (sob CGMSG): "O GABINETE DA CORREGEDORIA GERAL DA GUARDA MUNICIPAL DE SÃO GONÇALO" | antes CGGMSG |
| CMAS | Conselho Municipal de Assistência Social de São Gonçalo | 2026-09-04: "CONSELHO MUNICIPAL DE ASSISTÊNCIA SOCIAL DE SÃO GONÇALO – CMAS/SG" |  |
| CMC | Conselho Municipal de Cultura de São Gonçalo | 2019-09-19: "Conselho Municipal de Cultura de São Gonçalo – CMC/SG"; 2026-03-10: "Conselho Municipal de Cultura – CMC" | no Diário SMC também é "Sistema Municipal de Cultura – SMC" |
| CMDCA | Conselho Municipal dos Direitos da Criança e do Adolescente | 2026-08-25: "Conselho Municipal dos Direitos da Criança e do Adolescente – CMDCA/SG" |  |
| CMDDM | Conselho Municipal de Defesa dos Direitos das Mulheres de São Gonçalo | 2026-09-02: "Conselho Municipal de Defesa dos Direitos das Mulheres de São Gonçalo – CMDDM/SG" | no mesmo dia e desde 2023 também "…dos Direitos da Mulher de São Gonçalo – CMDDM-SG"; antes CMDM |
| CMDDMSG | Conselho Municipal de Defesa dos Direitos da Mulher de São Gonçalo | 2025-10-07: "CONSELHO MUNICIPAL DE DEFESA DOS DIREITOS DA MULHER DE SÃO GONÇALO – CMDDMSG" | variante de CMDDM |
| CMDM | Conselho Municipal dos Direitos da Mulher | 2020-02-04: "Conselho Municipal dos Direitos da Mulher – CMDM" | depois CMDDM (Defesa dos Direitos da Mulher); 2025-06-05 "Conselho Municipal de Defesa dos Direitos da Mulher – CMDM/SG" |
| CME | Conselho Municipal de Educação | 2026-09-14: "Conselho Municipal de Educação-CME/SG" |  |
| CMEL | Conselho Municipal de Esporte e Lazer de São Gonçalo | 2026-08-12: "CONSELHO MUNICIPAL DE ESPORTE E LAZER DE SÃO GONÇALO – CMEL/SG" |  |
| CMPA | Conselho Municipal de Proteção Animal | 2021-12-13: "Conselho Municipal de Proteção Animal – CMPA-SG" | 2012: "Conselho Municipal de Proteção aos Animais" |
| CMS | Conselho Municipal de Saúde | 2024-12-19: "CMS (Conselho Municipal de Saúde)"; 2013-12-06: "Conselho Municipal de Saúde (CMS)" |  |
| CODEPE | Conselho Municipal dos Direitos da Pessoa com Deficiência de São Gonçalo | 2014-11-18: cabeçalho CODEPE → "ATA DA REUNIÃO ORDINÁRIA DO CONSELHO MUNICIPAL DOS DIREITOS DA PESSOA COM DEFICIÊNCIA…" | provável erro de digitação de COPEDE (só em 2014-11-18); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| COMAD | Conselho Municipal Antidrogas de São Gonçalo | 2015-06-01: "Conselho Municipal Antidrogas de São Gonçalo - COMAD/SG" |  |
| COMDAR | Conselho Municipal de Desenvolvimento Agropecuário de São Gonçalo | 2019-04-25: "CONSELHO MUNICIPAL DE DESENVOLVIMENTO AGROPECUÁRIO DE SÃO GONÇALO – COMDAR" |  |
| COMDEC | Coordenadoria Municipal de Defesa Civil de São Gonçalo | 2025-03-18: "COMDEC A COORDENADORIA MUNICIPAL DE DEFESA CIVIL DE SÃO GONÇALO"; 2012-12-06: "…Defesa Civil - COMDEC" |  |
| COMDEPISG | Conselho Municipal de Defesa dos Direitos da Pessoa Idosa de São Gonçalo | 2026-07-29: "CONSELHO MUNICIPAL DE DEFESA DOS DIREITOS DA PESSOA IDOSA DE SÃO GONÇALO – COMDEPISG" | 2018: grafado "PESSOA IDOAS" (erro de digitação) |
| COMDESG | Conselho Municipal de Desenvolvimento Econômico de São Gonçalo | 2018-08-27: "Conselho Municipal de Desenvolvimento Econômico de São Gonçalo – COMDESG" |  |
| COMENQ | Comissão de Enquadramento | 2024-12-20: "…diretamente à Comissão de Enquadramento, conforme disposto no Edital 001/2024 - COMENQ" | comissão da SEMAD (Edital nº 002/SEMAD/COMENQ/2024) |
| COMINSG | Conselho Municipal de Defesa dos Direitos do Negro e Promoção da Igualdade Racial e Étnica em São Gonçalo | 2022-05-02: "EDITAL N.º 01/2022/COMINSG … IGUALDADE RACIAL E ÉTNICA NO MUNICÍPIO DE SÃO GONÇALO – COMIRSG" | erro de digitação de COMIRSG (só em 2022-05-02; o edital repetido traz "C0MIRSG") |
| COMIRSG | Conselho Municipal de Defesa dos Direitos do Negro e Promoção da Igualdade Racial e Étnica em São Gonçalo | 2026-08-21: "…Defesa dos Direitos do Negro e Promoção da Igualdade Racial e Étnica em São Gonçalo – COMIRSG" | variações: "…no Município de São Gonçalo", "dos Negros e Promoção da Igualdade Étnica" (2018–2021) |
| COMMADS | Conselho Municipal de Meio Ambiente e Desenvolvimento Sustentável | 2026-09-04: "Conselho Municipal de Meio Ambiente e Desenvolvimento Sustentável (COMMADS)" |  |
| COMPAD | Conselho Municipal de Políticas Públicas sobre Álcool e Drogas | 2026-07-22: "Conselho Municipal de Políticas Públicas sobre Álcool e Drogas (COMPAD)" | antes (2015–2024) "Conselho Municipal de Políticas sobre Álcool e Drogas" |
| COMSEA | Conselho Municipal de Segurança Alimentar e Nutricional de São Gonçalo | 2026-06-26: "Conselho Municipal de Segurança Alimentar e Nutricional de São Gonçalo – COMSEA/SG" |  |
| COMSEP | Conselho Municipal de Segurança Pública | 2025-12-12: "CONSELHO MUNICIPAL DE SEGURANÇA PÚBLICA – COMSEP" |  |
| CONCIDADES | Conselho Municipal da Cidade de São Gonçalo | 2026-09-10: "Conselho Municipal da Cidade de São Gonçalo – CONCIDADES-SG" | também "Conselho Municipal das Cidades de São Gonçalo" (2026-06-25) |
| CONDEC | Coordenadoria Municipal de Defesa Civil de São Gonçalo | 2010-10-07: cabeçalho CONDEC → "CONVOCAÇÃO A Coordenadoria Municipal de Defesa Civil de São Gonçalo…" | variante/erro de COMDEC (2010; "Ofício nº 004/CONDEC/2012"); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| CONGES | Conselho Gestor da Fundação Municipal de Assistência à Saúde dos Servidores de São Gonçalo | 2026-09-18: "PORTARIA Nº 022/CONGES/FUNASG/2026 … PRESIDENTE DO CONSELHO GESTOR (CONGES) da Fundação…" |  |
| CONSAD | Conselho de Administração do Instituto de Previdência do Município de São Gonçalo | 2025-02-20: "CONSELHO DE ADMINISTRAÇÃO DO INSTITUTO DE PREVIDÊNCIA DO MUNICÍPIO… REGIMENTO INTERNO DO CONSAD" | antes: "Conselho de Administração do IPASG (CONSAD)" (2013-09-23) |
| COPEDE | Conselho Municipal dos Direitos da Pessoa com Deficiência de São Gonçalo | 2026-07-20: "Conselho Municipal dos Direitos da Pessoa com Deficiência de São Gonçalo – COPEDE-SG" |  |
| CORIM | Comissão de Recursos de Infrações Municipais | 2026-01-08: "Comissão de Recursos de Infrações Municipais – CORIM" | 2023: "Conselho de Recursos de Infrações Municipais" |
| CPAC | Comissão Permanente de Acúmulos de Cargos | 2026-05-05: cabeçalho CPAC → "A COMISSÃO PERMANENTE DE ACÚMULOS DE CARGOS DA SECRETARIA MUNICIPAL DE ADMINISTRAÇÃO" | da SEMAD; também "NOTIFICAÇÃO Nº 001/CPAC/2023"; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| CPADAG | Comissão Permanente de Avaliação de Documentos do Arquivo Geral | 2025-04-09: cabeçalho CPADAG → "O Presidente da Comissão Permanente de Avaliação de Documentos do Arquivo Geral" | da SEMAD ("CPADAG/SEMAD", 2019); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| CPAPPF | Comissão Permanente de Análise para Progressão e Promoção Funcional | 2024-05-29 (ato sob CPAPPF): "Comissão Permanente de Análise para Progressão e Promoção Funcional" | da SEMAD (Edital nº 001/SEMAD/CPAPPF); os editais também dizem "Comissão Permanente de Progressão e Promoção Funcional" |
| CPIAD | Comissão Permanente de Inquérito Administrativo Disciplinar | 2026-08-20: "PORTARIA – 006/2026 - CPIAD A PRESIDENTE DA COMISSÃO PERMANENTE DE INQUÉRITO ADMINISTRATIVO DISCIP[LINAR]" |  |
| CPPF | Comissão Permanente de Promoção Funcional da Administração Direta | 2024-09-04 (ato sob CPPF): "Presidente da Comissão Permanente de Promoção Funcional da Administração Direta" | editais "Nº 001/2024/CPPF" de promoção dos servidores da administração direta |
| CPPFGMF | Comissão Permanente de Promoção Funcional dos Servidores Públicos da Guarda Municipal e Assistência à Saúde dos Servidores do Município de São Gonçalo | 2023-11-22 (sob CPPFGMF): "Comissão Permanente de Promoção Funcional dos Servidores Públicos da Guarda Municipal e…" | nome literal do Diário; abrange Guarda Municipal e FUNASG; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| CPPFS | Comissão Permanente de Promoção Funcional dos Servidores Públicos da Saúde do Município de São Gonçalo | 2024-12-30: "CPPFS A Comissão Permanente de Promoção Funcional dos Servidores Públicos da Saúde do Município…" |  |
| EDURSAN | Empresa Municipal de Desenvolvimento Urbano e Saneamento Ambiental | 2021-07-30: "Empresa Municipal de Desenvolvimento Urbano e Saneamento Ambiental - EDURSAN" | extinta (comissão liquidante 2016–2017) |
| FAELSG | Fundação de Artes, Esporte e Lazer de São Gonçalo | 2026-07-17: "FUNDAÇÃO DE ARTES, ESPORTE E LAZER DE SÃO GONÇALO – FAELSG" | sigla anterior: FAESG |
| FAESG | Fundação de Artes, Esporte e Lazer de São Gonçalo | 2024-04-08: "Fundação de Artes, Esporte e Lazer de São Gonçalo – FAESG" | sigla usada 2019–2023; depois FAELSG; antes FASG (Fundação de Artes de São Gonçalo) |
| FAESLG | Fundação de Artes, Esporte e Lazer de São Gonçalo | 2024-02-29: cabeçalho FAESLG → "PORTARIA N.º 002/2024 … CELEBRADO ENTRE A FUNDAÇÃO DE ARTES, E[SPORTE E LAZER]" | provável erro de digitação de FAELSG; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| FASG | Fundação de Artes de São Gonçalo | 2019-03-13: "Fundação de Artes de São Gonçalo – FASG" | também "Fundação de Artes São Gonçalo" (2010–2018); depois FAESG |
| FIASG | Fundação Municipal de Apoio à Educação e Assistência à Infância e à Adolescência de São Gonçalo | 2015-11-25: "FUNDAÇÃO MUNICIPAL DE APOIO A EDUCAÇÃO E ASSISTÊNCIA À INFÂNCIA E À ADOLESCÊNCIA DE SÃO GONÇALO – FIASG" | 2013–2014 também "Fundação Municipal de Apoio à Assistência à Infância e à Adolescência de São Gonçalo" |
| FMAS | Fundo Municipal de Assistência Social | 2026-05-13: "FMAS (Fundo Municipal de Assistência Social)" |  |
| FMS | Fundação Municipal de Saúde | 2026-02-23: "Fundação Municipal de Saúde (FMS)" | a mesma sigla aparece para "Fundo Municipal de Saúde" (2015–2022, poucos casos); atos sob o cabeçalho são da Fundação |
| FMSSG | Fundação Municipal de Saúde de São Gonçalo | 2012-12-20: cabeçalho FMSSG → "Contrato n° 33/FMS/2011 Partes: Fundação Municipal de Saúde de São Gonçalo…" | variante de FMS (também "PORTARIA N.º 033/FMSSG/2022"); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| FUMIA | Fundo Municipal para a Infância e Adolescência | 2026-05-20: "Fundo Municipal para a Infância e Adolescência (FUMIA)" | também "Fundo Municipal da Infância e Adolescência de São Gonçalo (FUMIA-SG)" (2026-06-03) |
| FUMPARJ | Fundação Municipal de Parques e Jardins | 2013-12-23: "FUNDAÇÃO MUNICIPAL DE PARQUES E JARDINS DO MUNICÍPIO DE SÃO GONÇALO – FUMPARJ" |  |
| FUNASG | Fundação Municipal de Assistência à Saúde dos Servidores de São Gonçalo | 2026-09-18: "Fundação Municipal de Assistência à Saúde dos Servidores de São Gonçalo – FUNASG" |  |
| FUNCULTURA | Fundo Municipal de Cultura | 2026-09-11: "FUNDO MUNICIPAL DE CULTURA – FUNCULTURA" |  |
| FUNDESG | Fundo de Gestão, Desenvolvimento e Modernização da Procuradoria Geral do Município de São Gonçalo | 2025-06-30: "FUNDO DE GESTÃO, DESENVOLVIMENTO E MODERNIZAÇÃO DA PROCURADORIA GERAL DO MUNICIPIO… - FUNDESG" |  |
| FUNPARJ | Fundação Municipal de Parques e Jardins | 2012-05-25: cabeçalho FUNPARJ → "FUNDAÇÃO MUNICIPAL DE PARQUES E JARDIM DE SÃO GONÇALO … Presidente da FUMPARJ" | provável erro de digitação de FUMPARJ; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| GM | Guarda Municipal de São Gonçalo | 2011-10-11: cabeçalho GM → "PORTARIA Nº 037/44.3.1/11 - GMSG O Comandante da Guarda Municipal de São Gonçalo" | variante de GMSG (2011–2017); "GM" também é prefixo de cargo de guarda e marca de veículo nos editais de leilão; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| GMSG | Guarda Municipal de São Gonçalo | 2023-11-06: "Guarda Municipal de São Gonçalo-GMSG"; 2022-02-10: "II - Guarda Municipal de São Gonçalo - GMSG" |  |
| IPASG | Instituto de Previdência dos Servidores Municipais de São Gonçalo | 2023-01-16: "Instituto de Previdência dos Servidores Municipais de São Gonçalo - IPASG" | até 2022: "Instituto de Previdência e Assistência dos Servidores Municipais de São Gonçalo" (2025-01-15 ainda citado assim); depois SG-PREVI ("Instituto de Previdência do Município de São Gonçalo") |
| IPIIBA |  |  | sigla parece falsa: "IPIIBA:" é o cemitério de Ipiíba nos editais de exumação da SEMAD; os 6 atos atribuídos (2021-01-20) são corrigendas de portarias |
| PGE | Procuradoria Geral do Município | 2010-05-19: cabeçalho PGE → "Resolução nº. 002/2010 PGM … O PROCURADOR GERAL DO MUNICÍPIO DE SÃO GONÇALO" | cabeçalho usado para a PGM em 2010–2014 (em tabela de 2012 "Procurador … PGE"); fora disso PGE é a Procuradoria Geral do Estado ("SEFAZ/PGE/DETRAN"); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| PGM | Procuradoria Geral do Município | 2026-09-01: "Procuradoria Geral do Município – PGM" |  |
| PROMEA | Programa Municipal de Educação Ambiental de São Gonçalo | 2026-01-15: "Programa Municipal de Educação Ambiental de São Gonçalo (PROMEA/SG)" | é programa, não órgão; os atos são da comissão do programa |
| SAMSADC | Secretaria Municipal de Saúde e Defesa Civil | 2020-08-27: "SAMSADC O SECRETÁRIO MUNICIPAL DE SAÚDE E DEFESA CIVIL, no uso das atribuições…" | erro de digitação de SEMSADC (2020) |
| SECGOV | Secretaria Municipal de Governo | 2017-08-17: "PORTARIA Nº 202/SECGOV/COODAF/2017 O SECRETARIO MUNICIPAL DE GOVERNO" | variante de SEMGOV; os atos sob o cabeçalho (2021-10-05) citam "Secretário Municipal de Governo e Comunicação Social" |
| SECHABITA | Secretaria Municipal de Habitação | 2011-11-04: "PORTARIA Nº 001/SECHABITA/2011 O SECRETÁRIO MUNICIPAL DE HABITAÇÃO" |  |
| SEMA | Secretaria Municipal de Meio Ambiente | 2019-12-30: "Secretaria Municipal de Meio Ambiente - SEMA" | sigla secundária (a principal é SEMMA); único ato atribuído (2012-08-08) é extrato de locação sem órgão claro |
| SEMAD | Secretaria Municipal de Administração | 2026-09-17: "Secretaria Municipal de Administração (SEMAD)" |  |
| SEMAIMPD |  |  | uma ocorrência (2014-08-29), sem nome no ato; talvez "Secretaria Municipal de Atenção ao Idoso, Mulher e Pessoa com Deficiência" (nome da SEMIMD em 2013–2014), não confirmado |
| SEMANT | Secretaria Municipal Antidrogas | 2013-02-14: "…DESPESA DA SECRETARIA MUNICIPAL ANTIDROGAS … ofício no 014/SEMANT/13" | 2014-09-19 ato sob o cabeçalho: "O SECRETÁRIO MUNICIPAL ANTIDROGAS"; depois SEMPAD |
| SEMAP | Secretaria Municipal de Agricultura e Pesca | 2023-08-30: "PORTARIA Nº. 01/SEMAP/2023 … NO ÂMBITO DA SECRETARIA MUNICIPAL DE AGRICULTURA E PESCA" | depois SEMAPPAP |
| SEMAPPAP | Secretaria Municipal de Agricultura e Pesca | 2026-06-19: "Secretaria Municipal de Agricultura e Pesca – SEMAPPAP" |  |
| SEMAS | Secretaria Municipal de Assistência Social | 2026-08-17: "SEMAS – Secretaria Municipal de Assistência Social" | sucede SMDSIA (2021) |
| SEMCI | Secretaria Municipal de Controle Interno | 2025-06-24: "Secretaria Municipal de Controle Interno – SEMCI" |  |
| SEMCOM | Secretaria Municipal de Comunicação Social | 2022-10-10: "Secretaria Municipal de Comunicação Social (SEMCOM)" | 2023–2025 também "Secretaria Municipal de Comunicação (SEMCOM)"; capa de 2026 lista "Secretaria Municipal de Comunicação Social" |
| SEMCOMP | Secretaria Municipal de Compras e Suprimentos | 2018-12-18: "Secretaria Municipal de Compras e Suprimentos (SEMCOMP)" | sucede SUBCOMP (Subsecretaria de Compras e Suprimentos) |
| SEMCON | Secretaria Municipal de Conservação | 2026-07-22: "Secretaria Municipal de Conservação – SEMCON" |  |
| SEMCS | Secretaria Municipal de Comunicação | 2011-06-07: cabeçalho SEMCS → "PARTES: MUNICÍPIO DE SÃO GONÇALO, através da Secretaria Municipal de Comunicação" | CS sugere Comunicação Social, não confirmado; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMDE | Secretaria Municipal de Desenvolvimento Econômico | 2026-09-10: "Secretaria Municipal de Desenvolvimento Econômico – SEMDE" | 2018–2020: "…Desenvolvimento Econômico, Ciência e Tecnologia – SEMDE"; ato de 2026-07-23 sob SEMDE: "…Desenvolvimento Econômico e Trabalho" |
| SEMDUR | Secretaria Municipal de Desenvolvimento Urbano | 2026-09-10: "Secretaria Municipal de Desenvolvimento Urbano – SEMDUR" |  |
| SEMED | Secretaria Municipal de Educação | 2026-09-14: "SECRETARIA MUNICIPAL DE EDUCAÇÃO – SEMED" |  |
| SEMEL | Secretaria Municipal de Esporte e Lazer | 2026-08-21: "Secretaria Municipal de Esporte e Lazer – SEMEL" |  |
| SEMFA | Secretaria Municipal de Fazenda | 2026-09-10: "Secretaria Municipal de Fazenda – SEMFA" | capa de 2026 grafa "Secretaria Municipal da Fazenda" |
| SEMGIPE | Secretaria Municipal de Gestão Integrada e Projetos Especiais | 2026-09-10: "Secretaria Municipal de Gestão Integrada e Projetos Especiais – SEMGIPE" |  |
| SEMGOV | Secretaria Municipal de Governo | 2025-06-24: "Secretaria Municipal de Governo – SEMGOV" | 2013–2016 a pasta foi "Governo e Comunicação Social" / "Governo, Comunicação Social e Posturas" (siglas SEMGOVCOM etc.) |
| SEMGOVCOM | Secretaria Municipal de Governo, Comunicação Social e Posturas | 2016-07-20: "PORTARIA Nº 001/SEMGOVCOM/2016 O SECRETÁRIO MUNICIPAL DE GOVERNO, COMUNICAÇÃO SOCIAL E POSTURAS" | em 2014: "PORTARIA Nº 003/SEMGOVCOM/2014 A SECRETÁRIA MUNICIPAL DE GOVERNO E COMUNICAÇÃO SOCIAL" |
| SEMGOVCOMS | Secretaria Municipal de Governo, Comunicação Social e Posturas | 2015-06-11: cabeçalho SEMGOVCOMS → ato assinado "Secretário Municipal de Governo, Comunicação Social e Posturas" | variante de SEMGOVCOM; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMGOVCON | Secretaria Municipal de Governo e Comunicação Social | 2014-08-14: cabeçalho SEMGOVCON → homologação assinada "Secretária Municipal de Governo e Comunicação Social" | variante/erro de SEMGOVCOM; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMHAB | Secretaria Municipal de Habitação | 2026-09-10: "Secretaria Municipal de Habitação – SEMHAB" |  |
| SEMHABITA | Secretaria Municipal de Habitação | 2014-08-27: cabeçalho SEMHABITA → "PORTARIA Nº 001/2014. A SECRETÁRIA MUNICIPAL DE HABITAÇÃO" | variante de SECHABITA; em 2014-03-21 "Joana Dantas … – SEMHABITA" em lista de órgãos; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMIDC | Secretaria Municipal de Integração, Defesa do Consumidor e Políticas para as Mulheres | 2011-05-17: cabeçalho SEMIDC → "…por meio da Secretaria Municipal de Integração e Políticas para Mulheres" | nome completo vem de atos de 2011–2012 ("Integração, Defesa do Consumidor e Políticas para as Mulheres"); provável variante de SEMINDECON; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMIMD | Secretaria Municipal de Políticas Públicas para o Idoso, Mulher e Pessoa com Deficiência | 2020-03-25: "Secretaria Municipal de Políticas Públicas para o Idoso, Mulher e Pessoa com Deficiência - SEMIMD" | 2014: "SECRETÁRIA MUNICIPAL DE ATENÇÃO AO IDOSO, MULHER E PESSOA COM DEFICIÊNCIA – SEMIMD" |
| SEMIND | Secretaria Municipal de Políticas Públicas para o Idoso, Mulher e Pessoa com Deficiência | 2020-11-04: cabeçalho SEMIND → assinatura "Secretaria Municipal de Políticas Públicas para Idoso, Mulher e Pessoa…" | provável variante/erro de SEMIMD (mesma secretária, Marta Maria Figueiredo); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMINDECON | Secretaria Municipal de Integração, Defesa do Consumidor e Políticas para as Mulheres | 2012-10-30: "PORTARIA Nº 004/GAB.SEMINDECON/2012. A SECRETÁRIA DE INTEGRAÇÃO E POLÍTICAS PARA AS MULHERES" | o ato com a sigla usa o nome curto; o nome com "Defesa do Consumidor" vem de atos de 2011–2012 da mesma pasta (letras DECON batem), não ao lado da sigla |
| SEMIURB | Secretaria Municipal de Infraestrutura e Urbanismo | 2016-03-08: "Secretaria Municipal de Infraestrutura e Urbanismo – SEMIURB" | de 2015 a 2016 o cabeçalho passou a SEMIURBCPARJ |
| SEMIURBCPARJ | Secretaria Municipal de Infraestrutura, Urbanismo e Conservação de Parques e Jardins | 2015-07-14: "…/SEMIURBCPARJ … O SECRETÁRIO MUNICIPAL DE INFRAESTRUTURA E URBANISMO E CONSERVAÇÃO DE PARQUES E JARDINS" |  |
| SEMIURME |  |  | uma edição (2010-02-02), atos assinados pela Prefeita (dragagem, obras); sem nome no Diário; talvez variante de SEMIURB, não confirmado |
| SEMMA | Secretaria Municipal de Meio Ambiente | 2025-08-20: "Secretaria Municipal de Meio Ambiente (SEMMA)" | em 2017 SEMMADU; a partir de 2025 SEMMATRAN |
| SEMMADU | Secretaria Municipal de Meio Ambiente e Desenvolvimento Urbano | 2017-02-23: corrigenda da Res. 006/SEMMADU: "…– SECRETÁRIA MUNICIPAL DE MEIO AMBIENTE E DESENVOLVIMENTO URBANO" | existiu jan–abr/2017 |
| SEMMATRAN | Secretaria Municipal de Meio Ambiente e Transportes | 2026-09-18: "Secretaria Municipal de Meio Ambiente e Transportes – SEMMATRAN" | junção de SEMMA e SEMTRAN (2025) |
| SEMOP | Secretaria Municipal de Ordem Pública | 2023-05-08: "Secretaria Municipal de Ordem Pública (SEMOP)" | mesma pasta da SEOP |
| SEMPAD | Secretaria Municipal de Políticas sobre Álcool e Drogas | 2026-09-16: "Secretaria Municipal de Políticas sobre Álcool e Drogas (SEMPAD)" | 2026-07-22 também "…de Políticas Públicas sobre Álcool e Drogas"; antes "Secretaria Municipal Antidrogas" (SEMANT) |
| SEMPESCA | Secretaria Municipal de Pesca | 2014-03-31: "Secretaria Municipal de Pesca (SEMPESCA)" |  |
| SEMPLAN | Secretaria Municipal de Planejamento | 2010-08-23: "O SECRETÁRIO MUNICIPAL DE PLANEJAMENTO-SEMPLAN" | ato de 2013 sob SEMPLAN assinado "Secretário Municipal de Planejamento e Projetos Especiais" (SEMPPE) |
| SEMPPE | Secretaria Municipal de Planejamento e Projetos Especiais | 2017-02-08: "Secretaria Municipal de Planejamento e Projetos Especiais – SEMPPE" |  |
| SEMSA | Secretaria Municipal de Saúde e Assistência | 2016-10-06: "Secretaria Municipal de Saúde e Assistência (SEMSA)" | 2010–2012: "Secretaria Municipal de Saúde" (ainda citada assim em 2026-04-13); sucedida por SEMSADC em 2017 |
| SEMSAD | Secretaria Municipal de Saúde e Defesa Civil | 2018-03-29: cabeçalho SEMSAD → resolução assinada "Secretário Municipal de Saúde e Defesa Civil" | erro de digitação de SEMSADC; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SEMSADC | Secretaria Municipal de Saúde e Defesa Civil | 2026-08-07: "Secretaria Municipal de Saúde e Defesa Civil – SEMSADC" |  |
| SEMSEP | Secretaria Municipal de Segurança Pública | 2020-11-05: "Secretaria Municipal de Segurança Pública (SEMSEP)" | sucedida por SEOP/SEMOP (Ordem Pública) em 2021 |
| SEMTCUL | Secretaria Municipal de Turismo e Cultura | 2020-11-11: "CHAMAMENTO PÚBLICO 001/2020/SEMTCUL. O SECRETÁRIO MUNICIPAL DE TURISMO E CULTURA" | mesma pasta da SMTC e SETURCUL |
| SEMTRAB | Secretaria Municipal do Trabalho | 2016-03-07: "Secretaria Municipal do Trabalho (SEMTRAB)" |  |
| SEMTRAN | Secretaria Municipal de Transportes | 2025-08-20: "Secretaria Municipal de Transportes (SEMTRAN)" | em 2025 incorporada à SEMMATRAN |
| SEOP | Secretaria Municipal de Ordem Pública | 2026-08-04: "Secretaria Municipal de Ordem Pública – SEOP" |  |
| SETURCUL | Secretaria Municipal de Turismo e Cultura | 2023-06-19: "…FOMENTO N.º 001/2022/SETURCUL … A SECRETÁRIA MUNICIPAL DE TURISMO E CULTURA" | mesma pasta da SMTC |
| SMAP |  |  | uma ocorrência (2012-01-05), ato assinado pelo "Subsecretário de Agricultura e Pesca"; não dá para afirmar se é Secretaria ou Subsecretaria |
| SMAS | Secretaria Municipal de Assistência Social | 2026-08-07: "Secretaria Municipal de Assistência Social – SMAS" | variante de SEMAS; cabeçalho só em jan/2021 |
| SMC |  | 2011-07-15: cabeçalho SMC → termo com a D-BUG Comunicação assinado "Secretária Municipal de Comunicação" | ambígua: no Diário "SMC" também é "Sistema Municipal de Cultura" e "Secretaria Municipal de Cultura" (em texto de lei); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SMDS | Secretaria Municipal de Desenvolvimento Social | 2020-07-15: "Secretaria Municipal de Desenvolvimento Social - SMDS" | cabeçalho 2010–2015; depois SMDSHABIA, SMDSIA e SEMAS |
| SMDSHABIA | Secretaria Municipal de Desenvolvimento Social, Habitação, Infância e Adolescência | 2016-10-06: "Secretaria Municipal de Desenvolvimento Social, Habitação, Infância e Adolescência – SMDSHABIA" |  |
| SMDSIA | Secretaria Municipal de Desenvolvimento Social, Infância e Adolescência | 2021-06-15: "Secretaria Municipal de Desenvolvimento Social, Infância e Adolescência (SMDSIA)" |  |
| SMPPE | Secretaria Municipal de Planejamento e Projetos Especiais | 2013-04-19: cabeçalho SMPPE → "O SECRETÁRIO MUNICIPAL DE PLANEJAMENTO E PROJETOS ESPECIAIS" | provável variante de SEMPPE; evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
| SMSP | Secretaria Municipal de Segurança Pública | 2012-08-07: "Secretaria Municipal de Segurança Pública – SMSP" | variante de SEMSEP |
| SMSS | Secretaria Municipal de Seguridade Social | 2011-12-21: "Secretaria Municipal de Seguridade Social - SMSS" |  |
| SMTC | Secretaria Municipal de Turismo e Cultura | 2026-09-11: "Secretaria Municipal de Turismo e Cultura – SMTC" |  |
| SSM |  |  | sigla parece falsa: "SSM" é símbolo de cargo em comissão (Subsecretário Municipal, "Símbolo SSM"); os 4 atos atribuídos (2013) são corrigendas de portarias |
| SUBCOMP | Subsecretaria de Compras e Suprimentos | 2011-05-10 (ato sob SUBCOMP): "informações poderão ser obtidas na Subsecretaria de Compras e Suprimentos…" | também "Subsecretaria Municipal de Compras e Suprimentos" (2012) e contratos "PMSG/GAB/SUBCOMP"; sucedida por SEMCOMP (2017); evidência indireta: cabeçalho seguido de ato que identifica o órgão, sem sigla e nome lado a lado |
