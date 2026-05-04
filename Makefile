SHELL := /bin/bash

GO ?= go
NPM ?= npm
WEB_DIR := web
BIN_DIR := bin
BINARY := $(BIN_DIR)/suitest

.PHONY: help setup dev dev-backend dev-frontend build build-backend build-frontend test test-go test-frontend fmt lint clean web

help:
	@echo "Available targets:"
	@echo "  make setup          Install dependencies for Go and web"
	@echo "  make dev            Run backend and frontend dev servers"
	@echo "  make dev-backend    Run Go web server in dev mode"
	@echo "  make dev-frontend   Run SvelteKit dev server"
	@echo "  make build          Build frontend and backend"
	@echo "  make build-backend  Build Go binary"
	@echo "  make build-frontend Build SvelteKit frontend"
	@echo "  make test           Run Go and frontend checks"
	@echo "  make fmt            Format Go and frontend sources"
	@echo "  make lint           Run frontend lint and Go vet"
	@echo "  make clean          Remove build artifacts"
	@echo "  make web            Start local web mode"

setup:
	$(GO) mod download
	cd $(WEB_DIR) && $(NPM) install

dev:
	make -j2 dev-backend dev-frontend

dev-backend:
	$(GO) run . web --dev

dev-frontend:
	cd $(WEB_DIR) && $(NPM) run dev

build: build-frontend build-backend

build-backend:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) .

build-frontend:
	cd $(WEB_DIR) && $(NPM) run build

test: test-go test-frontend

test-go:
	$(GO) test ./...

test-frontend:
	cd $(WEB_DIR) && $(NPM) run check

fmt:
	$(GO) fmt ./...
	cd $(WEB_DIR) && $(NPM) run format

lint:
	$(GO) vet ./...
	cd $(WEB_DIR) && $(NPM) run lint

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(WEB_DIR)/build $(WEB_DIR)/.svelte-kit

web:
	$(GO) run . web
