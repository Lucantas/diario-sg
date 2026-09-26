# Dinheiro federal (Entrega 5, parte 2)

Data: 2026-09-26. Item "Transferências federais e emendas destinadas ao
município" da Entrega 5. Traz, do Portal da Transparência (CGU), as
transferências da União para São Gonçalo e as emendas parlamentares com
aplicação no município ou pagas a quem está nele.

## Fatos que orientam o desenho

Arquivos baixados em 26/09/2026 de
`https://portaldatransparencia.gov.br/download-de-dados/`, todos zip com
CSV `;`, latin-1 e cabeçalho:

- `emendas-parlamentares/UNICO` (32 MB, atualizado todo dia) traz três
  CSVs. `EmendasParlamentares.csv`: uma linha por emenda e ação, com ano,
  tipo, autor, localidade (`Código Município IBGE`; São Gonçalo é
  3304904), função, programa, ação e valores empenhado, liquidado, pago e
  de restos a pagar. São Gonçalo: 73 linhas (2014 a 2026), R$ 68,1 milhões
  pagos; 71 de emenda individual e 2 de relator. Autores são
  parlamentares (agentes políticos); 9 linhas vêm "Sem informação".
  `EmendasParlamentares_PorFavorecido.csv`: cada pagamento de emenda por
  mês e favorecido (CNPJ ou CPF, natureza jurídica, UF e município).
  Favorecidos em São Gonçalo: 761 pessoas jurídicas (R$ 274,9 milhões,
  incluindo a Prefeitura, o Fundo de Saúde e empresas locais) e 27
  pessoas físicas.
- `transferencias/AAAAMM` (2,5 MB por mês): cada transferência do mês por
  município, tipo (constitucionais, legais e voluntárias…), órgão,
  função, programa, ação, "linguagem cidadã", favorecido e valor. São
  Gonçalo em janeiro de 2026: 31 linhas, R$ 60,9 milhões, para o
  Município, os fundos de saúde e de assistência e duas entidades.

## Decisões

1. **Job semanal `federal`** no módulo da API (`cmd/federal`,
   `make federal`), domingo às 08:00: lê o arquivo de emendas inteiro e as
   transferências dos três últimos meses publicados (`-from`/`-to`
   `AAAAMM` para outra faixa). Fontes `cgu_emendas` e
   `cgu_transferencias` em `fetch_runs`.
2. **Só pessoa jurídica** nos favorecidos: linhas com CPF são descartadas
   na leitura, sem chegar ao banco nem ao bucket.
3. **Tabelas** (migration 020): `federal_amendments` (emendas com
   localidade São Gonçalo, trocadas a cada carga),
   `federal_amendment_payments` (pagamentos de emenda a pessoa jurídica de
   São Gonçalo, trocados a cada carga) e `federal_transfers` (trocadas
   por mês).
4. **Arquivo bruto:** as linhas filtradas, com cabeçalho, em
   `raw/cgu_emendas/…` e `raw/cgu_transferencias/…`, e o manifesto com o
   SHA-256 do zip.
5. **Ligações:** o pagamento de emenda liga à empresa pelo CNPJ
   (`exata`).
6. **Página `/federal`** ("Dinheiro federal"): transferências por ano e
   tipo; emendas com aplicação em São Gonçalo por ano, com autor, ação e
   valores; favorecidos de emendas em São Gonçalo, somados por CNPJ, com
   link para a página da empresa. Link na página inicial.
7. **Página da empresa:** "Emendas parlamentares (CGU)" com os pagamentos
   de emenda à empresa (mês, autor, valor), em `amendment_payments` de
   `/v1/entities/cnpj/{cnpj}`; a ferramenta `entidade` traz
   `emendas_pagas_cgu`.
8. **API:** `GET /v1/federal` devolve os três blocos.

## Fora do escopo

- Convênios (`EmendasParlamentares_Convenios.csv` e a base de convênios).
- Página do parlamentar: o autor aparece só como texto.
- Transferências anteriores a 2021 (a carga aceita a faixa, mas o padrão
  é o recente).

## Testes

- Domínio: leitura de uma linha real de cada CSV, filtro de São Gonçalo,
  descarte de CPF, valores com vírgula.
- Adapter contra `httptest`: redirecionamento e zip com vários CSVs.
- Caso de uso com fakes: troca, bruto e manifesto, falha não grava.
- Integração: carga, `GET /v1/federal`, empresa com pagamento de emenda.
- Carga real local e conferência dos totais contra os CSVs.

## Depois da entrega

- Carga local das emendas em 11 segundos: 73 emendas com aplicação em São
  Gonçalo (R$ 68,1 milhões pagos) e 761 pagamentos a 167 pessoas
  jurídicas de São Gonçalo (R$ 274,9 milhões), os mesmos totais dos CSVs;
  342 pagamentos ligados a empresas citadas no Diário.
- A carga de transferências de janeiro de 2021 em diante parou em julho
  de 2023: depois de cerca de 30 downloads seguidos, o Portal passou a
  responder com verificação humana (AWS WAF, status 405). O adapter
  passou a esperar 20 segundos entre downloads e a dizer que é
  verificação humana; o job semanal baixa só três meses. Na base local,
  1.714 transferências de janeiro de 2021 a junho de 2023 (R$ 1,84
  bilhão); a página mostra os meses carregados.
- A página mostra os 30 favorecidos que mais receberam; o Município e os
  fundos municipais estão entre eles, porque são pessoas jurídicas de São
  Gonçalo.

**Pronto quando:** `/federal` mostra as transferências e as emendas de São
Gonçalo, e a página de uma empresa favorecida mostra os pagamentos.
