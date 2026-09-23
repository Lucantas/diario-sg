# Fontes públicas de São Gonçalo

Levantamento de 23/09/2026. Cada URL foi aberta nesse dia (só leitura). Onde
está escrito "não verificado", a URL veio de busca ou de link e não abriu.
O plano de uso está em [`../plano-fontes-publicas.md`](../plano-fontes-publicas.md).

Cuidados de acesso que valem para quase tudo abaixo:

- Os hosts `*.pmsg.rj.gov.br` e `*.saogoncalo.rj.gov.br` mandam a cadeia TLS
  incompleta (clientes que validam certificado falham) e não respondem por
  IPv6.
- `*.portaltp.com.br` e `*.el.com.br` devolvem 403 sem `User-Agent` de
  navegador.
- Vários portais são aplicações só em JavaScript; a API por trás delas não
  é documentada.

## Prefeitura

| Fonte | URL | O que tem | Acesso | Período | Chaves |
| --- | --- | --- | --- | --- | --- |
| Diário Oficial (já indexado) | https://do.pmsg.rj.gov.br | Atos do Executivo | PDF | 2010– | CNPJ, processo, contrato, valor (extraídos) |
| Portal da Transparência antigo (EL, "portaltp") | https://saogoncalo-rj.portaltp.com.br/api/dadosabertos.aspx | Empenhos, liquidações e pagamentos por favorecido; ordem cronológica; diárias, obras, convênios | **API aberta**: `/api/transparencia.asmx/json_empenhos?ano=&mes=` (JSON dentro de `<string>` XML), também `json_liquidacoes`, `json_pagamentos` | **2017 a 2022**; de 2023 em diante volta vazio (conferido: 06/2022 = 169 empenhos, 06/2023 = 0) | CNPJ/CPF do favorecido, nº do empenho |
| Portal da Transparência novo (Embras / SIAPEGOV) | https://sistema.pmsg.rj.gov.br/portal-transparencia/home | Receitas, empenhos, despesa paga, diárias, licitações, estrutura remuneratória, emendas; 30 entidades (Prefeitura, FMS, FAESG, FUNASG, SG-PREVI, fundos) | Aplicação Angular com API JSON em `/portal-transparencia/api/`; os endpoints de empenho e despesa **ainda não foram descobertos** | 2023– (esperado) | CNPJ do favorecido (a confirmar) |
| Mural de licitações e contratos | https://licitacao.pmsg.rj.gov.br/licitacoes.php (e `contratos.php`, `dispensas.php`, `inexigibilidades.php`) | Editais com anexos, contratos com fornecedor, valor e objeto | HTML; CSV por GET (`?submit=csv`), malformado (sem quebra de linha entre registros) | 2014– | Processo, nº do edital (sufixo indica o órgão), nome do fornecedor; **sem CNPJ** |
| Compras.gov (pregão eletrônico) | https://www.saogoncalo.rj.gov.br/compras/comprasnet/ | Pregões desde 2019 | UASG 926946 | 2019– | UASG, processo |
| PNCP | https://pncp.gov.br/api/consulta/v1/contratos?cnpjOrgao=28636579000100 | Contratos e contratações publicados pelo município | API JSON | Cobertura baixa: 2 contratos em 2024, 36 em 2025, 20 em 2026 (segundo a pesquisa; a API deu 504 na nossa reconferência) | CNPJ do fornecedor, processo, `numeroControlePNCP` |
| Consulta de leis (SIAPEGOV) | https://sistema.pmsg.rj.gov.br/pmsaogoncalo/websis/siapegov/legislativo/leis/consulta_leis.php | Leis, leis complementares, decretos, Lei Orgânica | Formulário POST; anexo PDF por GET | 1940– | Nº e ano |
| Contas públicas | https://www.saogoncalo.rj.gov.br/governo/contas-publicas/ | LDO, LOA, PPA por exercício | PDF | 2006–2027 | Nº da lei |
| Hub de transparência | https://www.pmsg.rj.gov.br/transparencia/ | RREO, RGF, prestação de contas, pareceres do TCE, planilha de obras (XLSX com CNPJ e processo) | PDF, XLSX, CSV | 2016– | CNPJ, processo (obras) |
| Parcerias com OSC | https://www.saogoncalo.rj.gov.br/parcerias-sociedade-civil/ | Termos de colaboração e fomento | PDF | 2021–2025 | Nº do termo, processo |
| Concursos | https://www.saogoncalo.rj.gov.br/concursos/ | Editais, resultados, convocados em XLSX | PDF, XLSX | 2020–2024 | Nome, inscrição |
| SEI | https://www.saogoncalo.rj.gov.br/sei/ | Consulta de processo | HTML | — | Nº do processo |
| SG-PREVI (IPASG) | https://ipasg.jgbaiao.com.br/ipasg_portal/ | Contratos, folha, investimentos, licitações | HTML (subitens não abertos) | — | — |

