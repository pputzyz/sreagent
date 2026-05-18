.PHONY: help build run dev test clean docker-up docker-down lint fmt web-install web-dev web-build check-modules

# Variables
APP_NAME := sreagent
BINARY := bin/$(APP_NAME)
GO := go
GOFLAGS := -v
LDFLAGS := -w -s

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ====== Backend ======

build: ## Build the Go binary
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/server

run: build ## Build and run the server
	./$(BINARY) --config configs/config.yaml

dev: ## Run with hot reload (requires air: go install github.com/air-verse/air@latest)
	air -c .air.toml

test: ## Run tests
	$(GO) test ./... -v -cover

lint: ## Run linter (requires golangci-lint)
	golangci-lint run ./...

fmt: ## Format Go code
	$(GO) fmt ./...
	goimports -w .

tidy: ## Tidy Go modules
	$(GO) mod tidy

check-modules: ## Verify MODULES.md counts match actual codebase
	$(GO) run scripts/check-modules.go

# ====== Frontend ======

web-install: ## Install frontend dependencies
	cd web && npm install

web-dev: ## Start frontend dev server
	cd web && npm run dev

web-build: ## Build frontend for production
	cd web && npm run build

# ====== Docker ======

docker-up: ## Start dev dependencies (MySQL + Redis) via docker run
	docker run -d --name sreagent-mysql \
		-e MYSQL_ROOT_PASSWORD=root \
		-e MYSQL_DATABASE=sreagent \
		-e MYSQL_USER=sreagent \
		-e MYSQL_PASSWORD=sreagent \
		-p 3306:3306 mysql:8.0
	docker run -d --name sreagent-redis \
		-p 6379:6379 redis:7-alpine

docker-down: ## Stop dev dependencies
	docker rm -f sreagent-mysql sreagent-redis 2>/dev/null || true

docker-build: ## Build Docker image
	docker build -t $(APP_NAME):latest -f deploy/docker/Dockerfile .

# ====== Database ======

db-migrate: build ## Run database migrations (via auto-migrate)
	./$(BINARY) --config configs/config.yaml

# ====== All ======

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/

all: tidy fmt build web-build ## Build everything
