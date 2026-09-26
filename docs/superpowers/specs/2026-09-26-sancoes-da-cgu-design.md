# Sanções da CGU (Entrega 3, parte 2)

Data: 2026-09-26. Segunda fonte externa. Traz para a página da empresa as
sanções do CEIS (empresas inidôneas e suspensas) e do CNEP (empresas
punidas pela Lei Anticorrupção) aplicadas aos CNPJs citados nos Diários.
A parte 3 (padrões novos) usa estas tabelas.

## Fatos que orientam o desenho

- O Portal da Transparência publica os dois cadastros como zip diário:
  `https://portaldatransparencia.gov.br/download-de-dados/{ceis|cnep}/AAAAMMDD`
  redireciona para `dadosabertos-download.cgu.gov.br`. Só o arquivo do dia
  mais recente está disponível: datas anteriores (conferidas 24/09/2026,
  01/09/2026, 2025, 2022, 2019, 2017) respondem 403. A data mais recente
  está na página `download-de-dados/<cadastro>`
  (`arquivos.push({"ano" : "2026", "mes" : "09", "dia" : "25", …})`).
- Cada zip tem um CSV com `;`, aspas, latin-1 e **com cabeçalho**. O CEIS
  tem 24 colunas e o CNEP 25 (a mais é `VALOR DA MULTA`). Tamanho em
  25/09/2026: CEIS 3,5 MB (23.739 sanções), CNEP 0,2 MB (1.817).
- `TIPO DE PESSOA` é `J` (CNPJ de 14 dígitos), `F` (CPF inteiro, sem
  máscara) ou vazio.
- Cruzado com os 4.394 CNPJs citados: 385 sanções do CEIS e 19 do CNEP
  pelo CNPJ exato; 2 do CEIS a outro estabelecimento da mesma empresa.
- O CEPIM (entidades sem fins lucrativos impedidas) responde 403 desde
  26/09/2026, pela página e pelo arquivo.
- Sanção vencida às vezes continua no cadastro (558 do CEIS com data final
  passada) e às vezes sai. Não há histórico oficial para baixar.

## Decisões

1. **Job diário no módulo da API** (`cmd/sancoes`, `make sancoes`), como o
   `receita` (ADR 0008): o filtro depende dos CNPJs da base. Agendamento
   diário às 07:00. Fonte `cgu_sancoes` em `fetch_runs`.
2. **Só pessoa jurídica.** Linhas `F` e sem tipo são descartadas na
   leitura, sem chegar ao banco nem ao arquivo bruto: o CPF vem inteiro e a
   base não tem página de pessoa (ADR 0006).
3. **Filtro pelo CNPJ básico** dos CNPJs citados: a sanção costuma ser
   registrada na matriz e vale para a empresa. A ligação com o CNPJ citado
   é `exata` quando o CNPJ é o mesmo e `forte` quando é outro
   estabelecimento da mesma empresa.
4. **Histórico que a base acumula.** Tabela `cgu_sanctions` (migration
   015), chave `(register, code)`, com os campos do cadastro e
   `first_seen`/`last_seen` (data do arquivo). A carga faz *upsert*: a
   sanção que sai do cadastro fica, com o último dia em que foi vista.
   "No cadastro hoje" = `last_seen` igual à data do último arquivo daquele
   cadastro.
5. **Troca atômica por carga:** os dois cadastros são lidos e só então,
   numa transação, gravados, e as ligações `cgu_sancoes` refeitas. Falha em
   qualquer um não grava nada. A tabela é conferida antes do download.
6. **Arquivo bruto:** as linhas filtradas, com o cabeçalho, em
   `raw/cgu_sancoes/AAAA/MM/DD/<CEIS|CNEP>.csv.gz`, e um `manifest.json`
   com o SHA-256 do zip baixado e o número de linhas.
7. **Página da empresa** ganha "Sanções (CGU)": para cada sanção, o
   cadastro, a categoria, o órgão sancionador (UF e esfera), a
   abrangência, o período, o processo, a fundamentação e a multa (CNEP);
   "Em vigor" quando está no cadastro e o período cobre hoje; "Encerrada"
   quando a data final passou; "Saiu do cadastro em <data>" quando não
   está no último arquivo. Aviso quando a sanção é de outro
   estabelecimento. Sem sanção: "Nenhuma sanção no CEIS nem no CNEP em
   <data>".
8. **API e MCP:** `GET /v1/entities/cnpj/{cnpj}` traz `sanctions` e
   `sanctions_as_of`; a ferramenta `entidade` traz `sancoes_cgu`. O dump
   não muda.
9. **Nuvem:** job, agendamento e conta de serviço no Terraform, como o
   `receita`.

## Fora do escopo

- CEPIM (403) e acordos de leniência.
- Sanções a pessoas físicas.
- Penalidades do TCE-RJ (Entrega 5).
- O padrão "fornecedor sancionado contratado" (parte 3).

## Testes

- Domínio: leitura de uma linha real de cada cadastro (nomes trocados),
  datas vazias, multa com vírgula, linha `F` recusada; "em vigor" e
  "encerrada".
- Adapter: data mais recente pela página, zip do dia com redirecionamento,
  contra `httptest`.
- Caso de uso: filtro pelo básico, descarte de pessoa física, arquivo
  bruto com manifesto, falha num cadastro não grava nada.
- Integração: duas cargas (a sanção que some fica com `last_seen`
  antigo), ligações `exata`/`forte`, `/v1/entities/cnpj/{cnpj}` com as
  sanções.
- Web: rótulo do estado de cada sanção; seção sem sanção.
- Carga real local e conferência de 10 sanções contra a consulta do
  Portal da Transparência.

**Pronto quando:** a página de uma empresa citada e sancionada mostra as
sanções do CEIS e do CNEP com o estado de cada uma, e a carga diária grava
sem falha.
