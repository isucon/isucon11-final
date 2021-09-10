.PHONY: up
up: ## prepare dev environment
	@docker-compose up -d --build

.PHONY: down
down: ## destroy dev environment
	@docker-compose down -v

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
