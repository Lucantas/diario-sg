# Normas municipais (consulta de leis do SIAPEGOV)

## Problema

Atos do Diário citam leis e decretos ("com base na Lei nº 1.406/2022"),
e o MCP não tem onde conferir o que a norma diz, quem propôs e quando foi
promulgada. O plano de fontes pede a consulta de leis do SIAPEGOV e a
entidade `norma`.

## Fonte (conferida em 27/09/2026)

`POST https://sistema.pmsg.rj.gov.br/pmsaogoncalo/websis/siapegov/legislativo/leis/resulta_leis.php`,
formulário em ISO-8859-1 sem chave. Com `paginacao=0` a resposta traz a
categoria inteira numa tabela: número/ano, categoria, altera, alterada
por, autor, ementa, data de promulgação e o botão do texto integral
(`formata_lei.php?Fnumero=N/AAAA&Fpromulgacao=DD/MM/AAAA&categoria=C`).
Categorias: 01 Lei Ordinária (3.473), 02 Lei Orgânica (8), 03 Lei
Complementar (35) e 05 Decreto (12.953, 11 MB em 23 s). O autor vem em
770 leis ("VEREADOR … PROJETO DE LEI 110/22"); há números repetidos (2
leis, 5 decretos).

## Desenho

- **Domínio.** `Norm` (tipo, número, ano, autor, ementa, promulgação, URL
  do texto) com os tipos `lei`, `lei_complementar`, `lei_organica` e
  `decreto`. `ParseNorms(tipo, html)` lê a tabela; `ParseNormNumber`
  aceita `1.406/2022` e `1406/22`. Repetidos: fica a primeira linha e a
  carga conta as outras em `skipped`.
- **Coleta.** Passo novo do job `agentes` (o que já lê a Prefeitura e a
  Câmara): as quatro categorias, 1 pedido por segundo. Bruto em
  `raw/siapegov_normas/AAAA/MM/DD/<tipo>.html.gz`; `fetch_runs` com a
  fonte `siapegov_normas`. Troca tudo.
- **Tabela** `norms` (migration 029).
- **Leitura.**
  - `GET /v1/norms?kind=&number=` (uma norma) ou `?q=` (ementa e autor,
    até 50, mais recentes primeiro).
  - Ferramenta `norma` no MCP: por tipo e número, ou por texto; cada
    norma traz a ementa, o autor, a promulgação, o link do texto integral
    e a busca do número no Diário.

## Fora

- Guardar o texto integral (um pedido por norma).
- Ligar cada ato do Diário às normas que ele cita (mudaria o parser e
  pediria reindexação); a ferramenta devolve a busca pronta.
- Página no site.

## Depois da entrega

- Carga real: 16.469 linhas em 31 segundos; 16.456 gravadas, 10
  repetidas e 3 com ano ilegível (`245/202`, `442/221`, `107/019`).
- O número pode ter letra (`057/1955 A` e `B` são normas diferentes) e
  caracteres de controle (`1561/2025\x1f`): a letra vira `suffix`, parte
  da chave, e os controles saem.
- A busca usa a configuração `portuguese_unaccent`, a mesma dos atos:
  "subsidio" acha "SUBSÍDIOS".
- A página chega em ISO-8859-1 e o adaptador converte para UTF-8.
