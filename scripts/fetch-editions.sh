#!/usr/bin/env bash
# Baixa edições reais do Diário Oficial de São Gonçalo e extrai o texto com
# pdftotext (modo de leitura, sem -layout) em services/api/testdata/editions/.
# Os arquivos contêm nomes de pessoas físicas e por isso NÃO são versionados.
#
#   ./scripts/fetch-editions.sh                # datas padrão usadas na fase 1
#   ./scripts/fetch-editions.sh 2026-09-18     # datas específicas
set -euo pipefail

BASE_URL="${SOURCE_URL:-https://do.pmsg.rj.gov.br/}"
UA="diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
OUT="$(cd "$(dirname "$0")/.." && pwd)/services/api/testdata/editions"
DEFAULT_DATES=(2024-03-15 2026-03-31 2026-06-30 2026-08-31 2026-09-16 2026-09-17 2026-09-18)

mkdir -p "$OUT"
dates=("${@:-${DEFAULT_DATES[@]}}")
for d in "${dates[@]}"; do
  name="${d//-/_}"
  url="${BASE_URL%/}/diario/${name}.pdf"
  if [[ -f "$OUT/$name.pdf" ]]; then
    echo "$name.pdf já existe, pulando download"
  else
    echo "baixando $url"
    curl -fsS -A "$UA" --max-time 120 -o "$OUT/$name.pdf" "$url"
    sleep 2
  fi
  pdftotext -enc UTF-8 "$OUT/$name.pdf" "$OUT/$name.txt" 2>/dev/null
  printf "  %s: %s páginas, %s linhas\n" "$name" "$(pdfinfo "$OUT/$name.pdf" | awk '/^Pages/{print $2}')" "$(wc -l < "$OUT/$name.txt")"
done
