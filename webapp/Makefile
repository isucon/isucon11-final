TBLS_VERSION := 1.50.0

.PHONY: up
up: ## prepare dev environment
	@docker compose up -d --build
	@curl -X POST http://localhost:7000/initialize

.PHONY: down
down: ## destroy dev environment
	@docker compose down -v

.PHONY: db-gen-docs
db-gen-docs: ## Generate db docs in docs/schema
	@if [ -d "./docs/schema" ]; then \
		rm -r ./docs/schema; \
	fi
	docker run --rm --net=host -v $$PWD:/work k1low/tbls:$(TBLS_VERSION) doc

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
