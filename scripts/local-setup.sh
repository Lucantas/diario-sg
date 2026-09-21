#!/usr/bin/env bash
# Cria bucket, tópicos e assinaturas push nos emuladores locais,
# espelhando o que o Terraform cria na nuvem.
set -euo pipefail

PROJECT=local
PUBSUB=http://localhost:8085
GCS=http://localhost:4443
WORKER=http://host.docker.internal:8081

echo "Bucket..."
curl -fsS -X POST "$GCS/storage/v1/b?project=$PROJECT" \
  -H 'Content-Type: application/json' -d '{"name":"diario-gazettes"}' >/dev/null || true

for topic in gazette-fetched gazette-indexed; do
  echo "Tópico $topic..."
  curl -fsS -X PUT "$PUBSUB/v1/projects/$PROJECT/topics/$topic" >/dev/null || true
  curl -fsS -X PUT "$PUBSUB/v1/projects/$PROJECT/subscriptions/$topic-push" \
    -H 'Content-Type: application/json' \
    -d "{\"topic\":\"projects/$PROJECT/topics/$topic\",\"ackDeadlineSeconds\":300,
         \"pushConfig\":{\"pushEndpoint\":\"$WORKER/events/$topic\"}}" >/dev/null || true
done
echo "Pronto."
