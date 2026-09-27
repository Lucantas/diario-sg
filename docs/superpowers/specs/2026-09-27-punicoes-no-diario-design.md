# Punições aplicadas pela Prefeitura e publicadas no Diário

## Problema

O CEIS e o CNEP (CGU) não têm nenhuma sanção aplicada pela Prefeitura de
São Gonçalo, e a lista "empresas penalizadas" do portal da transparência
volta vazia (conferido em 27/09/2026). As punições existem: saem no
Diário como "EXTRATO DE MULTA", "EXTRATO DE ADVERTÊNCIA", "EXTRATO DE
SANÇÃO", "TERMO DE ADVERTÊNCIA", "DESPACHO DE ADVERTÊNCIA" (Câmara),
"NOTIFICAÇÃO DE IMPOSIÇÃO DE PENALIDADE", ou dentro de portarias,
decretos e extratos de decisão ("APLICO a penalidade de multa especial
de 10%", "decide aplicar a sanção de multa à empresa…").

## O que a base mostra (163 mil atos)

- 25 atos com título de sanção (advertência, multa, sanção, penalidade),
  um deles falso ("anistia de taxas e multas de IPTU").
- Fórmulas de decisão aparecem também em ruído: portarias que dão à
  comissão de monitoramento a competência de "aplicar penalidade de
  advertência", leis e decretos com regra geral e punição disciplinar de
  servidor. Nenhum desses cita o CNPJ de uma empresa.

## Desenho

- **Domínio.** `ClassifyDiarioSanction(título, corpo)` devolve o tipo
  (`advertencia`, `multa`, `suspensao`, `impedimento`, `inidoneidade`)
  quando o título é de sanção (e não de anistia) ou quando o corpo tem
  uma fórmula de decisão ("aplico", "aplica", "aplicar", "decide
  aplicar", "impõe" + penalidade/sanção/pena + de + tipo). O tipo sai do
  título ou do trecho da decisão, do mais grave para o mais leve.
- **Leitura, sem reindexar.** Os candidatos são os atos ligados ao CNPJ
  (`entity_links`) cujo título ou corpo passa num filtro largo em SQL; o
  domínio decide. A empresa ganha `diario_sanctions` na API, na página
  (junto das sanções da CGU) e `punicoes_diario` no MCP `entidade`.
- Sem tabela nova nem migration.

## Fora

- Lista de todas as punições da cidade: a ligação é por CNPJ, e ato
  sem CNPJ (TERMO DE ADVERTÊNCIA sem número, por exemplo) não entra.
- Valor da multa: o ato traz, e a página mostra o ato.

## Depois da entrega

- Na base local: 96 CNPJs com ato candidato, 23 punições classificadas em
  21 empresas (14 advertências, 9 multas; nenhuma suspensão, impedimento
  ou inidoneidade publicada com CNPJ).
- Com o verbo no infinitivo ("aplicar", "impor"), a decisão só conta com
  "resolve", "decide" ou "decido" logo antes: assim a portaria que dá à
  comissão de monitoramento a competência de "aplicar penalidade de
  advertência" não vira punição, e "RESOLVE: Art. 1º - Aplicar a
  penalidade de Advertência à FUNDAÇÃO BIO RIO" vira.
- Um "EXTRATO DE ADVERTÊNCIA" saiu colado ao fim do ato anterior (um
  aditivo de locação de 2014). A punição aparece com o título do aditivo;
  a página avisa. Corrigir a segmentação é trabalho do parser.