Não existem: portal de dados abertos (CKAN ou similar), dado de frota,
site próprio da FMS.

### CNPJs dos órgãos (conferidos na base da Receita)

| Órgão | CNPJ |
| --- | --- |
| Município de São Gonçalo (Prefeitura) | 28.636.579/0001-00 |
| Fundação Municipal de Saúde | 39.260.120/0001-63 |
| Fundo Municipal de Saúde | 11.884.903/0001-07 |
| SG-PREVI (ex-IPASG) | 32.538.167/0001-05 |
| FAESG / FAELSG | 04.541.202/0001-00 |
| FUNASG | 14.472.412/0001-39 |
| Fundo Municipal de Educação | 31.023.457/0001-45 |
| Fundo Municipal de Apoio à Pessoa com Deficiência | 29.173.414/0001-02 |
| Câmara Municipal | 29.846.003/0001-22 |

## Câmara Municipal

| Fonte | URL | O que tem | Acesso | Período | Chaves |
| --- | --- | --- | --- | --- | --- |
| Diário Oficial Eletrônico do Legislativo | https://www.cmsg.rj.gov.br/diariooficialeletronico/ | Resoluções, leis promulgadas, decretos legislativos, atos da Mesa, portarias, licitações, contratos, prestação de contas da verba de gabinete (CEAPM) | Busca por formulário POST; PDF por edição em `PUBLICACOES/AAAA-MM-DD.pdf`; sem API | 08/02/2018– (5.822 registros) | Nº do ato, CNPJ e processo nos extratos |
| Processo legislativo (SICAM, DB Nova) | https://sg.processolegislativo.com.br/areapublica/processos | Proposições, tramitação (placar da votação no texto), pautas, atas, parlamentares | HTML; API JSON não documentada: `POST https://api.sicam.app/pesquisar` com cabeçalho `database: saogoncalo` | 2020– (pelo menos) | Nº/ano da proposição, processo, autor |
| Portal da Transparência (EL, portaltp) | https://cmsaogoncalo-rj.portaltp.com.br/ | Folha nominal **com os vereadores**, despesas, contratos, duodécimo, verbas indenizatórias | HTML; endpoints em `/api/dadosabertos.aspx`, mas o GET devolveu HTML (a exportação parece exigir postback) | Seletor 2011–2027 (cobertura não conferida) | Nome, cargo, lotação; CPF mascarado |
| e-SIC | https://gpi-services.cloud.el.com.br/rj-saogoncalo-cm/e-sic/ | Pedidos de informação | Exige login | — | Protocolo |
| TV Câmara | https://www.cmsg.rj.gov.br/tvcamara/ | Vídeos das sessões (YouTube) | HTML | — | Data da sessão |

Exemplo do que só existe aqui: a **Resolução 2.156/2024** fixa o subsídio
dos vereadores em R$ 21.840,43 para 2025–2028 (D.O.E. da Câmara de
28/11/2024, conferido no PDF); a anterior, Resolução 016/2020, fixava
R$ 18.991,68. Nenhuma das duas sai no Diário da Prefeitura.

Lacunas: votação nominal e presença não estão em formato estruturado; não
há seção de diárias; o D.O.E. da Câmara começa em 2018 e não se sabe onde
os atos saíam antes.

## Estado e União

Identificadores do município, que mudam de fonte para fonte (vai precisar
de tabela de-para): IBGE **3304904**; SIAFI **5897** (arquivos da CGU);
código de município próprio da Receita; texto `SAO GONCALO` no TCE-RJ.

