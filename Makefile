# Project Configuration
MODULE_NAME := covered-call-tracker
DESIGN_PKG  := $(MODULE_NAME)/design
MAIN_ENTRY  := ./cmd/server
APP_BINARY  := server

.PHONY: all gen vendor sync build clean docker-build docker-up help

# Default target: Run generation, vendor dependencies, and verify build
all: sync build

## gen: Runs GOA code generation by temporarily bypassing the vendor directory constraint
gen:
	@echo "==> Running GOA code generation..."
	@rm -rf gen/
	@GOFLAGS="-mod=mod" $$(go env GOPATH)/bin/goa gen $(DESIGN_PKG)
	@echo "==> GOA generation complete."

## vendor: Syncs go.mod dependencies and mirrors gen/ into the vendor directory
vendor:
	@echo "==> Tidying modules and updating vendor directory..."
	@GOFLAGS="-mod=mod" go mod tidy
	@go mod vendor
	@echo "==> Mirroring generated code into vendor directory..."
	@mkdir -p vendor/$(MODULE_NAME)/gen
	@cp -r gen/* vendor/$(MODULE_NAME)/gen/
	@echo "==> Vendoring complete."

## sync: Full pipeline - Runs code generation, tidies go.mod, and syncs vendor directory
sync: gen vendor

## build: Compiles the local server binary using vendored dependencies
build:
	@echo "==> Verifying local build with -mod=vendor..."
	@go build -mod=vendor -o $(APP_BINARY) $(MAIN_ENTRY)
	@echo "==> Build successful! Binary created: ./$(APP_BINARY)"

## docker-build: Rebuilds Docker images without using cache
docker-build: sync
	@echo "==> Building Docker container stack..."
	@docker compose build --no-cache

## docker-up: Rebuilds and launches the Docker stack in detached mode
docker-up: docker-build
	@echo "==> Starting Docker container stack..."
	@docker compose up -d

## clean: Removes generated code, vendored packages, and compiled binaries
clean:
	@echo "==> Cleaning generated artifacts..."
	@rm -rf gen/ vendor/ $(APP_BINARY)

## help: Displays available targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'