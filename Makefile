GO=go
GO_TARGETS=./cmd/... ./internal/...
GOLINT_VERSION=1.62.2

.PHONY: help
help:
	@grep -E '^[a-zA-Z_\-\/]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build server
	@${GO} build -o aws-vsock-proxy ./cmd/server

.PHONY: build/linux/amd64
build/linux/amd64: ## build server for linux amd64
	@GOOS=linux GOARCH=amd64 ${GO} build -o aws-vsock-proxy ./cmd/server

.PHONY: deps
deps: ## set up all dependencies to run these make commands
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint@v${GOLINT_VERSION}

.PHONY: lint
lint: ## lint the code
	@golangci-lint run --timeout 5m ${GO_TARGETS}

.PHONY: tidy
tidy: ## tidy dependencies
	@${GO} mod tidy

.PHONY: tidy/check
tidy/check: ## check whether module is tidied
	@${GO} mod tidy
	@git diff --exit-code || (echo "module is not tidy, please run 'go mod tidy'" && false)

.PHONY: test
test: ## run tests
	@${GO} test ${GO_TARGETS}

.PHONY: fmt
fmt: ## format the code
	@${GO} fmt ${GO_TARGETS}

.PHONY: fmt/check
fmt/check: ## check code formatting
	@${GO} fmt ${GO_TARGETS}
	@git diff --exit-code || (echo "code is not formatted, please run 'go fmt ./...'" && false)

.PHONY: precommit
precommit: tidy fmt lint test ## run all precommit checks
