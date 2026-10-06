# Deploy grátis

O Diário SG no ar sem cartão de crédito: coleta e cargas no GitHub
Actions, API e site no Render, banco no Neon. O desenho e as escolhas
estão em `docs/superpowers/specs/2026-10-06-deploy-gratis-design.md`. O
Terraform em `infra/` continua descrevendo o deploy no GCP, para quando
houver conta.

| Peça | Onde roda | Plano |
| --- | --- | --- |
| Coleta (scraper + worker) | GitHub Actions, `diario-prefeitura.yml` e `diario-camara.yml` | grátis em repositório público |
| Cargas auxiliares | GitHub Actions, `carga-*.yml` | idem |
| Dump semanal | GitHub Actions, `dump.yml`, publicado no branch `dados` | idem |
| Migrations | GitHub Actions, `migrate.yml` (push em `services/api/migrations/**` ou manual) | idem |
| API e MCP | Render, serviço `diario-sg-api` (Docker) | free, dorme após 15 min sem tráfego |
| Site | Render, static site `diario-sg-web` | free |
| Banco | Neon, projeto `diario-sg` | free, 1 GB por projeto |
| E-mail | Resend | free |

Sem bucket, a API abre o PDF oficial pelo `source_url` da edição, e o
cache do OCR da coleta fica no `actions/cache`.

## 1. Neon

1. Crie o projeto `diario-sg` na região **AWS US East 2 (Ohio)**, perto
   do Render, com Postgres 16.
2. Em Branches → `main` → Compute: autoscaling de 0,25 a 0,25 CU e
   suspensão após 5 minutos. O plano grátis dá 100 CU-hora por mês.
3. Em Connect, copie as duas connection strings, com `sslmode=require`:
   a **pooled** (host com `-pooler`) é o `DATABASE_URL` da coleta, das
   cargas e do Render; a **direta** (sem `-pooler`) é o
   `DATABASE_URL_DIRECT` do `migrate` e do `dump`. O `migrate` segura um
   advisory lock de sessão, que o pooler em modo transação não garante.

## 2. Carga inicial

Na máquina que tem a base local. O `pg_restore` roda dentro do container
do Postgres, então não precisa instalá-lo:

```bash
make up && make migrate
docker compose exec postgres psql -U postgres -d diario -c "VACUUM FULL ANALYZE"
docker compose exec postgres psql -U postgres -d diario -Atc "select pg_size_pretty(pg_database_size('diario'))"
docker compose exec postgres pg_dump -U postgres -d diario -Fc --no-owner --no-privileges -f /tmp/diario.dump
read -rs NEON_DATABASE_URL && export NEON_DATABASE_URL
docker compose exec -e NEON_DATABASE_URL postgres sh -c 'pg_restore --no-owner --no-privileges -d "$NEON_DATABASE_URL" /tmp/diario.dump'
```

Em 06/10/2026 a base local ficou com 715 MB depois da migration 034 e do
`VACUUM FULL`, e o dump com 78 MB.

O tamanho tem de ficar abaixo de 900 MB. Para a carga, use a connection
string **direta** (sem `-pooler`): o pooler não aceita tudo o que o
`pg_restore` faz. O dump traz as extensões `pg_trgm` e `unaccent`, que o
Neon aceita. Se alguma falhar por permissão, crie antes no SQL Editor do
Neon (`CREATE EXTENSION pg_trgm; CREATE EXTENSION unaccent;`) e rode o
`pg_restore` de novo.

## 3. GitHub

1. Em Settings → Environments, crie o ambiente `producao`.
2. Secrets do ambiente:
   - `DATABASE_URL` (pooled, do passo 1);
   - `DATABASE_URL_DIRECT` (direta, do passo 1);
   - `RESEND_API_KEY` e `EMAIL_FROM` (sem eles, os alertas só vão para o
     log da coleta).
3. Variável do ambiente: `PUBLIC_WEB_URL`, a URL final do site.

Pela linha de comando, o `gh` pede o valor sem mostrá-lo:

```bash
gh secret set DATABASE_URL -R Lucantas/diario-sg -e producao
gh secret set DATABASE_URL_DIRECT -R Lucantas/diario-sg -e producao
gh secret set RESEND_API_KEY -R Lucantas/diario-sg -e producao
gh secret set EMAIL_FROM -R Lucantas/diario-sg -e producao
gh variable set PUBLIC_WEB_URL -R Lucantas/diario-sg -e producao --body https://SEU-DOMINIO
```

Os agendamentos rodam no branch padrão: só começam depois do merge na
`main`.

## 4. Render

1. New → Blueprint → este repositório. O Render lê o `render.yaml`.
2. Preencha as variáveis marcadas `sync: false` do `diario-sg-api`:
   `DATABASE_URL`, `PUBLIC_WEB_URL`, `RESEND_API_KEY`, `EMAIL_FROM`.
3. Depois do primeiro deploy, confira `https://diario-sg-api.onrender.com/healthz`
   (responde `{"status":"ok"}`). Se o Render der outro endereço para a API (nome já
   usado), troque `diario-sg-api.onrender.com` nas regras do
   `diario-sg-web` no `render.yaml` e faça push.

A primeira requisição depois de 15 minutos parados acorda a API e leva
cerca de um minuto.

## 5. Domínio e e-mail

1. No Render, em `diario-sg-web` → Settings → Custom Domains, adicione o
   domínio e crie no DNS o registro que ele indicar (CNAME para
   `diario-sg-web.onrender.com`, ou A/ALIAS no domínio raiz).
2. No Resend, adicione o domínio e crie no DNS os registros que ele
   mostrar (SPF, DKIM). `EMAIL_FROM` passa a ser um endereço desse
   domínio, por exemplo `Diário SG <alertas@SEU-DOMINIO>`.
3. Atualize `PUBLIC_WEB_URL` no Render e no ambiente `producao` do
   GitHub. Os links dos e-mails e o OAuth do MCP usam essa URL.

## 6. Limite por IP

A API conta os pedidos por IP lendo o `X-Forwarded-For` na posição
`TRUSTED_PROXY_HOPS`, contada da direita (`render.yaml` começa com `1`).
Com o site no ar, faça uma requisição pelo domínio, veja no log da API
quais endereços chegam no cabeçalho e ajuste o número para a posição do
IP do cliente. `0` volta ao comportamento antigo (primeiro endereço, que
o cliente pode forjar).

A API tem dois caminhos públicos: pelo site (o rewrite do Render pode
somar um proxy) e direto em `diario-sg-api.onrender.com`. Um número fixo
só serve aos dois se ambos chegarem com a mesma quantidade de proxies.
Meça os dois antes de mudar o valor: se o caminho pelo site pedir `2`,
quem chama a API direto consegue forjar o IP com um `X-Forwarded-For`
próprio e escapar dos limites de emissão de chave, registro OAuth e
relatos. Nesse caso, fica pendente confiar em faixas de IP conhecidas do
proxy em vez de contar posições.

## 7. Rodar à mão

```bash
gh workflow run migrate -R Lucantas/diario-sg
gh workflow run diario-prefeitura -R Lucantas/diario-sg -f lookback_days=7
gh workflow run diario-camara -R Lucantas/diario-sg -f lookback_days=7
gh workflow run carga-tce -R Lucantas/diario-sg
gh workflow run dump -R Lucantas/diario-sg
```

## 8. Quando o banco passar de 900 MB

O workflow `dump` mede o banco toda semana e falha acima de 900 MB.
Opções, da mais barata à mais cara: `VACUUM FULL` pelo SQL Editor do
Neon; tirar do banco dados auxiliares antigos (pagamentos, PNCP) que a
busca não usa; passar para o plano pago do Neon (cobra por GB guardado).

## O que fica de fora

- A reindexação (`reindex.yml`) relê os PDFs do bucket do GCP e não roda
  neste deploy. Para reprocessar edições, rode `make reindex FROM=… TO=…` na
  máquina com a base local e refaça a carga do passo 2.
- Os workflows `deploy.yml`, `infra.yml` e `reindex.yml` só rodam à mão.
