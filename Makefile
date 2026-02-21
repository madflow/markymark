.DEFAULT_GOAL := help
.PHONY: build generate clean run watch check help

build: ## Build the application
	templ generate
	go build -o markymark .

generate: ## Generate templ files only
	templ generate

clean: ## Clean build artifacts
	rm -f markymark
	rm -f *_templ.go

run: build ## Run the application (builds first)
	./markymark

watch: ## Watch templ files and regenerate automatically
	templ generate -w

check: ## Run linter
	golangci-lint run

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
