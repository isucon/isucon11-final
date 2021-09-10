TARGET ?= golang

.PHONY: up
up: ## prepare dev environment
	@docker-compose -f docker-compose-$(TARGET).yaml up -d --build
	@until (curl --silent http://localhost:7000/api/users/me > /dev/null) do sleep 1;done; echo ready

.PHONY: down
down: ## destroy dev environment
	@docker-compose -f docker-compose-$(TARGET).yaml down -v

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
