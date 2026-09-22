# Atalhos de desenvolvimento. Rode `make help`.
SHELL := /bin/bash
-include .env
export

GO_MODULES := pkg services/api services/scraper

.PHONY: help up down setup migrate reindex ingest run-api run-worker run-scraper run-web test test-integration lint fmt tf-fmt

help: ## Lista os comandos
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n", $$1, $$2}'

up: ## Sobe Postgres e emuladores (Pub/Sub, GCS)
	docker compose up -d --wait postgres && docker compose up -d pubsub gcs

down: ## Derruba o ambiente local
	docker compose down

setup: ## Cria bucket, tópicos e assinaturas nos emuladores
	./scripts/local-setup.sh

migrate: ## Aplica migrations no banco local
	cd services/api && go run ./cmd/migrate

reindex: ## Reprocessa edições já indexadas com o parser atual: make reindex FROM=AAAA-MM-DD TO=AAAA-MM-DD
	@test -n "$(FROM)" -a -n "$(TO)" || (echo "uso: make reindex FROM=AAAA-MM-DD TO=AAAA-MM-DD"; exit 2)
	cd services/api && go run ./cmd/reindex -from "$(FROM)" -to "$(TO)"

ingest: ## Ingere um PDF baixado manualmente: make ingest FILE=edicao.pdf DATE=AAAA-MM-DD [EDITION=n]
	@test -n "$(FILE)" -a -n "$(DATE)" || (echo "uso: make ingest FILE=caminho.pdf DATE=AAAA-MM-DD [EDITION=n]"; exit 2)
	cd services/scraper && go run ./cmd/ingest -file "$(abspath $(FILE))" -date "$(DATE)" -edition "$(EDITION)"

run-api: ## API pública em :8080
	cd services/api && PORT=8080 go run ./cmd/api

run-worker: ## Worker (recebe push do Pub/Sub) em :8081
	cd services/api && PORT=8081 go run ./cmd/worker

run-scraper: ## Executa uma coleta (LOOKBACK_DAYS=n ou, para backfill, FROM=AAAA-MM-DD [TO=AAAA-MM-DD])
	cd services/scraper && go run ./cmd/scraper $(if $(FROM),-from $(FROM)) $(if $(TO),-to $(TO))

run-web: ## Front em :5173 (proxy /api -> :8080)
	cd apps/web && npm install && npm run dev

test: ## Testes unitários de todos os módulos Go
	@for m in $(GO_MODULES); do (cd $$m && go test -race ./...) || exit 1; done

# Banco separado (criado pelo próprio teste) para não misturar com edições ingeridas.
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/diario_test?sslmode=disable

test-integration: ## Testes de integração (precisa de `make up`)
	cd services/api && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -tags integration -count=1 ./internal/integration/

lint: ## go vet + gofmt
	@for m in $(GO_MODULES); do (cd $$m && go vet ./... && test -z "$$(gofmt -l .)") || exit 1; done

tf-fmt: ## Formata o Terraform
	terraform fmt -recursive infra
