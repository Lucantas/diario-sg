# Roadmap — de buscador do Diário a ferramenta de investigação

Público: jornalistas, mandatos e ativistas que investigam a Prefeitura de São
Gonçalo. Restrição de projeto: **nada depende de IA** (sem LLM, embeddings ou
classificação estatística). Todo cruzamento é feito por chave determinística
(CNPJ, nº de processo, nº de contrato, CPF parcial + nome) e todo alerta é uma
regra escrita, que mostra a regra e os atos que a acionaram.

A ideia central: o Diário diz o que foi **anunciado**. A investigação nasce do
cruzamento com o que foi **pago**, com **quem é dono** do fornecedor, com
**quem doou** para quem e com **quem foi punido**. `act_entities` (migration
003) já é o embrião disso; as entregas abaixo transformam as chaves extraídas
no centro do produto.

| Entrega | Tema | Fontes novas | Depende de |
| --- | --- | --- | --- |
| 0 | Base completa | — | — |
| 1 | Citável e exportável | — | 0 |
| 2 | Seguir o dinheiro dentro do Diário | — | 1 |
| 3 | Quem é o fornecedor | Receita (CNPJ), CGU (CEIS/CNEP/CEPIM) | 2 |
| 4 | Anunciado × pago | PNCP, Portal da Transparência de SG | 3 |
| 5 | Recorte político | TSE, Câmara Municipal, transferências federais, TCE-RJ | 3 |

Disponibilidade, formato e licença de cada fonte externa **ainda não foram
verificados**; o primeiro passo das entregas 3 a 5 é um levantamento curto
(`docs/fontes/<fonte>.md`: URL, formato, frequência de atualização, chaves,
limites de uso).

---

## Entrega 0 — Base completa

O scraper perde as **edições extraordinárias**. O site publica a extra do
mesmo dia em `diario/AAAA_MM_DD_1.pdf`, e `editionLinkRe`
(`services/scraper/internal/adapters/source/pmsg/source.go`) só aceita
`AAAA_MM_DD.pdf`; o domínio também assume uma edição por data. Exemplo: em
21/08/2024 a base tem a edição 1196, mas não a extra 1197. Na coleta do
Querido Diário até 30/08/2024 há 46 edições extras de 2020 em diante.

