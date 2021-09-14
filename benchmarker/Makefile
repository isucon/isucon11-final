COMMIT=$(shell cat /etc/REVISION 2>/dev/null || git rev-parse --short HEAD)
DIRTY=$(shell git diff --quiet || echo '+dirty')

GOPRIVATE="github.com/isucon"
GOLDFLAGS=-X main.COMMIT=$(COMMIT)$(DIRTY)

GOFILES=$(wildcard *.go **/*.go)

PUBLIC_FILES_DIR=$(abspath ../webapp/frontend/dist)
PUBLIC_FILES=$(abspath $(wildcard ../webapp/frontend/dist/*.* ../webapp/frontend/dist/**/*.* ../webapp/frontend/dist/**/**/*.*))

EXE=./bin/benchmarker

.PHONY: all
all: build ## Execute all tasks

.PHONY: build
build: $(EXE) ## Build benchmarker

.PHONY: clean
clean: ## Cleanup working directory
	@$(RM) $(EXE)
	@go clean

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: assets
assets: $(PUBLIC_FILES) ## regenerate checksum of assets
	@PUBLIC_FILES_DIR="$(PUBLIC_FILES_DIR)" PUBLIC_FILES="$(PUBLIC_FILES)" go generate ./assets/gen.go

.PHONY: initial_course
initial_course: ## generate initial course data
	@go generate ./tools/gen_course_data.go

$(EXE): Makefile go.mod $(GOFILES) assets
	@GOPRIVATE=$(GOPRIVATES) go build -ldflags "$(GOLDFLAGS)" -o $(EXE) -v github.com/isucon/isucon11-final/benchmarker

$(EXE)_linux_amd64: Makefile go.mod $(GOFILES) assets
	@GOOS=linux GOARCH=amd64 GOPRIVATE=$(GOPRIVATES) go build -ldflags "$(GOLDFLAGS)" -o $(EXE)_linux_amd64 -v github.com/isucon/isucon11-final/benchmarker
