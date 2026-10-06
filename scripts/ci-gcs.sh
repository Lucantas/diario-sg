#!/usr/bin/env bash
set -euo pipefail

DATA="${1:?uso: ci-gcs.sh <diretório de dados>}"
PORT="${GCS_PORT:-4443}"
mkdir -p "$DATA/diario-gazettes"
docker run -d --name gcs --user "$(id -u):$(id -g)" -p "$PORT:4443" -v "$DATA:/data" fsouza/fake-gcs-server:1.56.1 \
  -scheme http -port 4443 -public-host "localhost:$PORT" -backend filesystem -filesystem-root /data >/dev/null
for _ in $(seq 1 30); do
  curl -fsS "http://localhost:$PORT/storage/v1/b" >/dev/null 2>&1 && exit 0
  sleep 1
done
echo "emulador GCS não subiu" >&2
docker logs gcs >&2 || true
exit 1
