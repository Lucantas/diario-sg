# ADR 0005 — Registro de coletas e arquivo bruto

**Status:** aceito

## Contexto
Cada fonte nova vai ter um coletor: Câmara, TCE-RJ, PNCP e as outras do
plano. Para declarar a cobertura ("despesas com CNPJ só de 2017 a 2022"),
avisar quando uma coleta volta vazia e reprocessar quando o normalizador
mudar, é preciso saber o que cada coleta pediu, o que trouxe e guardar o
que a fonte devolveu.

## Decisão
- **`fetch_runs`** registra cada coleta:
  - fonte, período pedido, início e fim;
  - encontrados, gravados, pulados e com falha;
  - a mensagem de erro.
- **Por evento.** No fim de cada coleta, o coletor publica
  `fetch.completed.v1` (contrato em `contracts/events/`) num tópico
  próprio, e o worker grava a linha. O coletor continua sem acesso ao
  banco, como o scraper hoje. A entrega é "pelo menos uma vez", então o
  id da coleta vem no evento e a gravação é idempotente.
- **Arquivo bruto**: tudo o que a fonte devolve (PDF, JSON, CSV, HTML) vai
  para o bucket antes de ser interpretado:
  - caminho determinístico `raw/<fonte>/<AAAA>/<MM>/<DD>/<nome>`;
  - SHA-256 gravado na tabela da fonte.
  - O Diário da Prefeitura já faz isso com os PDFs (`storage_path` e
    `checksum` em `gazettes`) e mantém o caminho atual. As fontes novas
    seguem o formato acima a partir da etapa C.
- A ferramenta `fontes` do MCP mostra a última coleta de cada fonte:
  quando foi, o que trouxe e se falhou.

## Consequências
- Um tópico e uma assinatura push a mais por ambiente, que o Terraform e
  o `local-setup.sh` criam.
- Coleta que falha antes de publicar o evento (queda do job) não aparece
  em `fetch_runs`. A ausência de coleta recente também é sinal, e o
  `fontes` mostra a data da última.
