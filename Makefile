# Atalhos de desenvolvimento. Rode `make help`.
SHELL := /bin/bash
-include .env
export

GO_MODULES := pkg services/api services/scraper

.PHONY: help up down setup migrate run-api run-worker run-scraper run-web test test-integration lint fmt tf-fmt

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

run-api: ## API pública em :8080
	cd services/api && PORT=8080 go run ./cmd/api

run-worker: ## Worker (recebe push do Pub/Sub) em :8081
	cd services/api && PORT=8081 go run ./cmd/worker

run-scraper: ## Executa uma coleta
	cd services/scraper && go run ./cmd/scraper

run-web: ## Front em :5173 (proxy /api -> :8080)
	cd apps/web && npm install && npm run dev

test: ## Testes unitários de todos os módulos Go
	@for m in $(GO_MODULES); do (cd $$m && go test -race ./...) || exit 1; done

test-integration: ## Testes de integração (precisa de `make up migrate`)
	cd services/api && TEST_DATABASE_URL="$(DATABASE_URL)" go test -tags integration -count=1 ./internal/integration/

lint: ## go vet + gofmt
	@for m in $(GO_MODULES); do (cd $$m && go vet ./... && test -z "$$(gofmt -l .)") || exit 1; done

tf-fmt: ## Formata o Terraform
	terraform fmt -recursive infra