| Fonte | URL | O que tem sobre SG | Acesso | Atualização | Chaves | Licença/limites | Verificação |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **TCE-RJ, dados abertos** | https://dados.tcerj.tc.br/api/v1/docs | `empenho_municipio` (**CNPJ/CPF do credor, empenhado, liquidado, pago por empenho e mês**, inclusive 2025 e 2026); `compras_diretas_municipio` (dispensas com CNPJ); `licitacoes`; vencedores e perdedores (só nome); `prestacao_contas_municipio` (pareceres das contas de governo); `penalidades_ressarcimento_municipio` (multas, sem CPF/CNPJ); receitas, dotação, pessoal, obras paralisadas | API JSON ou CSV (`?csv=true`), sem chave, filtros `ano` e `municipio=SAO GONCALO` | Não declarada | CNPJ e nº do empenho (sem processo nem contrato); processo só em `licitacoes` | ODbL; sem limite declarado, mas empenhos de um ano levam minutos | Respondeu; empenhos de 2025 conferidos. `contratos_municipio` volta vazio para todos os municípios |
| **PNCP** | https://pncp.gov.br/api/consulta/swagger-ui/index.html | Contratações, contratos e atas da Prefeitura, fundações, SG-PREVI e Câmara | API JSON sem chave; datas e modalidade obrigatórias; páginas de 10 a 50 | Quase em tempo real | CNPJ do órgão e do fornecedor, processo, `numeroControlePNCP` | Não declarada | Respondeu, mas instável (500 a 504) |
| **CGU, downloads** | https://portaldatransparencia.gov.br/download-de-dados | CEIS, CNEP, CEPIM, leniência (sanções); transferências (283 linhas para SG em 08/2026: FPM, FUNDEB, saúde, FNAS, FNDE); emendas parlamentares (73 com destino a SG); convênios | ZIP com CSV `;` em ISO-8859-1, sem chave | Sanções diárias; transferências mensais | CNPJ/CPF, código SIAFI, nº da emenda e do convênio | Não declarada | Respondeu; CEIS, transferências e emendas baixados e contados. O CEIS não tem nenhuma sanção aplicada pela Prefeitura |
| CGU, API | https://api.portaldatransparencia.gov.br/swagger-ui/index.html | Recursos recebidos, convênios, emendas, sanções, contratos por CNPJ | JSON; **exige chave** (cadastro por e-mail) | — | IBGE, CNPJ | 400 req/min de dia, 700 de madrugada | Documentação conferida; dados não consultados (sem chave) |
| **Receita, CNPJ** | https://arquivos.receitafederal.gov.br/index.php/s/gn672Ad4CF8N6TK (pasta `Dados/Cadastros/CNPJ/AAAA-MM/`) | Todos os CNPJs: empresas, estabelecimentos, **sócios** (CPF mascarado), Simples | ZIP; ~7,8 GB compactados por mês (09/2026) | Mensal | CNPJ, CNAE | Não confirmada | Listagem conferida; arquivos não baixados. O caminho antigo dá 404 |
| **SICONFI (Tesouro)** | https://apidatalake.tesouro.gov.br/docs/siconfi/ | RREO, RGF, DCA, MSC do município e da Câmara | API JSON | Bimestral a anual | `id_ente=3304904`, conta contábil | Apache 2.0; 1 req/s | Respondeu com dados de SG |
| **Transferegov** | https://api.transferegov.gestao.gov.br/transferenciasespeciais/ | Emendas Pix do município (parlamentar, emenda, valor); fundo a fundo | API PostgREST | — | CNPJ do beneficiário, nº da emenda | Não declarada | Respondeu com dados de SG. O download em lote (ex-SICONV) está parado desde 17/07/2026 |
| TSE | https://dadosabertos.tse.jus.br/ | Candidatos, bens, receitas e doações de campanha | CSV em ZIP | Por eleição | CPF/CNPJ do doador, candidato | Não confirmada | **Não verificado**: 403 do Akamai em tudo. Alternativa a conferir: https://basedosdados.org/dataset/br-tse-eleicoes |
| FNDE, Fundo Nacional de Saúde | https://www.fnde.gov.br/dadosabertos/, https://consultafns.saude.gov.br/ | Repasses | Sem API aberta confirmada | — | CNPJ | — | Os repasses aparecem nas transferências da CGU (conferido) |
| CNJ (DataJud, CNIA), TJ-RJ, MP-RJ | https://datajud-wiki.cnj.jus.br/, https://www.cnj.jus.br/improbidade_adm/consultar_requerido.php | Processos (DataJud, sem as partes); condenações por improbidade (CNIA, só consulta manual); inquéritos civis (MP-RJ) | Formulários; DataJud por Elasticsearch | — | Nº CNJ; CPF/CNPJ só no CNIA | — | Páginas responderam; nada consultado |
| IBGE, DataSUS, INEP | https://servicodados.ibge.gov.br/api/docs/ | Contexto (população estimada 2026: 959.977) | APIs e CSV | Anual | IBGE | — | IBGE conferido |
| Querido Diário | https://docs.queridodiario.ok.org.br/ | **Não cobre SG** | — | — | — | — | Repositório conferido |
