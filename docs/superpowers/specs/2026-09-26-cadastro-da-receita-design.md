# Cadastro da Receita (Entrega 3, parte 1)

Data: 2026-09-26. Primeira fonte externa do roadmap ("Quem é o
fornecedor"). A Entrega 3 foi dividida em três partes, cada uma com spec
e plano próprios:

1. **Cadastro da Receita** (esta): empresa, estabelecimento e sócios dos
   CNPJs citados nos Diários; nome da empresa na página e nos painéis.
2. **Sanções da CGU**: CEIS e CNEP.
3. **Padrões novos**: empresa aberta pouco antes do contrato, capital
   social menor que o contratado, sócio ou endereço em comum, sancionado
   contratado. Dependem das partes 1 e 2.

## Fatos que orientam o desenho

- Os dados abertos do CNPJ estão num compartilhamento público do
  Nextcloud da Receita, lido por WebDAV:
  `https://arquivos.receitafederal.gov.br/public.php/webdav/AAAA-MM/`,
  com o token do compartilhamento como usuário e senha vazia. O endereço
  antigo (`dadosabertos.rfb.gov.br`) não responde.
- Cada mês tem cerca de 7,9 GB em zip: `Empresas0..9` (1,4 GB),
  `Estabelecimentos0..9` (5,4 GB), `Socios0..9` (0,7 GB) e as tabelas de
  código (`Cnaes`, `Municipios`, `Naturezas`, `Qualificacoes`, `Motivos`,
  `Paises`). O download sai a ~2 MB/s: o mês inteiro leva cerca de uma
  hora.
- O servidor aceita `Range` (responde 206), e cada zip tem um arquivo
  só: CSV com `;`, aspas, latin-1, sem cabeçalho.
- O CPF do sócio pessoa física já vem mascarado pela Receita
  (`***123456**`).
- Os Diários citam 4.394 CNPJs distintos (`entities`, `kind = cnpj`).

## Decisões

1. **Carga mensal filtrada, num job do módulo da API** (`cmd/receita`,
   `make receita [MONTH=AAAA-MM]`). O filtro depende dos CNPJs que estão
   na base, então o job lê o banco, ao contrário do coletor do Diário
   (ADR 0005). Registrar em ADR 0008.
2. **Leitura sem disco.** Cada zip é lido por `Range` através de um
   `io.ReaderAt` com leitura antecipada em blocos de 16 MB, o que basta
   para o `archive/zip` achar o índice no fim do arquivo e depois
   descompactar em sequência. Cada pedido tenta três vezes antes de
   falhar.
3. **Filtro:**
   - empresa e sócios pelo CNPJ básico (8 primeiros dígitos) dos CNPJs
     citados;
   - estabelecimento só o CNPJ exato citado.
4. **Tabelas** (migration 014), só com o mês mais recente:
   - `rf_companies`: CNPJ básico, razão social, natureza jurídica,
     capital social em centavos, porte, mês de referência;
   - `rf_establishments`: CNPJ, matriz ou filial, nome fantasia, situação
     cadastral e data, motivo, data de abertura, CNAE principal e
     secundários com descrição, endereço (logradouro, número,
     complemento, bairro, CEP, município, UF);
   - `rf_partners`: CNPJ básico, tipo (pessoa jurídica, física ou
     estrangeiro), nome, documento como a Receita publica, qualificação,
     data de entrada.
   Telefone e e-mail ficam de fora: não servem aos padrões e podem ser
   pessoais (MEI).
5. **Troca atômica.** O job lê tudo, e só então, numa transação,
   substitui as três tabelas e as ligações da fonte. Falha em qualquer
   arquivo não troca nada.
6. **Ligações** (ADR 0004): cada estabelecimento carregado liga o CNPJ
   (`source = receita_cnpj`, `record_kind = estabelecimento`, certeza
   `exata`).
7. **Coleta registrada** em `fetch_runs` (`source = receita_cnpj`):
   encontrados = CNPJs procurados, gravados = estabelecimentos achados,
   pulados = procurados sem registro, com a mensagem de erro se falhar.
8. **Arquivo bruto**: as linhas que passaram no filtro, por arquivo de
   origem, em `raw/receita_cnpj/AAAA/MM/01/<arquivo>.csv.gz`, com o
   SHA-256 do CSV de origem (descompactado) e o da extração gravados. O dump inteiro não
   é guardado.
9. **Página da empresa** ganha "Cadastro na Receita":
   - razão social no título, nome fantasia, situação com data (em
     destaque quando não é "ativa"), abertura, natureza jurídica, porte,
     capital social, CNAE principal e secundários, endereço;
   - sócios com qualificação e data de entrada, o documento como a
     Receita publica (ADR 0006: sócio só dentro da página da empresa);
   - "Dados da Receita de <mês>". Sem registro: "CNPJ não encontrado no
     cadastro da Receita de <mês>" (CNPJ com erro de digitação no Diário
     cai aqui).
10. **API e MCP**: `GET /v1/entities/cnpj/{cnpj}` e a ferramenta
    `entidade` trazem `registry` (nulo quando não há). O dump não muda:
    sócios não entram em lista pública (ADR 0006).
11. **Nome da empresa nos painéis**: `name` em cada fornecedor, pela
    razão social; sem cadastro, fica só o CNPJ. Fecha a pendência "Nome
    da empresa" dos painéis.
12. **Nuvem**: job e agendamento mensal (dia 20, depois da publicação
    habitual da Receita) no Terraform, como o `dump`. A aplicação fica na
    pendência de nuvem que já existe.

## Fora do escopo

- Histórico mensal do cadastro (só o mês mais recente).
- Simples Nacional, telefone, e-mail, representante legal do sócio.
- Filiais não citadas no Diário.
- Página ou busca por sócio (ADR 0006).

## Testes

- Domínio: leitura de cada tipo de linha com linhas reais (nomes
  trocados), latin-1, datas vazias (`0`, `00000000`), capital com
  vírgula.
- Adapter: `ReaderAt` por `Range` contra `httptest`, com um zip gerado no
  teste; filtro por CNPJ básico e exato.
- Integração: a carga grava as tabelas e as ligações, troca o mês
  anterior, e `/v1/entities/cnpj/{cnpj}` e `/v1/panels/suppliers` trazem
  o cadastro e o nome.
- Web: formatação da situação, do capital e do endereço; seção vazia sem
  cadastro.
- Carga real local do mês 2026-09, conferindo a razão social de 20
  empresas contra o nome citado no Diário (o comprovante de inscrição do
  site da Receita tem captcha e fica de fora).

**Pronto quando:** a página da empresa mostra o cadastro de pelo menos
95% dos CNPJs válidos citados, e os painéis mostram o nome.

## Depois da entrega

- Carga local de 2026-09: 83 minutos, 4.196 de 4.394 CNPJs (4.196 de
  4.197 com dígito verificador válido), 7.123 sócios. Na amostra de 20, a
  razão social bate com o nome citado no Diário em 18; nas outras 2 o
  trecho em volta do CNPJ não traz o nome.
- A primeira carga local falhou no fim porque a migration 014 não estava
  aplicada: a troca atômica funcionou (nada gravado), mas a hora de
  leitura se perdeu. Na nuvem, a migration vai antes do job.
- O SHA-256 guardado é o do CSV descompactado, num `manifest.json` ao lado
  das extrações: o índice do zip fica no fim e a leitura por `Range` não
  passa pelos bytes em ordem.
