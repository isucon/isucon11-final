COMMIT=$(shell cat /etc/REVISION 2>/dev/null || git rev-parse --short HEAD)
DIRTY=$(shell git diff --quiet || echo '+dirty')

GOPRIVATE="github.com/isucon"
GOLDFLAGS=-X main.COMMIT=$(COMMIT)$(DIRTY)

GOFILES=$(wildcard *.go **/*.go)

BINARY=./bin/benchmarker

.PHONY: all
all: build ## Execute all tasks

.PHONY: build
build: $(EXE) ## Build benchmarker

.PHONY: clean
clean: ## Cleanup working directory
	@$(RM) $(BINARY)
	@go clean

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'


$(EXE): Makefile go.mod $(GOFILES)
	GOPRIVATE=$(GOPRIVATES) go build -ldflags "$(GOLDFLAGS)" -o $(BINARY) -v github.com/isucon/isucon11-final/benchmarker
