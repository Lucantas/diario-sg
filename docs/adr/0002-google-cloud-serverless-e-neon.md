# ADR 0002 — Google Cloud serverless + Neon Postgres

**Status:** aceito

## Contexto
Precisamos de filas, jobs agendados, containers e banco relacional com custo
próximo de zero no início, sem usar AWS, e com um caminho claro para crescer.

## Decisão
- **Cloud Run** para API, worker e site (escala a zero) e **Cloud Run Jobs**
  para o scraper, disparado pelo **Cloud Scheduler**.
- **Pub/Sub** com assinaturas push autenticadas (OIDC), retry com backoff e
  dead-letter queue.
- **Cloud Storage** para os PDFs originais (reprocessáveis a qualquer momento).
- **Neon** como Postgres serverless (plano gratuito), provisionado pelo
  Terraform. Busca textual nativa do Postgres com dicionário português.
- **Workload Identity Federation**: o GitHub Actions não usa chave JSON.

Alternativas consideradas: Oracle Cloud Always Free (VMs generosas, mas
exige operar servidores); Azure Container Apps + Service Bus (bom, porém o
Service Bus não tem plano gratuito equivalente); Cloud SQL desde o início
(sem plano gratuito).

## Consequências
- Custo praticamente zero no volume de um município.
- O Neon roda em infraestrutura de terceiros (AWS/Azure por baixo), mas é
  consumido apenas como Postgres gerenciado via `DATABASE_URL`.
- **Migração para Cloud SQL/AlloyDB**: criar a instância via Terraform,
  `pg_dump`/`pg_restore`, trocar o secret `database-url` e, se usar IP
  privado, adicionar Direct VPC egress aos serviços. Nenhum código muda.