- Aceitar o sufixo `_N` no link e tratar a edição pela URL, não pela data.
- Gravar `is_extra` em `gazettes` (o card do site traz "Diário Oficial
  Extraordinário").
- Backfill das extras de 2020 a hoje e reindexação.

**Pronto quando:** a contagem de edições por ano bate com a listagem do site.

## Entrega 1 — Citável e exportável

Só com os dados atuais. É o que dá credibilidade para quem vai publicar.

- **Proveniência.** Guardar a página de cada ato (hoje `clean.go` troca `\f`
  por `\n` e a informação se perde) e linkar `…pdf#page=N`. Mostrar o hash do
  PDF e servir a cópia arquivada no nosso bucket (a prefeitura pode tirar o
  arquivo do ar). Botão "citar este ato" com edição, data, página e link.
- **Órgão.** Persistir a sigla que `isOrganSection` já reconhece e expor
  filtro por secretaria.
- **Busca de investigador.** Faixa de valor (a partir de `act_entities`),
  operadores booleanos, URL permanente para cada consulta.
- **Exportação.** CSV/JSON de qualquer busca e dump completo periódico
  (CSV/Parquet e um SQLite pronto para o Datasette), publicado no bucket.
- **RSS** por consulta, ao lado do alerta por e-mail.
- **Qualidade visível.** Marcar no ato os casos conhecidos do parser
  (`docs/fase-1-relatorio.md` §2.3: portaria sem número, tabela quebrada entre
  páginas) e um botão "reportar erro" que abre uma fila de correção.

**Pronto quando:** um jornalista consegue sair de uma busca com um CSV e uma
citação que aponta para a página exata do PDF arquivado.

## Entrega 2 — Seguir o dinheiro dentro do Diário

Ainda sem fonte externa.

- **Página de processo e de contrato.** `/processo/{n}` e `/contrato/{n}`
  juntam, em ordem, licitação → homologação → extrato → aditivos → rescisão,
  pelo nº extraído.
- **Alerta por entidade.** Inscrição em CNPJ, processo ou contrato, além de
  termo livre.
- **Painéis.** Maiores fornecedores por valor, por secretaria e por ano.
- **Padrões para verificar** (regras fixas, cada uma com a regra em texto e
  os atos que a acionaram):
  - várias dispensas para o mesmo fornecedor/objeto abaixo do limite legal
    numa janela curta (possível fracionamento);
  - aditivos que somam mais de 25% do valor original do contrato;
  - contratação emergencial renovada seguidamente;
  - picos de nomeação e exoneração nos meses antes das eleições (2020 e 2024
    estão na base, dá para comparar com anos sem eleição).

Linguagem: sempre "padrão para verificar", nunca "irregularidade".

**Pronto quando:** cada regra tem teste com atos reais que a acionam e atos
reais parecidos que não acionam.

## Entrega 3 — Quem é o fornecedor

Primeira fonte externa. Começa por uma mudança de modelo: uma tabela de
**entidades** (CNPJ, processo, contrato e, na entrega 5, agente político) que
funciona como ponto de ligação; cada fonte tem adapter e tabelas próprias,
ligadas às entidades. O Diário continua sendo documento com busca textual; as
outras fontes são registros estruturados. Registrar em ADR.

- **Receita Federal — CNPJ (dados abertos).** Situação, data de abertura,
  CNAE, capital social, endereço, quadro de sócios. Carga mensal só dos CNPJs
  que aparecem no Diário.
- **CGU — CEIS, CNEP, CEPIM.** Sanções vigentes e históricas.
- **Página da empresa** enriquecida com tudo acima.
- **Novos padrões:** empresa aberta pouco antes do primeiro contrato; capital
  social muito menor que o valor contratado; fornecedores com sócio ou
  endereço em comum; fornecedor sancionado contratado.
- **ADR de LGPD** antes de publicar sócios: agentes políticos e secretários
  podem ter página própria; servidores são encontráveis na busca, sem perfil;
  sócios aparecem só dentro da página da empresa.

## Entrega 4 — Anunciado × pago

- **PNCP** (contratações do município, pela Lei 14.133) e **Portal da
  Transparência de SG** (empenho, liquidação, pagamento).
- Ligação por CNPJ + nº de contrato/processo, com o grau de certeza de cada
  ligação visível.
- **Novos padrões:** pagamento a fornecedor sem contrato publicado no Diário;
  contrato publicado sem nenhum pagamento; pago acima do contratado mais
  aditivos.
- Folha de pagamento só em agregados por cargo/órgão (LGPD).

## Entrega 5 — Recorte político

- **TSE:** candidatos, bens e prestação de contas (doações). Cruzamento sócio
  de fornecedor × doador. A Receita mascara parte do CPF dos sócios, então o
  casamento é nome + dígitos visíveis: determinístico, mas a página mostra a
  evidência do casamento e nunca afirma identidade.
- **Câmara Municipal:** leis, projetos e votações; perfil de vereador.
- **Transferências federais e emendas** destinadas ao município.
- **TCE-RJ:** apontamentos sobre contratos já presentes no Diário.

---

## Fora do produto: Querido Diário

O Querido Diário (Open Knowledge Brasil) não coleta São Gonçalo. O PR
[okfn-brasil/querido-diario#1256](https://github.com/okfn-brasil/querido-diario/pull/1256)
(2024) traz um spider aprovado por um revisor voluntário, mas nunca foi
mergeado e ficou desatualizado em relação ao upstream (pasta
`data_collection/` virou `querido_diario_raspadores/`, spiders migraram para
`async def start()`) e o número da edição deixou de ser extraído porque o site
passou a escrever `N°1.771` sem espaço. Contribuir com a versão atualizada põe
os diários de SG na base nacional e na API deles.
