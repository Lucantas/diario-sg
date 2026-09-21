# testdata

- `editions/` — edições reais (PDF + texto extraído). **Não versionadas**: contêm
  nomes de pessoas físicas. Reproduza com `./scripts/fetch-editions.sh`.
  O teste `TestMeasureRealEditions` (`go test -run Measure -v ./internal/adapters/parser/`)
  usa estes arquivos quando existem e é pulado quando não existem.
- `fixtures/` — trechos reais **anonimizados** (nomes de pessoas físicas trocados
  por nomes fictícios, mantendo o formato). Versionados e usados nos testes unitários.
