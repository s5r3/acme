# Run `make help` for the list of targets.

GO        ?= go
CONFIG    ?= configs/config.yaml
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X example.com/acme/internal/app.Version=$(VERSION)
TEST_DSN  ?= postgres://acme:acme@localhost:5432/acme_test?sslmode=disable
COMPOSE   := docker compose -f deploy/docker-compose.yml

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## --- Develop ---------------------------------------------------------------

.PHONY: generate
generate: ## Generate Go code from .templ files
	$(GO) tool templ generate

.PHONY: build
build: generate ## Build api, web and migrate into ./bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/ ./cmd/...

.PHONY: run-api
run-api: generate ## Run the JSON API (configs/config.yaml + config.local.yaml)
	$(GO) run ./cmd/api -config $(CONFIG)

.PHONY: run-web
run-web: generate ## Run the HTML front-end
	$(GO) run ./cmd/web -config $(CONFIG)

.PHONY: db-up
db-up: ## Start only PostgreSQL in Docker (creates acme and acme_test)
	$(COMPOSE) up -d db

## --- Database --------------------------------------------------------------

.PHONY: migrate-up
migrate-up: ## Apply pending migrations
	$(GO) run ./cmd/migrate -config $(CONFIG) up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	$(GO) run ./cmd/migrate -config $(CONFIG) down

.PHONY: migrate-status
migrate-status: ## Show migration status
	$(GO) run ./cmd/migrate -config $(CONFIG) status

.PHONY: migrate-new
migrate-new: ## Create a migration: make migrate-new name=add_orders_index
	@test -n "$(name)" || (echo "usage: make migrate-new name=snake_case_name" && exit 1)
	$(GO) run ./cmd/migrate create $(name)

.PHONY: seed
seed: ## Create demo accounts (development only)
	$(GO) run ./cmd/migrate -config $(CONFIG) seed

## --- Quality ---------------------------------------------------------------

.PHONY: test
test: generate ## Unit tests with the race detector
	$(GO) test -race ./...

.PHONY: test-integration
test-integration: generate ## Unit + integration tests against PostgreSQL (TEST_DSN)
	ACME_TEST_DATABASE_DSN="$(TEST_DSN)" $(GO) test -race -count=1 -p 1 -tags=integration ./...

.PHONY: lint
lint: ## golangci-lint (includes the import-direction rules)
	golangci-lint run ./...

.PHONY: openapi
openapi: ## Lint and bundle the OpenAPI spec into bin/openapi.yaml
	npx --yes @redocly/cli@latest lint api/openapi/openapi.yaml
	npx --yes @redocly/cli@latest bundle api/openapi/openapi.yaml -o bin/openapi.yaml

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

## --- Scaffolding -----------------------------------------------------------

.PHONY: feature
feature: ## Scaffold a feature from internal/feature/_template: make feature name=invoice
	@test -n "$(name)" || (echo "usage: make feature name=invoice" && exit 1)
	./scripts/new-feature.sh $(name)

## --- Docker ----------------------------------------------------------------

.PHONY: up
up: ## Build and start db, migrations, api and web in Docker
	$(COMPOSE) up --build -d
	@echo "API: http://localhost:8080/healthz   Web: http://localhost:8081"

.PHONY: down
down: ## Stop the Docker stack
	$(COMPOSE) down

.PHONY: clean
clean: ## Remove build output
	rm -rf bin
