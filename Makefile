BINARY   := ctsnare
CMD_PATH := ./cmd/ctsnare
GO       := go
LINT     := golangci-lint

GIT_SHA  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build test lint fmt vet clean coverage check run help

build: ## Build the binary
	$(GO) build -o $(BINARY) $(CMD_PATH)

test: ## Run tests with race detection
	$(GO) test -race -count=1 ./...

lint: ## Run golangci-lint
	$(LINT) run ./...

fmt: ## Format code with gofmt
	gofmt -w .

vet: ## Run go vet
	$(GO) vet ./...

clean: ## Remove build artifacts
	rm -f $(BINARY) *.db coverage.out

coverage: ## Generate test coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

check: build vet lint test ## Full CI suite: build + vet + lint + test

run: ## Run the application
	$(GO) run $(CMD_PATH)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
