.PHONY: validate test lint fmt vet tidy help ready hooks

## Primary target - run before completing any task
validate: fmt vet tidy lint test ## Run all validation checks
	@echo "✓ All validation checks passed"

## Testing
test: ## Run tests with race detector
	go test -race ./...

## Linting
lint: ## Run golangci-lint
	golangci-lint run ./...

## Formatting
fmt: ## Run gofmt
	@go fmt ./...

## Vet
vet: ## Run go vet
	go vet ./...

## Tidy
tidy: ## Ensure go.mod is tidy
	@go mod tidy

## Beads workflow
ready: ## Show tasks with no blockers
	@bd ready

list: ## List all beads tasks
	@bd list

## Git hooks
hooks: ## Install git hooks
	@ln -sf ../../scripts/post-merge .git/hooks/post-merge
	@echo "✓ Git hooks installed"

## Help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
