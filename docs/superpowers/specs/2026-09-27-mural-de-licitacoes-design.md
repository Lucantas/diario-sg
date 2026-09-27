# Mural de licitações e contratos da Prefeitura

## Problema

O plano de fontes (etapa E) pede o mural de licitações e contratos da
Prefeitura (`licitacao.pmsg.rj.gov.br`). Ele é a única fonte com a lista
de contratos e atas publicada pela própria Prefeitura, com o documento
anexado; o PNCP cobre pouco (2 contratos em 2024).

## Fonte (conferida em 27/09/2026)

- Quatro listas em HTML, cada uma inteira numa página só:
  `licitacoes.php` (1.190), `dispensas.php` (44),
  `inexigibilidades.php` (37) e `contratos.php` (645). As três primeiras
  têm edital, processo, modalidade, critério, data de abertura, objeto e
  situação; a de contratos tem edital, processo, modalidade, objeto,
  valor, fornecedor (só o nome), o instrumento (contrato, ata de registro
  de preços, nota de empenho) e o link do documento
  (`download.php?idf_1=N`). Cada linha aponta para
  `licitacao.php?licitacao_id=N`.
- O certificado é do Let's Encrypt pela raiz nova "ISRG Root YR"
  (2025), que o repositório de certificados desta máquina ainda não tem.
  O adaptador confia no sistema e também nessa raiz, embutida no código
  (baixada de `letsencrypt.org/certs/gen-y/root-yr.pem`, SHA-256
  `E5:7B:…:A8:6F`). Não desliga a verificação.
- O processo vem como no portal de empenhos (número/ano), às vezes com
  ponto de milhar e zero à esquerda (`02.960/2026`).

## Desenho

- **Domínio.** `Procurement` (id, lista, edital, processo, chave do
  processo, modalidade, critério, abertura, objeto, situação) e
  `ProcurementContract` (id da licitação, edital, processo, chave,
  modalidade, objeto, valor, fornecedor, instrumento, URL do documento).
  `ProcessKey` normaliza para `número/ano` sem ponto nem zero à esquerda.
  O parser lê as tabelas e erra se a lista vier sem linhas ou sem as
  colunas esperadas.
- **Coleta.** Passo novo do job `pncp` (contratações): as quatro listas,
  1 pedido por segundo, User-Agent do projeto. Bruto de cada lista em
  `raw/pmsg_mural/AAAA/MM/DD/<lista>.html.gz` com manifesto;
  `fetch_runs` com a fonte `pmsg_mural`. Troca tudo a cada carga.
- **Tabelas** (migration 028): `procurements` e `procurement_contracts`.
- **Ligação com a empresa.** O mural não tem CNPJ. A empresa ganha
  `procurements`: as linhas do mural cujo processo é o de algum empenho
  dela no portal da Prefeitura. A chave do processo não tem o órgão,
  então a ligação é provável, e a página diz isso.

## Fora

- Página de detalhe de cada licitação (um pedido por linha): a lista já
  traz o que a ligação usa.
- Baixar os documentos anexados.

## Depois da entrega

- Carga real: 1.190 licitações, 44 dispensas, 37 inexigibilidades e 645
  contratos e atas em 3 segundos; 492 contratos casam com o processo de
  algum empenho do portal.
- `ProcessKey` também lê o ano com dois dígitos (`30656/25`) e o formato
  do Diário, com órgão e dígito (`74.00210/2025-0` vira `210/2025`): com
  cinco dígitos depois do ponto, o que vem antes é o órgão; com três, é
  ponto de milhar.
- A URL de detalhe da licitação é gravada na linha, como a do documento.
- O `golang.org/x/net/html` entrou na versão v0.50.0, a última que aceita
  Go 1.25; a v0.59 pedia Go 1.26 e subiria a versão do projeto.
