SHELL := /usr/bin/env bash
GO ?= go
PNPM ?= pnpm
WEB_DIR := web
BIN_DIR := bin
BIN := $(BIN_DIR)/budget-agent
WEB_DEPS := $(WEB_DIR)/node_modules/.installed

.DEFAULT_GOAL := help

.PHONY: help dev dev-api dev-web web-install web-build web-check web-lint web-test go-test build test fmt fmt-check lint check clean image deploy

help: ## Show available targets
	@printf 'Targets:\n'
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

$(WEB_DEPS): $(WEB_DIR)/package.json $(WEB_DIR)/pnpm-lock.yaml
	cd $(WEB_DIR) && $(PNPM) install --frozen-lockfile
	@touch $(WEB_DEPS)

web-install: $(WEB_DEPS) ## Install frontend dependencies

dev: $(WEB_DEPS) ## Run API and web UI together
	./scripts/dev.sh

dev-api: ## Run the Go API on 127.0.0.1:8080
	@mkdir -p .local/data .local/receipts
	@APP_ADDR=127.0.0.1:8080 \
		DATABASE_PATH="$(CURDIR)/.local/data/budget.db" \
		RECEIPTS_DIR="$(CURDIR)/.local/receipts" \
		BASE_URL=http://127.0.0.1:8080 \
		AUTH_MODE=dev \
		TOKEN_PEPPER=local-development-only \
		LOG_LEVEL=debug \
		$(GO) run ./cmd/budget-agent

dev-web: $(WEB_DEPS) ## Run the SvelteKit dev server on 127.0.0.1:5173
	cd $(WEB_DIR) && $(PNPM) dev --host 127.0.0.1

web-build: $(WEB_DEPS) ## Build the static web UI
	cd $(WEB_DIR) && $(PNPM) build
	@rm -rf internal/web/static
	@mkdir -p internal/web/static
	@cp -r $(WEB_DIR)/build/. internal/web/static/
	@touch internal/web/static/.gitkeep

web-check: $(WEB_DEPS) ## Type-check the web UI
	cd $(WEB_DIR) && $(PNPM) check

web-lint: $(WEB_DEPS) ## Format-check and lint the web UI
	cd $(WEB_DIR) && $(PNPM) lint

web-test: $(WEB_DEPS) ## Run frontend unit tests
	cd $(WEB_DIR) && $(PNPM) test

go-test: ## Run Go tests
	$(GO) test ./...

build: web-build ## Build the server binary
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/budget-agent

test: go-test web-test ## Run all unit tests

fmt: $(WEB_DEPS) ## Format Go and web sources
	$(GO) fmt ./...
	cd $(WEB_DIR) && $(PNPM) format

fmt-check: ## Check Go formatting
	@unformatted="$$(find . -path ./web -prune -o -name '*.go' -print | xargs gofmt -l)"; \
	if [ -n "$$unformatted" ]; then \
		printf 'gofmt needed:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

lint: fmt-check $(WEB_DEPS) ## Run static analysis and linters
	$(GO) vet ./...
	cd $(WEB_DIR) && $(PNPM) lint

check: lint web-check test web-build ## Run the full local quality gate

clean: ## Remove generated build output
	rm -rf $(BIN_DIR) $(WEB_DIR)/build $(WEB_DIR)/.svelte-kit $(WEB_DIR)/coverage $(WEB_DIR)/test-results $(WEB_DIR)/playwright-report

image: web-build ## Build the container image with Podman
	podman build -f Containerfile -t budget-agent:latest .

deploy: image ## Build image and print install reminder
	@printf 'Image built. Install units with:\n'
	@printf '  ln -sf "$$PWD/deploy/quadlet/"*.pod "$$PWD/deploy/quadlet/"*.container ~/.config/containers/systemd/\n'
	@printf '  systemctl --user daemon-reload && systemctl --user start budget-agent-pod.service\n'
