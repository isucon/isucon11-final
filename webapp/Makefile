.PHONY: up
up: ## prepare dev environment
	@docker compose up -d --build
	@curl -X POST http://localhost:7000/initialize

.PHONY: down
down: ## destroy dev environment
	@docker compose down -v
