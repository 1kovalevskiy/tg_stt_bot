TOOLS_DIR ?= $(CURDIR)/bin
CACHE_DIR ?= $(CURDIR)/.cache

# Caches are pinned inside the repository so lint, test and build never depend
# on the developer's or the runner's home directory. `?=` would be a no-op for
# anything already exported into the environment (HOME always is), hence `:=`.
export HOME := $(abspath $(CACHE_DIR))/home
export GOPATH := $(abspath $(CACHE_DIR))/gopath
export GOMODCACHE := $(abspath $(CACHE_DIR))/gomod
export GOCACHE := $(abspath $(CACHE_DIR))/go-build
export GOLANGCI_LINT_CACHE := $(abspath $(CACHE_DIR))/golangci-lint
export PATH := $(abspath $(TOOLS_DIR)):$(PATH)

GOLANGCI_LINT_VERSION ?= v2.8.0

.PHONY: .install-golangci-lint
.install-golangci-lint:
	@mkdir -p $(TOOLS_DIR)
	@if [ -x "$(TOOLS_DIR)/golangci-lint" ]; then \
		echo "Using existing golangci-lint from $(TOOLS_DIR)"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) into $(TOOLS_DIR)"; \
		GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

.PHONY: lint
lint: .install-golangci-lint
	@$(TOOLS_DIR)/golangci-lint run ./...

.PHONY: format
format:
	@go fmt ./...

# The bot runs a background delivery goroutine and several update workers, so
# the suite runs under the race detector: a data race would otherwise ship green.
.PHONY: test
test:
	@go test -race -count=1 ./...

.PHONY: build
build:
	@go build -o $(TOOLS_DIR)/tg_stt_bot ./cmd/app

.PHONY: run
run:
	@go run ./cmd/app

.PHONY: tidy
tidy:
	@go mod tidy
