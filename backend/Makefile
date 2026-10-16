GOLANGCI_LINT_VERSION := v2.14.0
IMAGE := family-expenses-backend

# Load ../.env for local runs, if present.
-include ../.env
export

.PHONY: build test test-docker test-integration test-live lint api-lint sqlc fmt run-api run-bot docker-build

build: ## Build all binaries into ./bin
	go build -o bin/ ./cmd/...

test: ## Run unit tests
	go test -race -count=1 ./...

test-docker: ## Run unit tests with the race detector in a Linux container (no local Go needed)
	docker run --rm -v "$(CURDIR):/src" -v gomodcache:/go/pkg/mod -w /src golang:1.27 go test -race -count=1 ./...

test-integration: ## Run unit + PostgreSQL integration tests (testcontainers, needs Docker)
	docker run --rm -v "$(CURDIR):/src" -v gomodcache:/go/pkg/mod \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal \
		-w /src golang:1.27 go test -tags integration -race -count=1 ./...

test-live: ## Smoke-test the real LLM from ../.env: one tool call + the task examples (~1 cent)
	docker run --rm --env-file ../.env -v "$(CURDIR):/src" -v gomodcache:/go/pkg/mod \
		-w /src golang:1.27 go test -tags live -count=1 -v -run Live ./internal/llm/ ./internal/agent/

lint: ## Run golangci-lint (via Docker, no local install needed)
	docker run --rm -v "$(CURDIR):/src" -w /src golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run --build-tags integration,live ./...

sqlc: ## Regenerate type-safe query code from internal/postgres/queries
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:1.31.1 generate

api-lint: ## Validate the OpenAPI contract (via Docker)
	docker run --rm -v "$(CURDIR)/api:/spec" -w /spec redocly/cli:2.64.0 lint openapi.yaml

fmt: ## Format code
	gofmt -w .

run-api: ## Run the API server with ../.env
	go run ./cmd/api

run-bot: ## Run the Telegram bot with ../.env
	go run ./cmd/bot

docker-build: ## Build the backend image
	docker build -t $(IMAGE) .
