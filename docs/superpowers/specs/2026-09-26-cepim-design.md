# CEPIM no job de sanções

## Problema

O CEPIM (Cadastro de Entidades Privadas sem Fins Lucrativos Impedidas, da
CGU) respondia 403 quando as sanções entraram, e ficou de fora. Em
26/09/2026 o arquivo voltou a responder: 3.529 linhas no arquivo de
24/09/2026, 16 delas de CNPJs citados no Diário (associações, ligas
esportivas, ONGs que recebem convênio da Prefeitura).

## Fonte

`https://portaldatransparencia.gov.br/download-de-dados/cepim/AAAAMMDD`,
o mesmo esquema do CEIS e do CNEP (ZIP com CSV `;` em ISO-8859-1). As
colunas são outras:

`CNPJ ENTIDADE; NOME ENTIDADE; NÚMERO CONVÊNIO; ÓRGÃO CONCEDENTE; MOTIVO DO IMPEDIMENTO`

Não há datas, tipo de pessoa nem código de sanção. O par (CNPJ, convênio)
é único no arquivo.

## Desenho

- O CEPIM vira mais um cadastro em `SanctionRegisters`; o loader escolhe o
  leitor de linhas pelo cadastro (`NewSanctionRows`). CEIS e CNEP continuam
  com o leitor de hoje.
- Mapeamento para `Sanction`: código `CNPJ/convênio`; categoria
  "Impedida de receber transferências da União"; órgão = órgão concedente;
  esfera "FEDERAL"; abrangência "Convênios e transferências voluntárias da
  União"; processo "Convênio N"; fundamentação = motivo; sem datas nem
  multa.
- Migration 021 troca o CHECK de `cgu_sanctions.register` para aceitar
  `CEPIM`.
- O padrão "contratação durante sanção" ignora o CEPIM: o impedimento é de
  receber dinheiro federal por convênio, não de contratar com o Município,
  e sem datas ele cobriria todo o período.
- Página da empresa e MCP ganham o rótulo do CEPIM.

## Fora

- Acordos de leniência.
- Histórico do CEPIM anterior à primeira carga (o arquivo só tem o
  estoque do dia).

## Depois da entrega

Carga de 26/09/2026 sobre o arquivo de 24/09/2026: 3.529 linhas no CEPIM,
16 impedimentos guardados, de 6 entidades citadas, a mesma contagem feita
à mão sobre o CSV. A Liga Gonçalense de Desportos aparece com um convênio
do Ministério do Turismo. O padrão "sancionado contratado" continua sem
caso, e os de anunciado × pago ficaram em 75, 9 e 68, como antes.
