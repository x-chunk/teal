BINARY := teal
PKG := ./...
GOFLAGS ?=
LDFLAGS := -s -w -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the package
	go build $(GOFLAGS) $(PKG)

.PHONY: test
test: ## Run the tests
	go test $(GOFLAGS) -race $(PKG)

.PHONY: cover
cover: ## Run the tests and open the coverage report
	go test $(GOFLAGS) -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out

.PHONY: fmt
fmt: ## Format the code
	go fmt $(PKG)

.PHONY: vet
vet: ## Run go vet
	go vet $(PKG)

.PHONY: tidy
tidy: ## Tidy go.mod
	go mod tidy

.PHONY: check
check: fmt vet test ## Format, vet and test

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin coverage.out
