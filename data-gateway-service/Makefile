# Go Data Gateway - Test-Driven Development Makefile
.PHONY: help test test-unit test-integration test-e2e test-coverage test-watch test-benchmark clean build run docker-build docker-run lint fmt vet

# Variables
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOFMT := gofmt
GOVET := $(GOCMD) vet
GOLINT := golangci-lint

# Coverage variables
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Test variables
TEST_TIMEOUT := 30s
INTEGRATION_TEST_TIMEOUT := 120s
UNIT_TEST_TAGS := unit
INTEGRATION_TEST_TAGS := integration
E2E_TEST_TAGS := e2e

# Color output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## test: Run all tests (unit + integration + e2e)
test:
	@echo "${YELLOW}Running all tests...${NC}"
	@$(GOTEST) -v -race -timeout $(INTEGRATION_TEST_TIMEOUT) ./...
	@echo "${GREEN}All tests passed!${NC}"

## test-unit: Run unit tests only
test-unit:
	@echo "${YELLOW}Running unit tests...${NC}"
	@$(GOTEST) -v -short -race -timeout $(TEST_TIMEOUT) ./internal/... ./cmd/...
	@echo "${GREEN}Unit tests passed!${NC}"

## test-integration: Run integration tests only
test-integration:
	@echo "${YELLOW}Running integration tests...${NC}"
	@$(GOTEST) -v -run Integration -race -timeout $(INTEGRATION_TEST_TIMEOUT) ./test/integration/...
	@echo "${GREEN}Integration tests passed!${NC}"

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "${YELLOW}Running E2E tests...${NC}"
	@$(GOTEST) -v -run E2E -race -timeout $(INTEGRATION_TEST_TIMEOUT) ./test/e2e/...
	@echo "${GREEN}E2E tests passed!${NC}"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "${YELLOW}Running tests with coverage...${NC}"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "${GREEN}Coverage report generated: $(COVERAGE_HTML)${NC}"
	@echo "Coverage summary:"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total Coverage: " $$3}'

## test-coverage-threshold: Check if coverage meets minimum threshold
test-coverage-threshold: test-coverage
	@echo "${YELLOW}Checking coverage threshold (70%)...${NC}"
	@coverage=$$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if (( $$(echo "$$coverage < 70" | bc -l) )); then \
		echo "${RED}Coverage $$coverage% is below threshold 70%${NC}"; \
		exit 1; \
	else \
		echo "${GREEN}Coverage $$coverage% meets threshold${NC}"; \
	fi

## test-watch: Run tests in watch mode (requires entr)
test-watch:
	@echo "${YELLOW}Starting test watch mode...${NC}"
	@echo "${YELLOW}Install entr first: apt-get install entr${NC}"
	@find . -name '*.go' | entr -c make test-unit

## test-benchmark: Run benchmark tests
test-benchmark:
	@echo "${YELLOW}Running benchmark tests...${NC}"
	@$(GOTEST) -bench=. -benchmem -run=^$ ./benchmark/...
	@echo "${GREEN}Benchmark tests completed!${NC}"

## test-specific: Run a specific test by name (use TEST_NAME=TestName)
test-specific:
	@echo "${YELLOW}Running test: $(TEST_NAME)${NC}"
	@$(GOTEST) -v -run $(TEST_NAME) ./...

## test-package: Test a specific package (use PKG=./internal/datasource)
test-package:
	@echo "${YELLOW}Testing package: $(PKG)${NC}"
	@$(GOTEST) -v -race $(PKG)

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "${YELLOW}Running tests (verbose)...${NC}"
	@$(GOTEST) -v -race ./... 2>&1 | tee test-output.log
	@echo "${GREEN}Test output saved to test-output.log${NC}"

## test-json: Run tests with JSON output (for CI/CD)
test-json:
	@$(GOTEST) -json ./... > test-results.json
	@echo "${GREEN}Test results saved to test-results.json${NC}"

## test-failfast: Stop on first test failure
test-failfast:
	@echo "${YELLOW}Running tests (fail fast)...${NC}"
	@$(GOTEST) -failfast -race ./...

## lint: Run linters
lint:
	@echo "${YELLOW}Running linters...${NC}"
	@$(GOLINT) run ./...
	@echo "${GREEN}Linting passed!${NC}"

## fmt: Format code
fmt:
	@echo "${YELLOW}Formatting code...${NC}"
	@$(GOFMT) -w -s .
	@echo "${GREEN}Code formatted!${NC}"

## vet: Run go vet
vet:
	@echo "${YELLOW}Running go vet...${NC}"
	@$(GOVET) ./...
	@echo "${GREEN}Vet passed!${NC}"

## clean: Clean test cache and coverage reports
clean:
	@echo "${YELLOW}Cleaning...${NC}"
	@$(GOCMD) clean -testcache
	@rm -rf $(COVERAGE_DIR)
	@rm -f test-output.log test-results.json
	@echo "${GREEN}Cleaned!${NC}"

## build: Build the application
build:
	@echo "${YELLOW}Building application...${NC}"
	@$(GOBUILD) -o bin/server-chi cmd/server/main_chi.go
	@echo "${GREEN}Build complete: bin/server-chi${NC}"

## run: Run the application
run:
	@echo "${YELLOW}Running application...${NC}"
	@$(GOCMD) run cmd/server/main_chi.go

## docker-build: Build Docker image
docker-build:
	@echo "${YELLOW}Building Docker image...${NC}"
	@docker build -t go-data-gateway .
	@echo "${GREEN}Docker image built!${NC}"

## docker-test: Run tests in Docker
docker-test:
	@echo "${YELLOW}Running tests in Docker...${NC}"
	@docker-compose -f docker-compose.test.yml up --abort-on-container-exit
	@docker-compose -f docker-compose.test.yml down
	@echo "${GREEN}Docker tests completed!${NC}"

## install-tools: Install required testing tools
install-tools:
	@echo "${YELLOW}Installing testing tools...${NC}"
	@$(GOGET) github.com/stretchr/testify/assert@latest
	@$(GOGET) github.com/stretchr/testify/mock@latest
	@$(GOGET) github.com/stretchr/testify/suite@latest
	@$(GOGET) github.com/golang/mock/mockgen@latest
	@$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "${GREEN}Testing tools installed!${NC}"

## generate-mocks: Generate mocks for interfaces
generate-mocks:
	@echo "${YELLOW}Generating mocks...${NC}"
	@mockgen -source=internal/datasource/interface.go -destination=internal/mocks/datasource_mock.go -package=mocks
	@mockgen -source=internal/cache/interface.go -destination=internal/mocks/cache_mock.go -package=mocks
	@echo "${GREEN}Mocks generated!${NC}"

## pre-commit: Run all checks before committing
pre-commit: fmt vet lint test-unit
	@echo "${GREEN}Pre-commit checks passed!${NC}"

## ci: Run CI pipeline locally
ci: clean fmt vet lint test-coverage-threshold
	@echo "${GREEN}CI pipeline passed!${NC}"

# Default target
.DEFAULT_GOAL := help