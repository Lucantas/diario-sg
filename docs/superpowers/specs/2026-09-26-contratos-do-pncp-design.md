# Contratos do PNCP (Entrega 4, parte 3)

Data: 2026-09-26. Terceira fonte da Entrega 4. Traz os contratos que a
Prefeitura, seus fundos e fundações e a Câmara registraram no Portal
Nacional de Contratações Públicas (Lei 14.133), mostra-os na página da
empresa e cruza com o Diário num padrão novo.

## Fatos que orientam o desenho

- A API de consulta é aberta:
  `https://pncp.gov.br/api/consulta/v1/contratos?dataInicial=AAAAMMDD&dataFinal=AAAAMMDD&cnpjOrgao=<cnpj>&pagina=N&tamanhoPagina=50`.
  Período máximo de 365 dias (422 acima disso); 204 sem corpo quando não
  há contrato; `paginasRestantes` diz se há mais páginas.
- Limite de uso: rajadas de pedidos recebem uma página HTML "Limite de
  Requisições Excedido" com status 200 ou 429. Cada pedido leva de 10 a
  20 segundos.
- O município registra pouco: a Prefeitura tem 2 contratos em 2024, 36 em
  2025 e 20 em 2026; os fundos e fundações, quase nada; antes de 2024,
  nada. A carga inteira cabe em algumas dezenas de pedidos.
- Cada contrato traz o número de controle PNCP, o órgão e a unidade, o
  número do contrato, o processo, o fornecedor (`tipoPessoa` PJ/PF e
  `niFornecedor`), o objeto, o valor global e as datas de assinatura,
  publicação e vigência. A página pública é
  `https://pncp.gov.br/app/contratos/<cnpj do órgão>/<ano>/<sequencial>`.

## Decisões

1. **Job semanal no módulo da API** (`cmd/pncp`, `make pncp`), domingo às
   06:00, como o `tce`. Fonte `pncp_contratos` em `fetch_runs`. Sem
   argumentos, lê de 2021 ao ano corrente; `-from`/`-to` restringem.
2. **Órgãos:** a Prefeitura (28.636.579/0001-00) e os CNPJs de órgãos
   municipais já conhecidos (`PublicBodyCNPJs`), um pedido por órgão, ano e
   página, com pausa de 3 segundos entre pedidos e espera de 1 minuto
   (até 4 tentativas) quando o limite estoura.
3. **Só pessoa jurídica.** Contratos com fornecedor pessoa física são
   descartados na leitura, sem chegar ao banco nem ao bucket (ADR 0006).
4. **Tabela `pncp_contracts`** (migration 017), chave o número de
   controle. A carga troca, numa transação, todos os contratos publicados
   nos anos lidos e refaz as ligações `contrato_pncp` com o CNPJ do fornecedor
   (`exata`). A tabela é conferida antes do primeiro pedido.
5. **Arquivo bruto:** os contratos lidos em
   `raw/pncp_contratos/AAAA/MM/DD/contratos.jsonl.gz` e um
   `manifest.json` com o SHA-256 das respostas e a contagem.
6. **Página da empresa** ganha "Contratos no PNCP": número, órgão, objeto,
   valor, assinatura, vigência e o link para o PNCP. `GET
   /v1/entities/cnpj/{cnpj}` traz `pncp_contracts`; a ferramenta
   `entidade` do MCP traz `contratos_pncp`.
7. **Padrão `pncp_sem_extrato`:** contrato do PNCP de pelo menos
   R$ 100 mil, assinado pelo menos 30 dias antes da última edição do
   Diário da Prefeitura, cujo fornecedor não é citado em nenhum ato do
   Diário (Prefeitura ou Câmara) e cujo processo também não é. O caso leva
   o link do contrato no PNCP.
8. **Nuvem:** job, agendamento e conta de serviço no Terraform, como o
   `tce`.

## Fora do escopo

- Atas de registro de preço e contratações (editais) do PNCP.
- Termos aditivos do PNCP.
- Casar cada contrato do PNCP com um extrato específico do Diário.

## Testes

- Adapter contra `httptest`: paginação, 204, página HTML de limite com
  nova tentativa, pessoa física marcada.
- Caso de uso com fakes: anos padrão, descarte de pessoa física, bruto e
  manifesto, falha num pedido não grava nada.
- Domínio do padrão: citado pelo CNPJ ou pelo processo não entra; abaixo
  do mínimo e recente não entram.
- Integração: troca dos anos lidos preservando os outros, ligação
  `exata`, `/v1/entities/cnpj/{cnpj}` com os contratos.
- Carga real local e conferência de 5 contratos na página do PNCP.

## Depois da entrega

- Carga local de 26/09/2026 em 7 minutos: 131 contratos, 130 gravados e 1
  descartado (pessoa física). Prefeitura: 58 (1 de 2023 publicado em
  2025, 5 de 2024, 35 de 2025, 17 de 2026); SG-PREVI 48; Câmara 20;
  FUNASG 3; Fundação Municipal de Saúde 1. Nada publicado antes de 2024.
- Um CNPJ da lista de órgãos municipais, 28.579.636/0001-00 (grafia
  errada do CNPJ do Município que aparece no Diário), não tem dígitos
  verificadores válidos, e o PNCP responde 422. A lista de órgãos pedidos
  ao PNCP passou a levar só CNPJ válido.
- A consulta filtra pela data de publicação no PNCP, e 8 contratos têm
  ano de contrato diferente do ano de publicação. A troca passou a ser
  pelo ano de publicação, para que uma carga parcial não apague contrato
  publicado em outro ano.
- 106 fornecedores, 102 deles citados pelo CNPJ no Diário (102 ligações
  `exata`).
- O padrão ficou sem nenhum caso. Os 7 contratos de pelo menos R$ 100 mil
  de fornecedor sem CNPJ no Diário têm extrato publicado sem CNPJ: casam
  pelo processo ou pelo nome do fornecedor lido do texto, regra que entrou
  depois da primeira carga (o processo no PNCP às vezes vem sem ano, como
  "29.973"). O padrão fica para as próximas cargas.
- O município às vezes registra o mesmo contrato duas vezes no PNCP (por
  exemplo, "006/2025" e "006", com vigências diferentes); a página mostra
  os dois registros.
- 5 contratos sorteados conferidos contra a API de detalhe do PNCP
  (`/api/pncp/v1/orgaos/<cnpj>/contratos/<ano>/<sequencial>`): fornecedor,
  valor e data de assinatura conferem.

**Pronto quando:** a página de uma empresa com contrato no PNCP mostra o
contrato com o link, e o padrão lista os contratos do PNCP sem extrato no
Diário.
