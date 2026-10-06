#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

read -rsp "Connection string DIRETA do Neon (sem -pooler): " NEON_DATABASE_URL
echo
export NEON_DATABASE_URL

docker compose exec -T postgres test -f /tmp/diario.dump || \
  docker compose exec -T postgres pg_dump -U postgres -d diario -Fc --no-owner --no-privileges -f /tmp/diario.dump

echo "Restaurando no Neon (alguns minutos)..."
docker compose exec -T -e NEON_DATABASE_URL postgres sh -c \
  'pg_restore --no-owner --no-privileges -d "$NEON_DATABASE_URL" /tmp/diario.dump' || \
  echo "pg_restore terminou com avisos; confira as contagens abaixo"

docker compose exec -T -e NEON_DATABASE_URL postgres sh -c \
  'psql "$NEON_DATABASE_URL" -Atc "select pg_size_pretty(pg_database_size(current_database())), (select count(*) from acts), (select max(version) from schema_migrations)"'

read -rp "Cadastrar os secrets no GitHub agora? [s/N] " answer
if [ "$answer" = "s" ]; then
  gh secret set DATABASE_URL_DIRECT -R Lucantas/diario-sg -e producao --body "$NEON_DATABASE_URL"
  read -rsp "Connection string POOLED do Neon (com -pooler): " pooled
  echo
  gh secret set DATABASE_URL -R Lucantas/diario-sg -e producao --body "$pooled"
fi
