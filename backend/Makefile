GOLANGCI_LINT_VERSION := v2.14.0
IMAGE := family-expenses-backend

# Load ../.env for local runs, if present.
-include ../.env
export

.PHONY: build test test-docker lint api-lint fmt run-api run-bot docker-build

build: ## Build all binaries into ./bin
	go build -o bin/ ./cmd/...

test: ## Run unit tests
	go test -race -count=1 ./...

test-docker: ## Run unit tests with the race detector in a Linux container (no local Go needed)
	docker run --rm -v "$(CURDIR):/src" -v gomodcache:/go/pkg/mod -w /src golang:1.27 go test -race -count=1 ./...

lint: ## Run golangci-lint (via Docker, no local install needed)
	docker run --rm -v "$(CURDIR):/src" -w /src golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

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
