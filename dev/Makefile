TBLS_VERSION := 1.50.0

.PHONY: up
up: ## prepare dev environment
	@docker compose up -d --build
	@until (curl -X POST --silent http://localhost:7000/initialize) do sleep 1;done; echo initialized

.PHONY: down
down: ## destroy dev environment
	@docker compose down -v

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
