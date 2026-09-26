# ADR 0008 — Cadastro da Receita (CNPJ)

**Status:** aceito

## Contexto
A Entrega 3 junta aos Diários o cadastro da Receita: empresa,
estabelecimento e sócios. A Receita publica todo mês o cadastro inteiro
do país (cerca de 7,9 GB em zip) num compartilhamento público do
Nextcloud, lido por WebDAV. Só interessam os CNPJs citados nos Diários
(4.394 em setembro de 2026).

## Decisão
- **Job do módulo da API** (`cmd/receita`), não coletor à parte. O
  filtro depende dos CNPJs que estão em `entities`, então o job lê o
  banco. É a exceção ao coletor sem banco do ADR 0005; o registro em
  `fetch_runs` é gravado direto, sem evento.
- **Leitura por `Range`**, sem guardar o zip em disco: o job lê o índice
  no fim do arquivo e descompacta em sequência.
- **Filtro:** empresa e sócios pelo CNPJ básico; estabelecimento só o
  CNPJ exato citado.
- **Só o mês mais recente.** A carga troca as tabelas `rf_*` e as
  ligações da fonte `receita_cnpj` numa transação; falha em qualquer
  arquivo não troca nada. Sem histórico mensal.
- **Arquivo bruto:** as linhas que passaram no filtro, por arquivo de
  origem, em `raw/receita_cnpj/AAAA/MM/01/<arquivo>.csv.gz`, com o
  SHA-256 do CSV de origem (descompactado; o índice do zip fica no fim
  e a leitura por `Range` não passa pelos bytes em ordem). O dump inteiro da Receita não é guardado.
- **Sócios** seguem o ADR 0006: só dentro da página da empresa, com o
  documento como a Receita publica (CPF mascarado). Telefone e e-mail
  não são carregados. O dump público não inclui as tabelas `rf_*`.

## Consequências
- Um CNPJ citado pela primeira vez só ganha cadastro na carga do mês
  seguinte.
- Mudança de endereço ou de token do compartilhamento pede só nova
  configuração (`RECEITA_SHARE_TOKEN`).
- A carga leva cerca de uma hora (download a ~2 MB/s).
