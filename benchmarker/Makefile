COMMIT=$(shell cat /etc/REVISION 2>/dev/null || git rev-parse --short HEAD)
DIRTY=$(shell git diff --quiet || echo '+dirty')

GOPRIVATE="github.com/isucon"
GOLDFLAGS=-X main.COMMIT=$(COMMIT)$(DIRTY)

GOFILES=$(wildcard *.go **/*.go)
PUBLIC_FILES=$(wildcard ../webapp/frontend/public/*.* ../webapp/frontend/public/**/*.*)
PUBLIC_FILES_CHECKSUM=$(patsubst ../webapp/frontend/public/%,checksum/%,$(PUBLIC_FILES))

EXE=./bin/benchmarker

.PHONY: all
all: build ## Execute all tasks

.PHONY: build
build: $(EXE) ## Build benchmarker
	GOPRIVATE=$(GOPRIVATE) go build -ldflags "$(GOLDFLAGS)" -o $(EXE) -v github.com/isucon/isucon11-final/benchmarker

.PHONY: clean
clean: ## Cleanup working directory
	@$(RM) $(EXE)
	@go clean

.PHONY: vendor
vendor:
	GOPRIVATE=$(GOPRIVATE) go mod vendor

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'


$(EXE): Makefile go.mod $(GOFILES) $(GOPROTOFILES)
	GOPRIVATE=$(GOPRIVATES) go build $(GOARGS) -ldflags "$(GOLDFLAGS)" -o $(EXE) -v github.com/isucon/isucon11-final/benchmarker
