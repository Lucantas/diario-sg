# Agentes políticos e subsídios

## Problema

A etapa C do plano de fontes e a Entrega 5 pedem os agentes políticos
com nome e remuneração mês a mês: prefeito, vice, secretários e
vereadores (decisão 3 do plano de fontes e ADR 0006). Os demais
servidores só entram em agregado, e isso a página `/pessoal` já faz.

## Fontes (conferidas em 26/09/2026)

- **Folha da Prefeitura.** API pública do portal de transparência:
  `GET https://sistema.pmsg.rj.gov.br/pmsaogoncalo/websis/portal_transparencia/financeiro/contas_publicas/lai_remuneracoes_api.php?flag=remuneracao&entidade=1&competencia=MM/AAAA`.
  JSON `{success, data}`, cerca de 9.500 linhas e 3 MB por mês, de 10/2010
  a 08/2026. Traz nome, cargo, função, lotação, referência e o valor
  bruto do mês (`remuneracao`); CPF e matrícula vêm mascarados, sem
  descontos nem líquido.
- **Folha da Câmara.** Dados abertos do portal da Câmara
  (`https://cmsaogoncalo-rj.portaltp.com.br/api/pessoal/api-servidores.aspx`):
  o formulário da página exporta em JSON, com o mesmo POST que o navegador
  faz ao clicar em "Exportar" (lê `__VIEWSTATE` e manda ano, mês `00` e
  formato). Um ano por pedido, de 2017 a 2026. Traz vencimentos, descontos
  e líquido por mês; o regime `Agente Político` separa os vereadores.
- **Vereadores.** `GET https://sg.processolegislativo.com.br/integracao/?Parlamentares/AAAA/json`
  (a API de integração do SICAM que o site da Câmara usa para a página de
  vereadores): nome civil, nome parlamentar, partido, legislatura e
  situação. Os outros campos (nascimento, e-mail, telefone, foto,
  biografia) são descartados na leitura.
- **Subsídio fixado.** Duas normas no próprio Diário: Resolução nº
  2.156/2024 da Câmara (vereador, R$ 21.840,43 de 2025 a 2028) e Lei nº
  1.554/2024 (prefeito R$ 23.813,22, vice R$ 19.148,16, secretário e
  Procurador-Geral R$ 16.754,64, de 2025 a 2028). O Procurador-Geral é
  agente político pela própria lei.

## Desenho

- **Quem entra.** Da Prefeitura, a linha cuja `funcao` é `PREFEITO`,
  `VICE-PREFEITO`, `SECRETARIO MUNICIPAL` ou `PROCURADOR GERAL DO
  MUNICIPIO` (a função, não o cargo: há secretário efetivo cujo cargo é o
  de origem; `SECRETARIO ESCOLAR` e `SUBSECRETARIO` ficam fora). Da Câmara,
  regime `Agente Político` e cargo `VEREADOR`. Toda outra linha é
  descartada na leitura e não é arquivada.
- **Tabelas** (migration 023). `political_agent_pay` (órgão, mês, nome,
  chave do nome, papel, lotação, bruto, descontos e líquido quando há) e
  `councillors` (legislatura, nome, chave, nome parlamentar, partido,
  situação). Sem CPF nem matrícula.
- **Job `agentes`** (`make agentes`), mensal no dia 10: por padrão os três
  últimos meses; `-from AAAA-MM -to AAAA-MM` para o histórico. Troca os
  meses do período numa transação por órgão; mês que falha aborta o órgão
  inteiro antes da troca. Pausa de 1 s entre pedidos. Arquiva só as linhas
  dos agentes, em `raw/agentes_politicos/AAAA/MM/DD/`, com o SHA-256 de
  cada resposta inteira no manifesto.
- **Leitura.** Agente = órgão + papel + chave do nome, com os meses, as
  lotações e, para vereador, partido e nome parlamentar da legislatura
  (casados pela chave do nome, sem acento; se não casa, pelo nome
  parlamentar da lotação sem o "VEREADOR").
- **API** `GET /v1/agentes` (`role` e `q` filtram): agentes, normas de
  subsídio e cobertura de cada folha. **Web** `/agentes`: normas no topo,
  lista por papel, cada agente com a remuneração mês a mês e a busca do
  nome no Diário. **MCP** `agentes_politicos` (nome e cargo).

## Cuidados

- Nome é a única chave entre as fontes (CPF e matrícula mascarados).
- Dezembro da Câmara traz o 13º junto; a Prefeitura só informa o bruto.
- Partido é o que o SICAM informa hoje para a legislatura.

## Fora

- Nomeações de secretário pelo Diário: a regra achou 7 dos 20 secretários
  da folha de 08/2026; a folha é a fonte do período, e o Diário fica na
  busca pelo nome.
- Normas anteriores a 2024 (Resolução 016/2020 não está na base).

## Depois da entrega

- A troca é por órgão, e cada órgão é gravado à parte: se a Câmara falha, a
  Prefeitura é gravada assim mesmo (e o contrário), e o erro fica na coleta.
- Da Câmara, só a mensagem "não houve informações disponibilizadas" conta
  como ano vazio; outra mensagem, ou um valor que não seja número, é erro.
- `agentes_politicos` devolve até 20 agentes por chamada e aceita `pular`,
  para chegar aos 27 vereadores.
- A folha da Prefeitura não traz prefeito nem vice antes de 06/2012, nem
  secretários antes de 07/2011 (conferido em 12/2010).
- Grafias diferentes do mesmo nome entre meses (NATAM e NATAN) viram dois
  agentes; sem CPF nem matrícula, não há como juntar com segurança.
