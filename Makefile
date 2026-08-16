APP_NAME := payment-sandbox
CMD_PATH := ./cmd/payment-sandbox
COVERAGE_FILE := coverage.out

.PHONY: help build run test test-race coverage fmt lint tidy check clean

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*## / { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/$(APP_NAME) $(CMD_PATH)

run: ## Run the application
	go run $(CMD_PATH)

test: ## Run tests
	go test ./...

test-race: ## Run tests with race detector
	go test -race ./...

coverage: ## Generate test coverage
	go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

fmt: ## Format Go source files
	gofmt -w .
	golangci-lint fmt

lint: ## Run static analysis
	golangci-lint run

tidy: ## Normalize Go module dependencies
	go mod tidy

check: ## Run local CI checks
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	go build ./...
	go test ./...
	golangci-lint run

clean: ## Remove generated local artifacts
	rm -rf bin
	rm -f $(COVERAGE_FILE)