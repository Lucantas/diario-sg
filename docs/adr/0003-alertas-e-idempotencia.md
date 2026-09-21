# ADR 0003 — Alertas por evento e idempotência

**Status:** aceito

## Contexto
O Pub/Sub entrega mensagens "pelo menos uma vez". Uma mesma edição pode ser
coletada, indexada ou gerar alertas mais de uma vez.

## Decisão
- Scraper: caminho determinístico no bucket + marcador `.published` gravado
  só depois do evento publicado.
- Indexação: `checksum` único; reprocessar a mesma edição apenas republica o
  evento `gazette.indexed.v1`.
- Alertas: tabela `notifications_sent` com chave (inscrição, edição).
- O consumidor responde 2xx para sucesso e para erros permanentes (payload
  inválido) e 5xx para falhas temporárias, que vão para retry e, após 5
  tentativas, para a DLQ.
- Casamento de alertas: para cada edição indexada, uma busca por inscrição
  ativa, restrita àquela edição.

## Consequências
- Reentregas e reexecuções são seguras.
- O casamento é O(inscrições) por edição. Com milhares de inscrições,
  migrar para busca reversa (percolator do OpenSearch, ou agrupar inscrições
  com termos iguais) sem mudar o contrato do caso de uso.
