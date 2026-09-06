# Kabanos developer shortcuts. `make help` lists targets.
.DEFAULT_GOAL := help
COMPOSE := docker compose

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# --- local stack (docker) ---
.PHONY: up
up: ## Start the full stack (build + run) in the background
	$(COMPOSE) up --build -d

.PHONY: down
down: ## Stop the stack
	$(COMPOSE) down

.PHONY: clean
clean: ## Stop the stack and remove volumes (DESTROYS data)
	$(COMPOSE) down -v

.PHONY: logs
logs: ## Tail logs from all services
	$(COMPOSE) logs -f

.PHONY: scale
scale: ## Start the replicated/HA topology (postgres primary+replica, 3x api)
	$(COMPOSE) -f docker-compose.scale.yml up --build -d

.PHONY: scale-down
scale-down: ## Stop the replicated topology
	$(COMPOSE) -f docker-compose.scale.yml down

# --- backend (requires local Go 1.23+; otherwise use docker targets) ---
.PHONY: backend-test
backend-test: ## Run backend unit tests
	cd backend && go test ./...

.PHONY: backend-build
backend-build: ## Compile backend binaries
	cd backend && go build ./...

.PHONY: backend-tidy
backend-tidy: ## Sync go.mod/go.sum
	cd backend && go mod tidy

# --- frontend ---
.PHONY: frontend-dev
frontend-dev: ## Run the Next.js dev server (expects API on :8080)
	cd frontend && npm run dev

.PHONY: frontend-build
frontend-build: ## Production build of the frontend
	cd frontend && npm run build
