DEST=$(PWD)/isucholar
COMPILER=go
GO_FILES=$(wildcard ./*.go ./**/*.go)

.PHONY: all
all: clean build ## Cleanup and Build

.PHONY: build
build: $(GO_FILES) ## Build executable files
	GOOS=linux GOARCH=amd64 $(COMPILER) build -o $(DEST) -ldflags "-s -w"

.PHONY: clean
clean: ## Cleanup files
	@$(RM) -r $(DEST)

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deploy:
	rsync -avh --progress ./isucholar isucon11f-1:/home/isucon/webapp/go/
	ssh isucon11f-1 sudo systemctl restart isucholar.go.service
