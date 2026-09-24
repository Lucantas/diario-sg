# Alerta por entidade (Entrega 2)

Data: 2026-09-24. Parte da Entrega 2 de `docs/roadmap.md` ("Seguir o
dinheiro dentro do Diário"): inscrição em CNPJ, processo ou contrato,
além de termo livre.

## Objetivo

Quem acompanha um fornecedor, um processo ou um contrato recebe um e-mail
no dia em que uma edição nova cita aquele número, ou assina um feed RSS
com os atos que o citam. A ligação é a mesma das páginas de entidade
(`entity_links`), por chave exata, sem busca textual.

## Fora do escopo

- Painel "meus alertas" e login: cada alerta continua sendo um e-mail
  com links de confirmação e de cancelamento.
- Filtro por órgão no alerta de contrato. O aviso de número em mais de um
  órgão fica na página; o e-mail traz o órgão de cada ato.
- Filtro de entidade na tela de busca e na ferramenta `buscar_atos` do
  MCP. A API aceita o filtro; a tela e o MCP ficam para quando pedirem.

## Fatos que orientam o desenho

- `subscriptions` guarda `email` e `query`; o worker (`MatchSubscriptions`)
  roda `SearchInGazette` com o texto para cada edição nova.
- A indexação chama `link_diario_acts` na mesma transação que grava os
  atos, antes de publicar `gazette.indexed`. Quando o worker procura os
  alertas, as ligações da edição já existem.
- Buscar o número como texto erra: CNPJ e processo aparecem com e sem
  pontuação, contrato com zeros à esquerda. `entities.key` já é a forma
  normalizada, a mesma das páginas.
- `ParseEntityInput` valida e normaliza CNPJ, processo e contrato, e
  aceita o slug da página (`-` no lugar de `/`).
- O RSS usa o mesmo `ActFilter` da busca, que não filtra por entidade.

## Decisões

1. **Alerta de entidade dispara por `entity_links`**, não por texto: o
   ato entra no e-mail quando a edição tem uma ligação para
   `entities(kind, key)`, de qualquer certeza. Vale para os dois diários.
2. **A entidade não precisa existir** na base para a inscrição: dá para
   acompanhar um CNPJ antes da primeira aparição. Basta ter formato
   válido.
3. **O rótulo vem da entrada, limpo pelo servidor**: CNPJ formatado;
   processo e contrato pelo `EntityLabel`, que só deixa o número. Nenhum
   texto livre do usuário vai no e-mail de entidade.
4. **Filtro `entity=<tipo>:<número>` no `ActFilter`**, aceito pela busca,
   pela exportação e pelo RSS (`/v1/feeds/acts?entity=cnpj:…`). O link do
   canal do feed de entidade aponta para a página da entidade.
5. **Trecho no e-mail em volta da menção**: o e-mail mostra o título do
   ato e um trecho do corpo centrado no texto que gerou a ligação
   (`entity_links.evidence`), com o número marcado.

## Desenho

### Dados

Migration `012_subscription_entity.sql`:

```sql
ALTER TABLE subscriptions
    ALTER COLUMN query DROP NOT NULL,
    ADD COLUMN entity_kind  text CHECK (entity_kind IN ('cnpj', 'processo', 'contrato')),
    ADD COLUMN entity_key   text,
    ADD COLUMN entity_label text,
    ADD CONSTRAINT subscriptions_query_or_entity CHECK (
        (query IS NOT NULL AND entity_kind IS NULL AND entity_key IS NULL AND entity_label IS NULL)
        OR (query IS NULL AND entity_kind IS NOT NULL AND entity_key IS NOT NULL AND entity_label IS NOT NULL)
    );
```

### Domínio

- `EntityRef{Kind, Key, Label}` e `ParseEntityRef(kind, value)`: valida
  com `ParseEntityInput` e monta o rótulo (`FormatCNPJ` ou `EntityLabel`).
  Só `cnpj`, `processo` e `contrato`.
- `ParseEntityFilter("cnpj:…")`: separa no primeiro `:` e chama
  `ParseEntityRef`; erro vira `ErrInvalidFilter`.
- `Subscription.Entity *EntityRef`; `NewEntitySubscription(email, ref,
  now)`. `Subscription.Subject()`: `“merenda”` para termo, `CNPJ
  12.345.678/0001-90`, `processo 12345/2024` ou `contrato 12/2024` para
  entidade.
- `ActFilter.Entity *EntityRef`.
- `MentionSnippet(body, evidence)`: até 280 caracteres em volta da
  primeira ocorrência de `evidence`, com `⟦ ⟧` em volta dela; sem
  ocorrência, o começo do corpo.

### Casos de uso e portas

- `Subscriptions.SubscribeEntity(ctx, email, kind, value)`.
- `ActRepository.EntityHitsInGazette(ctx, gazetteID, ref)`: atos da
  edição ligados à entidade, na ordem da edição, até 20, com o trecho.
- `MatchSubscriptions` escolhe `EntityHitsInGazette` quando
  `s.Entity != nil` e `SearchInGazette` quando não.

### Postgres

- `filterSQL`: `AND a.id IN (SELECT l.record_id::uuid FROM entities e
  JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = 'ato'
  WHERE e.kind = $n AND e.key = $m)`.
- `SubscriptionRepo` lê e grava as três colunas novas.

### HTTP

- `POST /v1/subscriptions` aceita `{email, query}` (como hoje) ou
  `{email, entity: {kind, value}}`; os dois juntos ou nenhum é 400.
- A resposta de inscrição e de confirmação ganha `subject` e `entity`
  (`{kind, key, label}`); `query` fica vazio no alerta de entidade.
- `filterFromQuery` lê `entity`. No feed de entidade, o título do canal
  é `Diário SG: <subject>` e o link é a página da entidade
  (`/empresa/<cnpj>`, `/processo/<slug>`, `/contrato/<slug>`).

### E-mail

- Confirmação e resultados usam `Subject()`. O assunto do e-mail de
  resultados é `<Subject com inicial maiúscula> no Diário da <fonte> de
  <dd/mm>`, igual ao de hoje para termo.
- No alerta de entidade, cada ato traz o órgão quando houver.

### Front

- Componente `EntityAlert` nas páginas de empresa, processo e contrato:
  formulário de e-mail ("Avisar quando <rótulo> aparecer de novo") e o
  link do feed (`entityFeedUrl(kind, value)`).
- Página de confirmação: "Você será avisado quando houver ato novo sobre
  <subject>."

## Testes

- Domínio: `ParseEntityRef` (formatos aceitos, rótulo, tipo inválido),
  `ParseEntityFilter`, `NewEntitySubscription`, `Subject()`,
  `MentionSnippet` (meio, começo, fim do corpo, sem ocorrência,
  caracteres acentuados).
- Casos de uso: `MatchSubscriptions` chama a busca por entidade para
  inscrição de entidade e a textual para termo; não repete edição já
  enviada.
- HTTP: inscrição com entidade, com os dois campos, com nenhum; filtro
  `entity` inválido é 400; link do canal do feed de entidade.
- E-mail: assunto e corpo do alerta de entidade.
- Integração (Postgres real): inscrição de entidade gravada e lida;
  `EntityHitsInGazette` traz só os atos ligados; busca e feed com
  `entity` trazem só os atos que citam o número, nos dois formatos de
  contrato (`012/2024` e `12/2024`).
- Front (Vitest): `entityFeedUrl` e o texto de confirmação.
