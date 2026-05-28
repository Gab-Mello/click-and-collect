.PHONY: run build test fmt tidy help

run: ## Run the API
	go run ./cmd/api

build: ## Build the binary into ./bin/api
	go build -o bin/api ./cmd/api

test: ## Run tests with race detector
	go test ./... -race -count=1

fmt: ## Format the code
	go fmt ./...

tidy: ## Tidy go.mod / go.sum
	go mod tidy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
