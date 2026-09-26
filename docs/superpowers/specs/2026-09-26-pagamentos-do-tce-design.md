# Pagamentos do TCE-RJ (Entrega 4, parte 1)

Data: 2026-09-26. Abre a Entrega 4 ("Anunciado × pago"). O Diário diz o
que foi contratado; o TCE-RJ diz o que foi empenhado, liquidado e pago a
cada credor. Esta parte traz os pagamentos para a página da empresa, os
painéis e o MCP. A parte 2 são os padrões que cruzam os dois lados.

## Fatos que orientam o desenho

- API de dados abertos do TCE-RJ, sem chave:
  `https://dados.tcerj.tc.br/api/v1/empenho_municipio?ano=AAAA&municipio=SAO%20GONCALO&inicio=0&limite=1000000&csv=true`.
  CSV com `;`, UTF-8 com BOM, cabeçalho `Ente;Unidade;Ano;Mes;NumeroEmpenho;TipoPessoa;CPFCNPJ;Funcao;Empenhado;Liquidado;Pago`,
  decimais com ponto. Cada linha é um empenho num mês: o que foi
  empenhado, liquidado e pago naquele mês (há valores negativos, de
  anulação).
- Cobertura para São Gonçalo: 2020 a agosto de 2026 (2010 a 2019 voltam
  vazios). Um ano leva de 15 a 35 segundos e tem de 4.604 (2020) a 17.857
  (2024) linhas; 68.583 no total.
- `TipoPessoa`: `JURÍDICA` (57.440 linhas, CNPJ de 14 dígitos), `FÍSICA`
  (11.129, CPF inteiro) e vazio (14).
- Unidades: Prefeitura, fundos (saúde, educação, assistência), fundações,
  IPASG e Câmara. O TCE não traz número de processo nem de contrato.
- Em 2025, 656 CNPJs receberam pagamento; 512 deles são citados no
  Diário e ficaram com 92% do valor pago a pessoas jurídicas.
- O portal antigo da Prefeitura (portaltp) tem pagamentos de 2017 a 2022
  por mês, com retenções e bloqueios misturados; fica para depois.

## Decisões

1. **Job no módulo da API** (`cmd/tce`, `make tce [FROM=AAAA TO=AAAA]`),
   como os das partes anteriores. Padrão: o ano corrente e o anterior (o
   TCE atualiza meses passados). Agendamento semanal, domingo às 05:00.
   Fonte `tce_empenhos` em `fetch_runs`.
2. **Só pessoa jurídica.** Linhas `FÍSICA` e sem tipo são descartadas na
   leitura (CPF inteiro; ADR 0006).
3. **Todas as pessoas jurídicas**, citadas ou não no Diário: o padrão
   "pago sem publicação" precisa das que não aparecem.
4. **Tabela `payments`** (migration 016): fonte, ano, mês, unidade,
   empenho, CNPJ, função, empenhado, liquidado e pago em centavos. Chave
   `(source, year, month, unit, commitment, cnpj)`. A carga troca o ano
   inteiro da fonte numa transação.
5. **Ligações** (ADR 0004): uma por CNPJ citado e ano
   (`record_kind = pagamentos_ano`, `record_id = AAAA`, certeza `exata`,
   evidência com o valor pago), refeitas a cada carga do ano.
6. **Arquivo bruto:** as linhas de pessoa jurídica, com o cabeçalho, em
   `raw/tce_empenhos/AAAA/<ano>.csv.gz` (pasta do dia da carga), e o
   SHA-256 da resposta inteira no manifesto.
7. **Página da empresa:** "Pagamentos (TCE-RJ)" com uma linha por ano:
   empenhado, liquidado e pago, e as unidades que pagaram. Aviso da
   cobertura (2020 em diante) e de que o TCE não diz a que contrato cada
   pagamento se refere.
8. **Painéis:** coluna "Pago (TCE-RJ)" por fornecedor, no ano do filtro
   ou no total; o filtro de secretaria não se aplica ao pago (as unidades
   do TCE são fundos e a Prefeitura, não as secretarias do Diário), e a
   página diz isso. Nos totais por ano, o pago a pessoas jurídicas que
   não são órgãos públicos.
9. **MCP:** `entidade` traz `pagamentos_tce` por ano.
10. **Nuvem:** job, agendamento e conta de serviço no Terraform.

## Fora do escopo

- Portal antigo (2017–2022) e o novo (Embras).
- Pagamentos a pessoa física, folha.
- Ligação de pagamento a contrato (o TCE não traz o número).

## Testes

- Domínio: linha de pessoa jurídica, física e vazia; decimais e valor
  negativo; cabeçalho sem coluna.
- Adapter: CSV com BOM contra `httptest`; resposta vazia (só cabeçalho).
- Caso de uso: física fora, troca por ano, falha num ano registrada.
- Integração: carga de dois anos, recarga do mesmo ano troca as linhas,
  página da empresa e painel com o pago.
- Carga real 2020–2026 e conferência do total de um ano contra a soma do
  CSV.

**Pronto quando:** a página de um fornecedor citado mostra o pago por ano
e os painéis mostram o pago ao lado do contratado.
